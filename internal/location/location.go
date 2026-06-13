// Package location derives a job's geography — ISO 3166-1 alpha-2 country codes
// and region codes — and a work-mode hint deterministically from the free-text
// ATS location string.
//
// It is a curated dictionary, not a geocoder: it resolves the high-frequency
// country names, ATS shorthands ("USA", "UK"), macro-region names ("Europe",
// "APAC"), and a few beacon cities that real ATS location fields use, and emits
// nothing for anything it cannot resolve (it never guesses). Region codes are
// drawn from the same controlled vocabulary the enrichment contract defines
// (enrich.RegionValues), and work modes from enrich.WorkModeValues, so the
// parser, the enrichment payload, and the search facet all speak one set of
// values.
package location

import (
	"sort"
	"strings"
)

// Geo is the geography parsed from a location string: zero or more country codes
// and region codes, and an optional work-mode hint. Each field is empty when the
// location states nothing the parser can resolve.
type Geo struct {
	Countries []string
	Regions   []string
	WorkMode  string // "", "remote", "hybrid", or "onsite" — only on an explicit marker
}

// separatorReplacer normalizes every token separator to a comma in one pass so a
// single Split yields the geography tokens. The multi-character forms (" - ",
// " or ") and parentheses are included, so "Berlin (On-site)" -> "berlin",
// "on-site".
var separatorReplacer = strings.NewReplacer(
	";", ",", "/", ",", "|", ",", "(", ",", ")", ",", " - ", ",", " or ", ",",
)

// Parse maps a location string to its geography. Countries/regions are
// deduplicated and sorted; nil when nothing resolves. WorkMode is set only from
// an explicit marker — a bare "Remote" yields WorkMode "remote" with no
// geography, while a plain city/country yields geography with no WorkMode. The
// "global" region is emitted only from an explicit open-anywhere marker, never
// inferred from a bare "Remote".
func Parse(location string) Geo {
	lower := strings.ToLower(location)

	s := separatorReplacer.Replace(lower)

	countrySet := map[string]struct{}{}
	regionSet := map[string]struct{}{}
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		tok = stripCityPrefix(tok)
		if code, ok := nameToCountry[tok]; ok {
			countrySet[code] = struct{}{}
			if r, ok := countryToRegion[code]; ok {
				regionSet[r] = struct{}{}
			}
			continue
		}
		if r, ok := nameToRegion[tok]; ok {
			regionSet[r] = struct{}{}
		}
	}

	return Geo{
		Countries: sortedKeys(countrySet),
		Regions:   sortedKeys(regionSet),
		WorkMode:  detectWorkMode(lower),
	}
}

// cityMarkerPrefixes are the Russian "city" abbreviations that RU-segment ATS
// data prepends to a bare city name ("г Москва", "город Самара"). Stripped from a
// token before lookup so the city resolves; checked longest-first so "город "
// wins over "г ". A city whose name merely starts with "г" ("Грозный") is
// untouched — every prefix ends in a separator the name doesn't.
var cityMarkerPrefixes = []string{"город ", "г. ", "г.", "г "}

// stripCityPrefix removes a leading Russian city marker from an already-lowercased,
// trimmed token, returning the bare city name (or the token unchanged).
func stripCityPrefix(tok string) string {
	for _, p := range cityMarkerPrefixes {
		if rest, ok := strings.CutPrefix(tok, p); ok {
			return strings.TrimSpace(rest)
		}
	}
	return tok
}

// workModeMarkers maps a work mode to the substrings that signal it, checked in
// priority order: hybrid (most specific) beats a remote marker in the same
// string, and an explicit onsite marker is the last resort. A location with no
// marker yields "" — onsite is never assumed from a bare city.
var workModeMarkers = []struct {
	mode    string
	markers []string
}{
	{"hybrid", []string{"hybrid", "гибрид"}},
	{"remote", []string{"remote", "work from home", "wfh", "anywhere", "worldwide", "distributed", "удал"}},
	{"onsite", []string{"on-site", "onsite", "on site", "in office", "in-office"}},
}

// detectWorkMode scans the whole lowercased location for a work-mode marker,
// independent of tokenization so a marker embedded in a token ("Berlin
// (On-site)") is still found.
func detectWorkMode(lower string) string {
	for _, wm := range workModeMarkers {
		for _, m := range wm.markers {
			if strings.Contains(lower, m) {
				return wm.mode
			}
		}
	}
	return ""
}

// sortedKeys returns the set's keys sorted ascending, or nil when empty so an
// absent facet omits cleanly (and matches the text[] default '{}').
func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
