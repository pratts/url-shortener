package users

import (
	"shortener/auth"
	"shortener/models"

	"github.com/gofiber/fiber/v2"
)

func InitUserRoutes() func(router fiber.Router) {
	return func(router fiber.Router) {
		router.Post("/login", login)
		router.Post("/register", register)
		router.Get("/me", auth.ValidateAuthHeader, getUserInfo)
		router.Patch("/me", auth.ValidateAuthHeader, updateUserInfo)
	}
}

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
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
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

func register(c *fiber.Ctx) error {
	// Implement registration logic here
	createDto := models.UserCreateDto{}
	if err := c.BodyParser(&createDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if createDto.Email == "" || createDto.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	userDto, err := CreateUser(createDto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(userDto)
}

func getUserInfo(ctx *fiber.Ctx) error {
	// Implement get user info logic here
	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id

	userDto, err := GetUserById(userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(userDto)
}

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
	user, err := UpdateUser(userId, updateDto)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(user)
}
