package linksource

import (
	"context"
	"strings"
	"testing"
	"time"
)

// greenhouseJobJSON mirrors the public per-job boards API: content is HTML-escaped (the
// adapter unescapes then sanitizes), and company_name gives a real company.
const greenhouseJobJSON = `{
 "id": 5745893004,
 "title": "Sales Engineer - MENA",
 "absolute_url": "https://job-boards.greenhouse.io/alpaca/jobs/5745893004",
 "updated_at": "2026-04-15T12:28:38-04:00",
 "company_name": "Alpaca",
 "location": {"name": "Remote - MENA"},
 "content": "&lt;p&gt;Build it.&lt;/p&gt;&lt;script&gt;evil()&lt;/script&gt;"
}`

func TestGreenhouseResolvesAlignedIdentity(t *testing.T) {
	const link = "https://job-boards.greenhouse.io/alpaca/jobs/5745893004?utm_source=telegram"
	c := (&fakeClient{}).route("boards-api.greenhouse.io/v1/boards/alpaca/jobs/5745893004", greenhouseJobJSON, "")

	job, ok, err := NewGreenhouse(c).Resolve(context.Background(), link)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !ok {
		t.Fatal("ok=false, want the vacancy resolved")
	}
	// External id is namespaced by board, matching what the ingest pipeline writes, so the
	// same vacancy crawled directly dedups on (source=greenhouse, external_id) instead of
	// duplicating.
	if job.ExternalID != "alpaca:5745893004" {
		t.Errorf("ExternalID = %q, want alpaca:5745893004", job.ExternalID)
	}
	if job.URL != "https://job-boards.greenhouse.io/alpaca/jobs/5745893004" {
		t.Errorf("URL = %q", job.URL)
	}
	if job.Title != "Sales Engineer - MENA" {
		t.Errorf("Title = %q", job.Title)
	}
	if job.Company != "Alpaca" {
		t.Errorf("Company = %q, want Alpaca", job.Company)
	}
	if !job.Remote {
		t.Error("Remote = false, want true (Remote - MENA)")
	}
	if strings.Contains(job.Description, "<script>") || !strings.Contains(job.Description, "Build it.") {
		t.Errorf("Description not unescaped/sanitized: %q", job.Description)
	}
	if job.PostedAt == nil || !job.PostedAt.Equal(time.Date(2026, 4, 15, 16, 28, 38, 0, time.UTC)) {
		t.Errorf("PostedAt = %v, want 2026-04-15T16:28:38Z", job.PostedAt)
	}
}

func TestGreenhouseSourceKeyMatchesIngestProvider(t *testing.T) {
	// Identity alignment hinges on the source key being exactly the ingest provider.
	if got := NewGreenhouse(nil).Source(); got != "greenhouse" {
		t.Errorf("Source() = %q, want greenhouse", got)
	}
}
