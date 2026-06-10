package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/hire/internal/db"
)

// ListJobs returns a page of jobs using limit/offset pagination.
func (h *Handler) ListJobs(c *fiber.Ctx) error {
	limit, offset := pageParams(c)

	jobs, err := h.queries.ListJobs(c.Context(), db.ListJobsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list jobs")
	}

	total, err := h.queries.CountJobs(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count jobs")
	}

	return c.JSON(fiber.Map{
		"data": jobs,
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetJob returns a single job by id.
func (h *Handler) GetJob(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid job id")
	}

	job, err := h.queries.GetJob(c.Context(), id)
	if err != nil {
		// ErrorHandler maps pgx.ErrNoRows to 404, anything else to 500.
		return err
	}

	return c.JSON(fiber.Map{"data": job})
}
