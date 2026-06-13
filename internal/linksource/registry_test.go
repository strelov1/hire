package linksource

import (
	"net/url"
	"testing"
)

func TestFindMatchesAdapterByLinkHost(t *testing.T) {
	reg := All(nil) // Match inspects only the URL, so a nil client is fine here.

	cases := []struct {
		raw  string
		want string // expected adapter Source(), "" = no match
	}{
		{"https://u.habr.com/PnBO7", "habr_career"},
		{"https://career.habr.com/vacancies/1000166712", "habr_career"},
		{"https://remoteyeah.com/jobs/remote-senior-quality-engineer-peek", "remoteyeah"},
		{"https://geekjob.ru/vacancy/6a1ebb8520ad023342091661", "geekjob"},
		{"https://job-boards.greenhouse.io/alpaca/jobs/5745893004", "greenhouse"},
		{"https://boards.eu.greenhouse.io/acme/jobs/123", "greenhouse"},
		{"https://jobs.ashbyhq.com/ruby-labs/62661b07-ac6b-4283-ae38-6c3255c47bd4", "ashby"},
		{"https://jobs.ashbyhq.com/ruby-labs", ""},      // board page, not a /<board>/<id> link
		{"https://job-boards.greenhouse.io/alpaca", ""}, // board page, not a /jobs/<id> link
		{"https://geekjob.ru/", ""},                     // homepage, not a /vacancy/<id> link
		{"https://remoteyeah.com/", ""},                 // homepage, not a /jobs/<slug> link
		{"https://example.com/jobs/x", ""},              // unknown domain
		{"https://t.me/habr_career/75410", ""},          // the post itself, not an outbound link
	}

	for _, c := range cases {
		u, err := url.Parse(c.raw)
		if err != nil {
			t.Fatalf("parse %q: %v", c.raw, err)
		}
		ls := Find(reg, u)
		got := ""
		if ls != nil {
			got = ls.Source()
		}
		if got != c.want {
			t.Errorf("Find(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}
