package users

import (
	"errors"
	"shortener/auth"
	"shortener/configs"
	"shortener/models"
	"shortener/ratelimit"

	"github.com/gofiber/fiber/v2"
)

func InitUserRoutes() func(router fiber.Router) {
	return func(router fiber.Router) {
		router.Post("/login", ratelimit.LoginByIP(), ratelimit.LoginByAccount(), login)
		if configs.AppConfig.RegistrationEnabled {
			router.Post("/register", ratelimit.RegisterByIP(), register)
		}
		router.Get("/me", auth.ValidateAuthHeader, getUserInfo)
		router.Patch("/me", auth.ValidateAuthHeader, updateUserInfo)
	}
}

// @Summary Login a user
// @Description Authenticate with email and password and return a JWT token. Email matching is case-insensitive.
// @Tags Users
// @Accept json
// @Produce json
// @Param loginDto body models.UserLoginDto true "Login details"
// @Success 200 {object} models.UserLoginResponseDto
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Router /users/login [post]
func login(c *fiber.Ctx) error {
	// Implement login logic here
	loginDto := models.UserLoginDto{}
	if err := c.BodyParser(&loginDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if loginDto.Email == "" || loginDto.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}
	userDto, err := ValidateUser(loginDto.Email, loginDto.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to log in",
		})
	}

	token, err := auth.CreateTokenForUser(&userDto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create token",
		})
	}
	userDto.Token = token
	return c.Status(fiber.StatusOK).JSON(userDto)
}

// @Summary Register a new user
// @Description Create an account. The email is the login ID; it is stored lowercase and must be unique (case-insensitive). Returns 409 when the email is already registered. Limited to 5 registrations per IP per hour. Available only when REGISTRATION_ENABLED is true.
// @Tags Users
// @Accept json
// @Produce json
// @Param createDto body models.UserCreateDto true "Registration details"
// @Success 201 {object} models.UserDto
// @Failure 400 {object} map[string]interface{} "Invalid body, or per-field errors under \"fields\""
// @Failure 409 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/register [post]
func register(c *fiber.Ctx) error {
	createDto := models.UserCreateDto{}
	if err := c.BodyParser(&createDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userDto, err := CreateUser(createDto)
	var validationErrs ValidationErrors
	switch {
	case errors.As(err, &validationErrs):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Validation failed",
			"fields": validationErrs,
		})
	case errors.Is(err, ErrEmailTaken):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":  err.Error(),
			"fields": fiber.Map{"email": "is already registered"},
		})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(userDto)
}

// @Summary Get user info
// @Description Retrieve information about the authenticated user
// @Tags Users
// @Produce json
// @Success 200 {object} models.UserDto
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/me [get]
// @Security BearerAuth
func getUserInfo(ctx *fiber.Ctx) error {
	// Implement get user info logic here
	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id

	userDto, err := GetUserById(userId)
	if errors.Is(err, ErrUserNotFound) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch user",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(userDto)
}

// @Summary Update user info
// @Description Update the authenticated user's information
// @Tags Users
// @Accept json
// @Produce json
// @Param updateDto body models.UserUpdateDto true "User update details. current_password is required when changing password"
// @Success 200 {object} models.UserDto
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/me [patch]
// @Security BearerAuth
func updateUserInfo(ctx *fiber.Ctx) error {
	// Implement update user info logic here
	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id

	updateDto := models.UserUpdateDto{}
	if err := ctx.BodyParser(&updateDto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if updateDto.Password == "" && updateDto.Name == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one field (password or name) is required",
		})
	}
	if updateDto.Password != "" && updateDto.CurrentPassword == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "current_password is required to change password",
		})
	}
	user, err := UpdateUser(userId, updateDto)
	var validationErrs ValidationErrors
	if errors.As(err, &validationErrs) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Validation failed",
			"fields": validationErrs,
		})
	}
	if errors.Is(err, ErrCurrentPassword) {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if errors.Is(err, ErrInvalidPassword) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(user)
}
