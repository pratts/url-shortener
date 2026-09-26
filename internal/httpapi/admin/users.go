package admin

import (
	"shortener/internal/httpapi"
	"shortener/internal/user"

	"github.com/gofiber/fiber/v2"
)

// @Summary Log in
// @Description Authenticate with email and password and return a JWT token. Email matching is case-insensitive.
// @Tags Users
// @Accept json
// @Produce json
// @Param body body user.LoginInput true "Login details"
// @Success 200 {object} user.Session
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Router /users/login [post]
func (h *Handler) login(ctx *fiber.Ctx) error {
	var in user.LoginInput
	if err := ctx.BodyParser(&in); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid request body")
	}
	if in.Email == "" || in.Password == "" {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Email and password are required")
	}
	profile, err := h.Users.Authenticate(ctx.UserContext(), in.Email, in.Password)
	if err != nil {
		return serviceError(ctx, err, "log in")
	}
	token, err := h.Tokens.Issue(profile.ID)
	if err != nil {
		return serviceError(ctx, err, "create token")
	}
	return ctx.JSON(user.Session{Profile: profile, Token: token})
}

// @Summary Register
// @Description Create an account. The email is the login ID; it is stored lowercase and must be unique (case-insensitive). Returns 409 when the email is already registered. Limited to 5 registrations per IP per hour. Available only when REGISTRATION_ENABLED is true.
// @Tags Users
// @Accept json
// @Produce json
// @Param body body user.RegisterInput true "Registration details"
// @Success 201 {object} user.Profile
// @Failure 400 {object} map[string]interface{} "Invalid body, or per-field errors under \"fields\""
// @Failure 409 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/register [post]
func (h *Handler) register(ctx *fiber.Ctx) error {
	var in user.RegisterInput
	if err := ctx.BodyParser(&in); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid request body")
	}
	profile, err := h.Users.Register(ctx.UserContext(), in)
	if err != nil {
		return serviceError(ctx, err, "create user")
	}
	return ctx.Status(fiber.StatusCreated).JSON(profile)
}

// @Summary Get the current user
// @Tags Users
// @Produce json
// @Success 200 {object} user.Profile
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users/me [get]
func (h *Handler) getMe(ctx *fiber.Ctx, userID uint64) error {
	profile, err := h.Users.Get(ctx.UserContext(), userID)
	if err != nil {
		return serviceError(ctx, err, "fetch user")
	}
	return ctx.JSON(profile)
}

// @Summary Update the current user
// @Description Change the name and/or password. current_password is required when changing the password.
// @Tags Users
// @Accept json
// @Produce json
// @Param body body user.UpdateInput true "Fields to change"
// @Success 200 {object} user.Profile
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users/me [patch]
func (h *Handler) updateMe(ctx *fiber.Ctx, userID uint64) error {
	var in user.UpdateInput
	if err := ctx.BodyParser(&in); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid request body")
	}
	profile, err := h.Users.Update(ctx.UserContext(), userID, in)
	if err != nil {
		return serviceError(ctx, err, "update user")
	}
	return ctx.JSON(profile)
}
