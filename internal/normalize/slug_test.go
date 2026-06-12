package normalize

import "testing"

func TestSlug(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Yandex", "yandex"},
		{"lowercases", "GitHub", "github"},
		{"spaces to hyphen", "Yandex LLC", "yandex-llc"},
		{"collapses whitespace and punctuation", "  Acme,  Inc. ", "acme-inc"},
		{"keeps digits", "3M", "3m"},
		{"transliterates cyrillic to ascii", "Яндекс", "iandeks"},
		{"transliterates cjk to ascii", "小红书", "xiao-hong-shu"},
		{"strips accents", "Köln", "koln"},
		{"empty stays empty", "", ""},
		{"whitespace only stays empty", "   ", ""},
		{"punctuation only stays empty", "!!!", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Slug(tc.in); got != tc.want {
				t.Errorf("Slug(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
