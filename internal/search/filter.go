package search

import (
	"strconv"
	"strings"
)

// Filter expression helpers. These build Meilisearch filter fragments so the
// handler can express facet intent without knowing Meilisearch syntax, and so
// untrusted query-param values are always escaped at one place (see Eq).

// Eq builds an equality fragment `attr = "value"` with the value quoted and
// escaped. Quoting is mandatory: an unescaped value could otherwise inject
// filter logic (e.g. `senior" OR work_mode = "remote`).
func Eq(attr, value string) string {
	return attr + " = " + quote(value)
}

// Neq builds an inequality fragment `attr != "value"` (escaped), used by the
// exclude facets to filter a value out.
func Neq(attr, value string) string {
	return attr + " != " + quote(value)
}

// EqBool builds an equality fragment against a boolean attribute (unquoted, as
// Meilisearch compares booleans literally).
func EqBool(attr string, v bool) string {
	return attr + " = " + strconv.FormatBool(v)
}

// Gte builds a `attr >= n` numeric fragment.
func Gte(attr string, n int) string {
	return attr + " >= " + strconv.Itoa(n)
}

// Lte builds a `attr <= n` numeric fragment.
func Lte(attr string, n int) string {
	return attr + " <= " + strconv.Itoa(n)
}

// Filter nests OR-groups into a single AND filter for Meilisearch: fragments
// within a group are ORed, groups are ANDed. Empty groups are dropped; the
// result is nil when nothing remains, which Meilisearch treats as "no filter".
func Filter(groups ...[]string) any {
	out := make([][]string, 0, len(groups))
	for _, g := range groups {
		if len(g) > 0 {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// quote wraps a value in a Meilisearch string literal, backslash-escaping the
// double-quote and backslash characters.
func quote(value string) string {
	var b strings.Builder
	b.Grow(len(value) + 2)
	b.WriteByte('"')
	for _, r := range value {
		if r == '"' || r == '\\' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}
