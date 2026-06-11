package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/jobview"
	"github.com/strelov1/freehire/internal/search"
)

type fakeSearcher struct {
	got search.SearchParams
	res search.SearchResult
	err error
}

func (f *fakeSearcher) Search(_ context.Context, p search.SearchParams) (search.SearchResult, error) {
	f.got = p
	return f.res, f.err
}

func searchApp(s searcher) *fiber.App {
	h := &Handler{search: s}
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Get("/jobs/search", h.SearchJobs)
	return app
}

func doGet(t *testing.T, app *fiber.App, target string) (int, map[string]any) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, target, nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

func TestSearchJobs_DisabledReturns503(t *testing.T) {
	app := searchApp(nil) // search not configured
	status, _ := doGet(t, app, "/jobs/search?q=go")
	if status != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", status)
	}
}

func TestSearchJobs_PassesParamsAndShapesResponse(t *testing.T) {
	fake := &fakeSearcher{res: search.SearchResult{
		Hits:  []search.JobDocument{{ID: 1, Job: jobview.Job{PublicSlug: "go-dev-acme-x", Title: "Go Dev"}}},
		Total: 5,
	}}
	app := searchApp(fake)

	status, body := doGet(t, app, "/jobs/search?q=golang&limit=10&offset=20&seniority=senior&remote=true&semantic_ratio=0.3")
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	// Params mapped onto SearchParams.
	if fake.got.Query != "golang" {
		t.Errorf("Query = %q, want golang", fake.got.Query)
	}
	if fake.got.Limit != 10 || fake.got.Offset != 20 {
		t.Errorf("limit/offset = %d/%d, want 10/20", fake.got.Limit, fake.got.Offset)
	}
	if fake.got.SemanticRatio != 0.3 {
		t.Errorf("SemanticRatio = %v, want 0.3", fake.got.SemanticRatio)
	}
	groups, ok := fake.got.Filter.([][]string)
	if !ok {
		t.Fatalf("Filter = %#v, want [][]string", fake.got.Filter)
	}
	if !filterHas(groups, `enrichment.seniority = "senior"`) || !filterHas(groups, "remote = true") {
		t.Errorf("Filter missing facets: %#v", groups)
	}

	// Response envelope: data carries the public view (no id), meta has totals.
	data, _ := body["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("data len = %d, want 1", len(data))
	}
	first, _ := data[0].(map[string]any)
	if first["public_slug"] != "go-dev-acme-x" {
		t.Errorf("public_slug = %v", first["public_slug"])
	}
	if _, leaked := first["id"]; leaked {
		t.Errorf("internal id leaked in search result: %v", first)
	}
	meta, _ := body["meta"].(map[string]any)
	if meta["total"].(float64) != 5 || meta["limit"].(float64) != 10 || meta["offset"].(float64) != 20 {
		t.Errorf("meta = %v", meta)
	}
}

func TestSearchJobs_DefaultsToHybridSemanticRatio(t *testing.T) {
	fake := &fakeSearcher{}
	app := searchApp(fake)
	doGet(t, app, "/jobs/search?q=go")
	if fake.got.SemanticRatio != 0.5 {
		t.Errorf("default SemanticRatio = %v, want 0.5", fake.got.SemanticRatio)
	}
}

func TestSearchJobs_BackendErrorMaps500(t *testing.T) {
	fake := &fakeSearcher{err: context.DeadlineExceeded}
	app := searchApp(fake)
	status, _ := doGet(t, app, "/jobs/search?q=go")
	if status != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
}

func filterHas(groups [][]string, expr string) bool {
	for _, g := range groups {
		for _, e := range g {
			if e == expr {
				return true
			}
		}
	}
	return false
}
