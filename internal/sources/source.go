// Package sources holds the modular job-source adapters and the registry that maps
// a platform key to its adapter. Each adapter implements one ATS platform; adding a
// platform is a new file plus one line in All.
package sources

import (
	"context"
	"strings"
	"sync"
	"time"
)

// CompanyEntry is one configured board from sources.yml: the company whose jobs we
// crawl, the platform it uses (Provider), and the platform-specific board id.
type CompanyEntry struct {
	Company  string `yaml:"company"`
	Provider string `yaml:"provider"`
	Board    string `yaml:"board"`
}

// Job is a raw posting as an adapter yields it, before the pipeline normalizes it
// into the catalogue. ExternalID carries the platform's native posting id; the
// pipeline namespaces it by board before persisting.
type Job struct {
	ExternalID  string
	URL         string
	Title       string
	Company     string
	Location    string
	Description string
	Remote      bool
	PostedAt    *time.Time
}

// Source adapts one job-source platform. Provider is the platform key that selects
// the adapter (it matches CompanyEntry.Provider and the stored jobs.source); Fetch
// returns all current postings for one configured board.
type Source interface {
	Provider() string
	Fetch(ctx context.Context, e CompanyEntry) ([]Job, error)
}

// All assembles the registered adapters into a provider-keyed registry, sharing one
// HTTP client across them. Adding a platform is a new adapter plus one line here.
func All(c HTTPClient) map[string]Source {
	return reg(
		NewGreenhouse(c),
		NewLever(c),
		NewAshby(c),
		NewWorkable(c),
		NewRecruitee(c),
		NewSmartRecruiters(c),
		NewPersonio(c),
		NewPinpoint(c),
		NewRippling(c),
		NewBambooHR(c),
		NewWorkday(c),
	)
}

// fetchDetails maps each posting to a Job via fetch, running fetch concurrently with a
// bounded worker pool of the given size. A posting whose fetch returns ok=false is
// dropped, so one failed detail request never aborts the board. The surviving jobs keep
// their postings' relative order. Adapters whose list endpoint omits the description
// (SmartRecruiters, Rippling, BambooHR) share this so the bound and isolation behave
// identically across platforms.
func fetchDetails[P any](postings []P, workers int, fetch func(P) (Job, bool)) []Job {
	jobs := make([]*Job, len(postings))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, p := range postings {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, p P) {
			defer wg.Done()
			defer func() { <-sem }()
			if j, ok := fetch(p); ok {
				jobs[i] = &j
			}
		}(i, p)
	}
	wg.Wait()

	out := make([]Job, 0, len(jobs))
	for _, j := range jobs {
		if j != nil { // nil = detail fetch failed, skipped
			out = append(out, *j)
		}
	}
	return out
}

// isRemote infers a job's remote flag from its location text. Adapters share it so
// the heuristic stays consistent across platforms.
func isRemote(location string) bool {
	return strings.Contains(strings.ToLower(location), "remote")
}

// parseLayout parses a platform timestamp with the given layout into a posted_at,
// returning nil on an empty or unparseable value (posted_at is nullable — a missing or
// malformed date is not an error).
func parseLayout(layout, s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(layout, s)
	if err != nil {
		return nil
	}
	return &t
}

// parseRFC3339 parses an RFC3339 timestamp (the common ATS format).
func parseRFC3339(s string) *time.Time { return parseLayout(time.RFC3339, s) }

// parseDate parses a date-only timestamp ("2006-01-02", as Workable emits).
func parseDate(s string) *time.Time { return parseLayout("2006-01-02", s) }

// parseSpaceTime parses a space-separated, zone-named timestamp ("2006-01-02 15:04:05
// MST", as Recruitee emits). Recruitee emits UTC; an unrecognized zone abbreviation
// would be read as offset 0, acceptable for an approximate posted_at.
func parseSpaceTime(s string) *time.Time { return parseLayout("2006-01-02 15:04:05 MST", s) }

// joinNonEmpty joins the non-empty parts with ", ", so a location built from
// separate city/state/country fields skips blanks.
func joinNonEmpty(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, ", ")
}

// parseEpochMillis converts a Unix-millisecond timestamp into a posted_at, returning
// nil for a zero value (treated as "no date").
func parseEpochMillis(ms int64) *time.Time {
	if ms == 0 {
		return nil
	}
	t := time.UnixMilli(ms).UTC()
	return &t
}

// reg indexes sources by provider key. A duplicate key means two adapters claim the
// same platform — a programming error — so it panics rather than silently dropping one.
func reg(sources ...Source) map[string]Source {
	m := make(map[string]Source, len(sources))
	for _, s := range sources {
		if _, dup := m[s.Provider()]; dup {
			panic("sources: duplicate provider " + s.Provider())
		}
		m[s.Provider()] = s
	}
	return m
}
