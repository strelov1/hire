package handler

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/jobview"
	"github.com/strelov1/freehire/internal/search"
)

// searcher is the search backend the handler depends on. *search.Client
// satisfies it; tests inject a fake. A nil searcher means search is not
// configured (no MEILI_MASTER_KEY) and the endpoint reports 503.
type searcher interface {
	Search(ctx context.Context, p search.SearchParams) (search.SearchResult, error)
}

// defaultSemanticRatio blends keyword and semantic ranking by default (hybrid
// search is the headline feature). Clients can pass semantic_ratio=0 for pure
// keyword search.
const defaultSemanticRatio = 0.5

// searchStringFacets maps an equality-facet query param to its index attribute.
// Enrichment facets live under the nested "enrichment" object, so they filter on
// a dot path. Repeated params (?seniority=a&seniority=b) are ORed.
var searchStringFacets = map[string]string{
	"source":           "source",
	"company_slug":     "company_slug",
	"regions":          "enrichment.regions",
	"work_mode":        "enrichment.work_mode",
	"employment_type":  "enrichment.employment_type",
	"seniority":        "enrichment.seniority",
	"category":         "enrichment.category",
	"domains":          "enrichment.domains",
	"countries":        "enrichment.countries",
	"company_type":     "enrichment.company_type",
	"company_size":     "enrichment.company_size",
	"salary_currency":  "enrichment.salary_currency",
	"salary_period":    "enrichment.salary_period",
	"skills":           "enrichment.skills",
	"relocation":       "enrichment.relocation",
	"english_level":    "enrichment.english_level",
	"posting_language": "enrichment.posting_language",
}

// searchSortable is the allowlist of sort params mapped to their index attribute;
// anything else is ignored so a bad param cannot make Meilisearch reject the query.
var searchSortable = map[string]string{
	"created_at": "created_at",
	"posted_at":  "posted_at",
	"salary_min": "enrichment.salary_min",
	"salary_max": "enrichment.salary_max",
}

// SearchJobs runs a full-text + hybrid search over the jobs index. It is public
// (unauthenticated) like the other job reads. Response: {"data": [job view...],
// "meta": {total, limit, offset}} — results carry public_slug and never the
// internal id.
func (h *Handler) SearchJobs(c *fiber.Ctx) error {
	if h.search == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "search is not available")
	}

	limit, offset := pageParams(c)
	ratio := min(max(c.QueryFloat("semantic_ratio", defaultSemanticRatio), 0), 1)

	res, err := h.search.Search(c.Context(), search.SearchParams{
		Query:         c.Query("q"),
		Filter:        buildSearchFilter(c),
		Sort:          searchSort(c),
		Limit:         limit,
		Offset:        offset,
		SemanticRatio: ratio,
	})
	if err != nil {
		// ErrorHandler renders a generic 500; returning the error keeps the
		// Meilisearch failure cause visible to logging instead of swallowing it.
		return err
	}

	views := make([]jobview.Job, len(res.Hits))
	for i, hit := range res.Hits {
		views[i] = hit.Job
	}

	return listResponse(c, views, res.Total, limit, offset)
}

// searchSort builds the Meilisearch sort directive from ?sort=<field>&order=<dir>.
// Without a valid sort param, a no-text browse defaults to newest-added first
// (created_at desc) — relevance is meaningless for an empty query — while a text
// query keeps relevance order (nil).
func searchSort(c *fiber.Ctx) []string {
	attr, ok := searchSortable[c.Query("sort")]
	if !ok {
		if c.Query("q") == "" {
			return []string{"created_at:desc"}
		}
		return nil
	}
	order := c.Query("order", "desc")
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	return []string{attr + ":" + order}
}

// buildSearchFilter turns facet query params into a Meilisearch filter. Within a
// facet, included values are ORed by default (or ANDed when `<param>_mode=and`);
// excluded values (`<param>_exclude=...`) become NOT fragments. Facets are ANDed.
// Returns nil when no facet is set.
func buildSearchFilter(c *fiber.Ctx) any {
	var groups [][]string

	for param, attr := range searchStringFacets {
		if vals := queryValues(c, param); len(vals) > 0 {
			if c.Query(param+"_mode") == "and" {
				// Each value its own AND group: a job must match all of them.
				for _, v := range vals {
					groups = append(groups, []string{search.Eq(attr, v)})
				}
			} else {
				group := make([]string, len(vals))
				for i, v := range vals {
					group[i] = search.Eq(attr, v)
				}
				groups = append(groups, group)
			}
		}
		// Excluded values: each is its own AND group so all are filtered out.
		for _, v := range queryValues(c, param+"_exclude") {
			groups = append(groups, []string{search.Neq(attr, v)})
		}
	}

	if raw := c.Query("visa_sponsorship"); raw != "" {
		groups = append(groups, []string{search.EqBool("enrichment.visa_sponsorship", raw == "true")})
	}

	if n, ok := queryInt(c, "salary_min"); ok {
		groups = append(groups, []string{search.Gte("enrichment.salary_min", n)})
	}
	if n, ok := queryInt(c, "salary_max"); ok {
		groups = append(groups, []string{search.Lte("enrichment.salary_max", n)})
	}
	if n, ok := queryInt(c, "experience_years_min"); ok {
		groups = append(groups, []string{search.Gte("enrichment.experience_years_min", n)})
	}

	return search.Filter(groups...)
}

// queryValues returns all values of a (possibly repeated) query parameter.
func queryValues(c *fiber.Ctx, key string) []string {
	multi := c.Context().QueryArgs().PeekMulti(key)
	out := make([]string, 0, len(multi))
	for _, v := range multi {
		if s := string(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// queryInt reads an integer query param, reporting whether a valid number was
// given. A missing or non-numeric value reports false so no bogus `>= 0` filter
// fragment is emitted (Fiber's QueryInt would silently return 0 on parse error).
func queryInt(c *fiber.Ctx, key string) (int, bool) {
	n, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return 0, false
	}
	return n, true
}
