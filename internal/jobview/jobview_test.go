package jobview

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
)

func ptr[T any](v T) *T { return &v }

func TestFromRow_MapsCoreAndNestedEnrichment(t *testing.T) {
	posted := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	raw, err := json.Marshal(enrich.Enrichment{
		Seniority:       "senior",
		Category:        "backend",
		Domains:         []string{"fintech"},
		VisaSponsorship: ptr(true),
		SalaryMin:       ptr(100000),
		Skills:          []string{"go", "postgres"},
	})
	if err != nil {
		t.Fatalf("marshal enrichment: %v", err)
	}

	view, err := FromRow(db.Job{
		ID:          42,
		Source:      "manual",
		ExternalID:  "ext-1",
		Title:       "Senior Go Developer",
		Company:     "Acme",
		CompanySlug: "acme",
		Location:    "Berlin",
		Remote:      true,
		Description: "Build durable systems",
		PostedAt:    pgtype.Timestamptz{Time: posted, Valid: true},
		PublicSlug:  "senior-go-developer-acme-abcd1234",
		Enrichment:  raw,
	})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}

	if view.PublicSlug != "senior-go-developer-acme-abcd1234" {
		t.Errorf("PublicSlug = %q", view.PublicSlug)
	}
	if view.Title != "Senior Go Developer" || view.Company != "Acme" || view.Source != "manual" {
		t.Errorf("core fields not mapped: %+v", view)
	}
	if view.PostedAt == nil || *view.PostedAt != "2025-01-02T03:04:05Z" {
		t.Errorf("PostedAt = %v, want RFC3339 UTC", view.PostedAt)
	}
	// Enrichment stays nested and typed.
	if view.Enrichment.Seniority != "senior" || view.Enrichment.Category != "backend" {
		t.Errorf("nested enrichment not mapped: %+v", view.Enrichment)
	}
	if view.Enrichment.SalaryMin == nil || *view.Enrichment.SalaryMin != 100000 {
		t.Errorf("nested salary_min = %v", view.Enrichment.SalaryMin)
	}
	if len(view.Enrichment.Skills) != 2 || view.Enrichment.VisaSponsorship == nil || !*view.Enrichment.VisaSponsorship {
		t.Errorf("nested skills/visa not mapped: %+v", view.Enrichment)
	}
}

func TestFromRow_EmptyEnrichmentIsZero(t *testing.T) {
	// An unenriched job's column arrives as "{}" (the table default) or, in
	// edge cases, a nil byte slice. Both must decode to the zero Enrichment,
	// never fail.
	for _, payload := range [][]byte{[]byte("{}"), nil} {
		view, err := FromRow(db.Job{ID: 1, Title: "x", PublicSlug: "x-1", Enrichment: payload})
		if err != nil {
			t.Fatalf("FromRow with enrichment %q: %v", payload, err)
		}
		if view.Enrichment.Seniority != "" || view.Enrichment.SalaryMin != nil || len(view.Enrichment.Skills) != 0 {
			t.Errorf("enrichment %q: expected zero enrichment, got %+v", payload, view.Enrichment)
		}
	}
}

// Job's JSON encoding IS the API contract for every jobs read endpoint. These
// tests lock two requirements: the internal numeric id is never exposed and the
// public slug is (specs/job-public-identity), and the enrichment payload
// survives the mapping (specs/job-enrichment): an unenriched job serializes
// enrichment as {} (not null), and an enriched payload keeps its fields.

func TestJobJSON_HidesIDExposesSlug(t *testing.T) {
	fields := marshalToFields(t, db.Job{
		ID:         123,
		Title:      "Go Developer",
		PublicSlug: "go-developer-acme-t35nijto",
	})

	if _, leaked := fields["id"]; leaked {
		t.Error("wire shape leaks the internal numeric id")
	}
	if got := string(fields["public_slug"]); got != `"go-developer-acme-t35nijto"` {
		t.Errorf("public_slug: want the slug, got %s", got)
	}
}

// The raw remote flag is demoted to an internal enrichment hint and must not
// appear in the public job object — "remote" is expressed solely through
// enrichment.work_mode / regions.
func TestJobJSON_OmitsRawRemoteFlag(t *testing.T) {
	fields := marshalToFields(t, db.Job{ID: 1, Title: "x", PublicSlug: "x-1", Remote: true})

	if _, present := fields["remote"]; present {
		t.Error("public job object must not include the raw remote flag")
	}
}

// Un-enriched job: enrichment is {} (not null), enriched_at is null,
// enrichment_version is 0.
func TestJobJSON_Unenriched(t *testing.T) {
	fields := marshalToFields(t, db.Job{ID: 1, Title: "Go Developer"})

	if got := string(fields["enrichment"]); got != "{}" {
		t.Errorf("enrichment: want {}, got %s", got)
	}
	if got := string(fields["posted_at"]); got != "null" {
		t.Errorf("posted_at: want null for an unset timestamp, got %s", got)
	}
	if got := string(fields["enriched_at"]); got != "null" {
		t.Errorf("enriched_at: want null, got %s", got)
	}
	if got := string(fields["enrichment_version"]); got != "0" {
		t.Errorf("enrichment_version: want 0, got %s", got)
	}
}

// Enriched job: the JSONB payload survives the typed decode/encode round-trip,
// enriched_at is the RFC3339 UTC timestamp, version is set.
func TestJobJSON_Enriched(t *testing.T) {
	enrichedAt := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	fields := marshalToFields(t, db.Job{
		ID:                2,
		Title:             "Senior Go Developer",
		Enrichment:        json.RawMessage(`{"seniority":"senior","work_mode":"remote"}`),
		EnrichedAt:        pgtype.Timestamptz{Time: enrichedAt, Valid: true},
		EnrichmentVersion: 1,
	})

	var enrichment map[string]any
	if err := json.Unmarshal(fields["enrichment"], &enrichment); err != nil {
		t.Fatalf("enrichment is not a JSON object: %v", err)
	}
	if enrichment["seniority"] != "senior" || enrichment["work_mode"] != "remote" {
		t.Errorf("enrichment payload not preserved: %v", enrichment)
	}
	if got := string(fields["enriched_at"]); got != `"2026-06-09T12:00:00Z"` {
		t.Errorf("enriched_at: want the timestamp, got %s", got)
	}
	if got := string(fields["enrichment_version"]); got != "1" {
		t.Errorf("enrichment_version: want 1, got %s", got)
	}
}

// marshalToFields maps a db.Job through the wire shape and returns its
// top-level JSON fields — the actual public contract.
func marshalToFields(t *testing.T, job db.Job) map[string]json.RawMessage {
	t.Helper()
	view, err := FromRow(job)
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return fields
}
