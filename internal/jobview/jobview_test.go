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

func TestFromRow_UnenrichedHasZeroEnrichment(t *testing.T) {
	view, err := FromRow(db.Job{
		ID:         1,
		Title:      "Go Developer",
		Company:    "Acme",
		PublicSlug: "go-developer-acme-x",
		Enrichment: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}

	if view.Title != "Go Developer" {
		t.Errorf("Title = %q, want mapped", view.Title)
	}
	if view.Enrichment.Seniority != "" || view.Enrichment.SalaryMin != nil || len(view.Enrichment.Skills) != 0 {
		t.Errorf("expected zero enrichment, got %+v", view.Enrichment)
	}
	if view.PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil for an unset timestamp", view.PostedAt)
	}
}

func TestFromRow_NilEnrichmentByteSliceIsSafe(t *testing.T) {
	// A job whose enrichment column round-trips as a nil/empty byte slice must
	// not fail decoding — it is simply unenriched.
	view, err := FromRow(db.Job{ID: 2, Title: "x", PublicSlug: "x-2", Enrichment: nil})
	if err != nil {
		t.Fatalf("FromRow with nil enrichment: %v", err)
	}
	if view.Enrichment.Seniority != "" {
		t.Errorf("expected no enrichment, got seniority %q", view.Enrichment.Seniority)
	}
}

func TestJob_NeverSerializesInternalID(t *testing.T) {
	// The wire shape must not leak the internal numeric id — it is enumerable
	// and its growth leaks inventory size.
	view, err := FromRow(db.Job{ID: 99, Title: "x", PublicSlug: "x-99"})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, leaked := m["id"]; leaked {
		t.Errorf("internal id leaked in wire shape: %s", raw)
	}
}
