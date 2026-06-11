// Package jobview defines the single public wire shape of a job — the JSON
// representation served by the list, detail, and search endpoints and stored in
// the search index. Keeping one type (instead of parallel per-endpoint structs)
// makes drift between the API surfaces impossible.
package jobview

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/enrich"
)

// Job is the public wire shape of a job. It carries the public_slug and
// deliberately omits the internal numeric id, which must never be exposed: the
// id is enumerable and its growth leaks inventory size and fill rate.
//
// Enrichment is nested (not flattened) and typed; an unenriched job serializes
// it as `{}`. Timestamps are RFC3339 UTC strings (or null) — the lexicographic
// order is chronological, which the search index relies on for sorting.
type Job struct {
	PublicSlug        string            `json:"public_slug"`
	Source            string            `json:"source"`
	ExternalID        string            `json:"external_id"`
	URL               string            `json:"url"`
	Title             string            `json:"title"`
	Company           string            `json:"company"`
	CompanySlug       string            `json:"company_slug"`
	Location          string            `json:"location"`
	Remote            bool              `json:"remote"`
	Description       string            `json:"description"`
	PostedAt          *string           `json:"posted_at"`
	CreatedAt         *string           `json:"created_at"`
	UpdatedAt         *string           `json:"updated_at"`
	Enrichment        enrich.Enrichment `json:"enrichment"`
	EnrichedAt        *string           `json:"enriched_at"`
	EnrichmentVersion int32             `json:"enrichment_version"`
}

// FromRow maps a database job row to the public wire shape. The enrichment
// JSONB is decoded into the typed Enrichment; an empty or absent payload yields
// the zero Enrichment.
func FromRow(j db.Job) (Job, error) {
	var e enrich.Enrichment
	if len(j.Enrichment) > 0 {
		if err := json.Unmarshal(j.Enrichment, &e); err != nil {
			return Job{}, fmt.Errorf("jobview: decode enrichment for job %d: %w", j.ID, err)
		}
	}

	return Job{
		PublicSlug:        j.PublicSlug,
		Source:            j.Source,
		ExternalID:        j.ExternalID,
		URL:               j.URL,
		Title:             j.Title,
		Company:           j.Company,
		CompanySlug:       j.CompanySlug,
		Location:          j.Location,
		Remote:            j.Remote,
		Description:       j.Description,
		PostedAt:          rfc3339(j.PostedAt),
		CreatedAt:         rfc3339(j.CreatedAt),
		UpdatedAt:         rfc3339(j.UpdatedAt),
		Enrichment:        e,
		EnrichedAt:        rfc3339(j.EnrichedAt),
		EnrichmentVersion: j.EnrichmentVersion,
	}, nil
}

// FromRows maps a batch of database rows to the public wire shape.
func FromRows(jobs []db.Job) ([]Job, error) {
	out := make([]Job, len(jobs))
	for i, j := range jobs {
		v, err := FromRow(j)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// rfc3339 renders a nullable Postgres timestamp as an RFC3339 UTC string, or nil
// when unset. UTC keeps the lexicographic order chronological for sorting.
func rfc3339(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := ts.Time.UTC().Format(time.RFC3339)
	return &s
}
