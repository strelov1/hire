package search

import (
	"encoding/json"
	"testing"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/jobview"
)

func TestFromJob_CarriesIDAndView(t *testing.T) {
	doc, err := FromJob(db.Job{ID: 42, Title: "Go Dev", PublicSlug: "go-dev-acme-x"})
	if err != nil {
		t.Fatalf("FromJob: %v", err)
	}
	if doc.ID != 42 {
		t.Errorf("ID = %d, want 42", doc.ID)
	}
	if doc.PublicSlug != "go-dev-acme-x" || doc.Title != "Go Dev" {
		t.Errorf("view not mapped: %+v", doc.Job)
	}
}

func TestJobDocument_FlattensIDAndViewToTopLevelJSON(t *testing.T) {
	// Meilisearch reads the primary key "id" from the top level of the document,
	// and the embedded jobview.Job must flatten (no nesting) so its fields are
	// the searchable attributes. A json tag on the embedded field would break
	// this. Enrichment itself stays a nested object (filtered via dot paths).
	doc := JobDocument{ID: 7, Job: jobview.Job{PublicSlug: "x-7", Title: "x"}}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["id"]; !ok {
		t.Errorf("missing top-level id in %s", raw)
	}
	if _, ok := m["public_slug"]; !ok {
		t.Errorf("public_slug not flattened to top level in %s", raw)
	}
	if _, ok := m["enrichment"]; !ok {
		t.Errorf("enrichment should be a nested object in %s", raw)
	}
}
