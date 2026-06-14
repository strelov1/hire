package sources

import "testing"

func TestFirstNonEmpty(t *testing.T) {
	cases := []struct {
		parts []string
		want  string
	}{
		{[]string{"a", "b"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", "", "c"}, "c"},
		{[]string{"  ", "b"}, "  "}, // exact-empty check: whitespace-only is NOT blank (drop-in for `== ""`)
		{[]string{"", ""}, ""},
		{nil, ""},
		{[]string{" verbatim "}, " verbatim "}, // returned verbatim
	}
	for _, c := range cases {
		if got := firstNonEmpty(c.parts...); got != c.want {
			t.Errorf("firstNonEmpty(%q) = %q, want %q", c.parts, got, c.want)
		}
	}
}
