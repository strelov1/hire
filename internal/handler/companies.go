package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/jobview"
)

// companyDetailResponse is the public shape of a company together with a page of
// its jobs. Its Jobs field is []jobview.Job, not []db.Job, so the internal job
// id cannot leak through this endpoint — the type enforces the DTO mapping.
type companyDetailResponse struct {
	Company db.Company    `json:"company"`
	Jobs    []jobview.Job `json:"jobs"`
}

// ListCompanies returns a page of companies with their job counts. Counts are
// computed at query time; there is no denormalized counter yet. An optional `q`
// query param filters companies by a case-insensitive name substring; an empty
// or absent `q` returns the full list. meta.total reports the count matching the
// filter so pagination over a search is correct.
func (h *Handler) ListCompanies(c *fiber.Ctx) error {
	limit, offset := pageParams(c)
	search := c.Query("q")

	companies, err := h.queries.ListCompanies(c.Context(), db.ListCompaniesParams{
		Search: search,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return err
	}

	total, err := h.queries.CountCompanies(c.Context(), search)
	if err != nil {
		return err
	}

	return listResponse(c, companies, total, limit, offset)
}

// GetCompany returns a single company together with a page of its jobs. The
// company is read from companies and its jobs from a single-table filter on
// company_slug — no join between the two tables.
func (h *Handler) GetCompany(c *fiber.Ctx) error {
	slug := c.Params("slug")

	company, err := h.queries.GetCompany(c.Context(), slug)
	if err != nil {
		// ErrorHandler maps pgx.ErrNoRows to 404, anything else to 500.
		return err
	}

	limit, offset := pageParams(c)

	jobs, err := h.queries.ListJobsByCompany(c.Context(), db.ListJobsByCompanyParams{
		CompanySlug: slug,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return err
	}

	views, err := jobview.FromRows(jobs)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"data": companyDetailResponse{Company: company, Jobs: views}})
}
