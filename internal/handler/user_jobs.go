package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/auth"
	"github.com/strelov1/freehire/internal/db"
)

// interactionResponse is the public shape of a user's interaction with a job. It
// omits user_id (the caller is the user) and carries saved_at/applied_at as null
// until the job is saved/applied to.
type interactionResponse struct {
	JobID     int64              `json:"job_id"`
	ViewedAt  pgtype.Timestamptz `json:"viewed_at"`
	SavedAt   pgtype.Timestamptz `json:"saved_at"`
	AppliedAt pgtype.Timestamptz `json:"applied_at"`
}

func toInteraction(row db.UserJob) interactionResponse {
	return interactionResponse{JobID: row.JobID, ViewedAt: row.ViewedAt, SavedAt: row.SavedAt, AppliedAt: row.AppliedAt}
}

// RecordView records that the authenticated user viewed a job and returns the
// resulting interaction, including whether they have already applied.
func (h *Handler) RecordView(c *fiber.Ctx) error {
	userID, jobID, err := h.interactionParams(c)
	if err != nil {
		return err
	}

	row, err := h.queries.RecordJobView(c.Context(), db.RecordJobViewParams{UserID: userID, JobID: jobID})
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"data": toInteraction(row)})
}

// MarkApplied marks a job as applied for the authenticated user and returns the
// updated interaction.
func (h *Handler) MarkApplied(c *fiber.Ctx) error {
	userID, jobID, err := h.interactionParams(c)
	if err != nil {
		return err
	}

	row, err := h.queries.MarkJobApplied(c.Context(), db.MarkJobAppliedParams{UserID: userID, JobID: jobID})
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"data": toInteraction(row)})
}

// SaveJob saves (bookmarks) a job for the authenticated user and returns the
// updated interaction.
func (h *Handler) SaveJob(c *fiber.Ctx) error {
	userID, jobID, err := h.interactionParams(c)
	if err != nil {
		return err
	}

	row, err := h.queries.SaveJob(c.Context(), db.SaveJobParams{UserID: userID, JobID: jobID})
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"data": toInteraction(row)})
}

// UnsaveJob clears a job's saved mark for the authenticated user. The interaction
// row (view/apply history) survives; if no row exists at all, unsaving is a no-op
// that answers with the zero interaction state — DELETE is idempotent, so "already
// not saved" is success, not an error.
func (h *Handler) UnsaveJob(c *fiber.Ctx) error {
	userID, jobID, err := h.interactionParams(c)
	if err != nil {
		return err
	}

	row, err := h.queries.UnsaveJob(c.Context(), db.UnsaveJobParams{UserID: userID, JobID: jobID})
	if errors.Is(err, pgx.ErrNoRows) {
		return c.JSON(fiber.Map{"data": interactionResponse{JobID: jobID}})
	}
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"data": toInteraction(row)})
}

// interactionParams resolves the authenticated user id and the internal job id
// for the view/apply handlers. The job is addressed publicly by its :slug, which
// is resolved to the internal bigint id (the user_jobs FK) via GetJobIDBySlug — a
// slim id-only lookup, since this hot path never needs the wide job columns. The
// user id is always present behind RequireAuth; an unknown slug surfaces as
// pgx.ErrNoRows, which ErrorHandler maps to 404.
func (h *Handler) interactionParams(c *fiber.Ctx) (int64, int64, error) {
	userID, ok := auth.UserID(c)
	if !ok {
		return 0, 0, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	jobID, err := h.queries.GetJobIDBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return 0, 0, err
	}
	return userID, jobID, nil
}
