package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/jobview"
)

// The jobs read endpoints return a jobview.Job (mapped from db.Job) via c.JSON,
// so its JSON encoding IS the API contract. These tests lock two requirements:
// the internal numeric id is never exposed and the public slug is
// (specs/job-public-identity), and the enrichment payload survives the mapping
// (specs/job-enrichment): an unenriched job serializes enrichment as {} (not
// null), and an enriched payload keeps its fields.

// The internal bigint id must not leak; the public slug must be present.
func TestJobResponseHidesIDExposesSlug(t *testing.T) {
	job := db.Job{
		ID:         123,
		Title:      "Go Developer",
		PublicSlug: "go-developer-acme-t35nijto",
		Enrichment: json.RawMessage("{}"),
	}

	fields := marshalToFields(t, job)

	if _, ok := fields["id"]; ok {
		t.Error("response leaks the internal numeric id")
	}
	if got := string(fields["public_slug"]); got != `"go-developer-acme-t35nijto"` {
		t.Errorf("public_slug: want the slug, got %s", got)
	}
}

// Un-enriched job: enrichment is {} (not null), enriched_at is null,
// enrichment_version is 0.
func TestUnenrichedJobSerialization(t *testing.T) {
	job := db.Job{
		ID:         1,
		Title:      "Go Developer",
		Enrichment: json.RawMessage("{}"), // as scanned from NOT NULL DEFAULT '{}'
	}

	fields := marshalToFields(t, job)

	if got := string(fields["enrichment"]); got != "{}" {
		t.Errorf("enrichment: want {}, got %s", got)
	}
	if got := string(fields["enriched_at"]); got != "null" {
		t.Errorf("enriched_at: want null, got %s", got)
	}
	if got := string(fields["enrichment_version"]); got != "0" {
		t.Errorf("enrichment_version: want 0, got %s", got)
	}
}

// Enriched job: the JSONB payload survives the typed decode/encode round-trip,
// enriched_at is the timestamp, version is set.
func TestEnrichedJobSerialization(t *testing.T) {
	var enrichedAt pgtype.Timestamptz
	if err := enrichedAt.Scan(time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("scan timestamp: %v", err)
	}

	job := db.Job{
		ID:                2,
		Title:             "Senior Go Developer",
		Enrichment:        json.RawMessage(`{"seniority":"senior","work_mode":"remote"}`),
		EnrichedAt:        enrichedAt,
		EnrichmentVersion: 1,
	}

	fields := marshalToFields(t, job)

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

// marshalToFields maps a db.Job through the wire DTO and returns its top-level
// JSON fields — the actual public contract for the jobs endpoints.
func marshalToFields(t *testing.T, job db.Job) map[string]json.RawMessage {
	t.Helper()
	view, err := jobview.FromRow(job)
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
