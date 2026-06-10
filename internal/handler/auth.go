package handler

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/hire/internal/auth"
	"github.com/strelov1/hire/internal/db"
)

const minPasswordLen = 8

// userResponse is the public shape of a user. It deliberately omits
// password_hash so the hash never reaches a response (db.User carries it).
type userResponse struct {
	ID        int64              `json:"id"`
	Email     string             `json:"email"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates an account, starts a session (auth cookie), and returns the
// user.
func (h *Handler) Register(c *fiber.Ctx) error {
	var in credentials
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	email, err := normalizeEmail(in.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid email")
	}
	if len(in.Password) < minPasswordLen {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	row, err := h.queries.CreateUser(c.Context(), db.CreateUserParams{
		Email:        email,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	})
	if isUniqueViolation(err) {
		return fiber.NewError(fiber.StatusConflict, "email already registered")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create user")
	}

	if err := h.setSession(c, row.ID); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": userResponse{ID: row.ID, Email: row.Email, CreatedAt: row.CreatedAt},
	})
}

// Login verifies credentials, starts a session (auth cookie), and returns the
// user. Unknown email, wrong password, and passwordless accounts all yield the
// same generic 401 so the response never reveals which factor failed.
func (h *Handler) Login(c *fiber.Ctx) error {
	var in credentials
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Normalize through the same seam as Register so the Go layer is the single
	// normalizer; a malformed email simply has no account (generic 401).
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	user, err := h.queries.GetUserByEmail(c.Context(), email)
	if err != nil || !user.PasswordHash.Valid || auth.CheckPassword(user.PasswordHash.String, in.Password) != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	if err := h.setSession(c, user.ID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"data": userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	})
}

// Logout clears the auth cookie. It is public and idempotent: clearing an
// absent or already-expired cookie is a no-op.
func (h *Handler) Logout(c *fiber.Ctx) error {
	auth.ClearTokenCookie(c, h.cookieSecure)
	return c.SendStatus(fiber.StatusNoContent)
}

// setSession issues a token for userID and writes it as the auth cookie.
func (h *Handler) setSession(c *fiber.Ctx, userID int64) error {
	token, err := h.issuer.Issue(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start session")
	}
	auth.SetTokenCookie(c, token, h.issuer.TTL(), h.cookieSecure)
	return nil
}

// Me returns the authenticated user. It runs behind auth.RequireAuth, which has
// already resolved and stored the user id.
func (h *Handler) Me(c *fiber.Ctx) error {
	id, ok := auth.UserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	user, err := h.queries.GetUserByID(c.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Valid token, but the user is gone: unauthorized, not 404.
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if err != nil {
		// Other failures fall through to ErrorHandler (500).
		return err
	}

	return c.JSON(fiber.Map{"data": userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}})
}

// normalizeEmail validates and lowercases an email address. Lowercasing matches
// the case-insensitive unique index on users(lower(email)).
func normalizeEmail(raw string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return strings.ToLower(addr.Address), nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505) — here, a duplicate email.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
