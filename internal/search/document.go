package search

import (
	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/jobview"
)

// JobDocument is a job as stored in the Meilisearch index: the internal id (the
// primary key) plus the public jobview.Job — the exact wire shape served by the
// list and detail endpoints, so search hits render with the same SPA components.
// The embedded view flattens into the document JSON, so the stored document is
// `{ "id": ..., "public_slug": ..., ... }` and Meilisearch reads "id" as the
// primary key. The id is never returned to clients — handlers respond with the
// embedded view alone. Meilisearch filters/sorts on the nested enrichment facets
// via dot paths (e.g. "enrichment.seniority", "enrichment.salary_min").
type JobDocument struct {
	ID int64 `json:"id"`
	jobview.Job
}

// FromJob maps a database job row to its index document. An empty or absent
// enrichment payload yields the zero Enrichment (the job is still fully
// searchable by its text).
func FromJob(j db.Job) (JobDocument, error) {
	view, err := jobview.FromRow(j)
	if err != nil {
		return JobDocument{}, err
	}
	return JobDocument{ID: j.ID, Job: view}, nil
}
