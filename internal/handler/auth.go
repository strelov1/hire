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

// Register creates an account and returns the user plus a signed token.
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

	token, err := h.issuer.Issue(row.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to issue token")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": fiber.Map{
			"user":  userResponse{ID: row.ID, Email: row.Email, CreatedAt: row.CreatedAt},
			"token": token,
		},
	})
}

// Login verifies credentials and returns the user plus a signed token. Unknown
// email, wrong password, and passwordless accounts all yield the same generic
// 401 so the response never reveals which factor failed.
func (h *Handler) Login(c *fiber.Ctx) error {
	var in credentials
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	user, err := h.queries.GetUserByEmail(c.Context(), in.Email)
	if err != nil || !user.PasswordHash.Valid || auth.CheckPassword(user.PasswordHash.String, in.Password) != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	token, err := h.issuer.Issue(user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to issue token")
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"user":  userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
			"token": token,
		},
	})
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
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get user")
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
