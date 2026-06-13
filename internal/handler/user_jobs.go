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
// omits user_id (the caller is the user) and carries saved_at/applied_at/stage/
// notes as null until the job is saved, applied to, or tracked.
type interactionResponse struct {
	JobID     int64              `json:"job_id"`
	ViewedAt  pgtype.Timestamptz `json:"viewed_at"`
	SavedAt   pgtype.Timestamptz `json:"saved_at"`
	AppliedAt pgtype.Timestamptz `json:"applied_at"`
	Stage     pgtype.Text        `json:"stage"`
	Notes     pgtype.Text        `json:"notes"`
}

func toInteraction(row db.UserJob) interactionResponse {
	return interactionResponse{
		JobID:     row.JobID,
		ViewedAt:  row.ViewedAt,
		SavedAt:   row.SavedAt,
		AppliedAt: row.AppliedAt,
		Stage:     row.Stage,
		Notes:     row.Notes,
	}
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

// validStages is the controlled application-stage vocabulary and the source of
// truth: the track endpoint rejects any value not in this set (the SPA mirrors
// it). Active pipeline then terminal states.
var validStages = map[string]bool{
	"applied": true, "screening": true, "responded": true, "interview": true,
	"offer": true, "accepted": true, "rejected": true, "withdrawn": true,
}

func isValidStage(s string) bool { return validStages[s] }

// trackRequest is the track body: an optional stage and/or notes. A nil field is
// left unchanged by the upsert; at least one must be present.
type trackRequest struct {
	Stage *string `json:"stage"`
	Notes *string `json:"notes"`
}

// TrackJob sets the application stage and/or notes for the authenticated user's
// interaction with a job (session cookie or API key). The body is validated
// before the slug lookup, so a bad request never touches the DB: an empty body or
// an unknown stage is a 400. A nil field is left unchanged by the upsert. Returns
// the updated interaction.
func (h *Handler) TrackJob(c *fiber.Ctx) error {
	userID, ok := auth.UserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var in trackRequest
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if in.Stage == nil && in.Notes == nil {
		return fiber.NewError(fiber.StatusBadRequest, "provide stage and/or notes")
	}
	if in.Stage != nil && !isValidStage(*in.Stage) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid stage")
	}

	jobID, err := h.queries.GetJobIDBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return err
	}

	params := db.TrackJobParams{UserID: userID, JobID: jobID}
	if in.Stage != nil {
		params.Stage = pgtype.Text{String: *in.Stage, Valid: true}
	}
	if in.Notes != nil {
		params.Notes = pgtype.Text{String: *in.Notes, Valid: true}
	}
	row, err := h.queries.TrackJob(c.Context(), params)
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
