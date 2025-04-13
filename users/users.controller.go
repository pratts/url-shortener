package users

import (
	"shortener/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func Init(app *fiber.App) {
	InitTokenParams()
	app.Post("/users/login", login)
	app.Post("/users/register", register)
	app.Get("/users/:userId", getUserInfo)
	app.Patch("/users/:userId", updateUserInfo)
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

	token, err := CreateTokenForUser(&userDto)
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

func getUserInfo(c *fiber.Ctx) error {
	// Implement get user info logic here
	userId := c.Params("userId")
	if userId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}
	userIdUint64, err := strconv.ParseUint(userId, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	userDto, err := GetUserById(userIdUint64)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	return c.Status(fiber.StatusOK).JSON(userDto)
}

func updateUserInfo(c *fiber.Ctx) error {
	// Implement update user info logic here
	userId := c.Params("userId")
	if userId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}
	userIdUint64, err := strconv.ParseUint(userId, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}
	updateDto := models.UserUpdateDto{}
	if err := c.BodyParser(&updateDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if updateDto.Password == "" && updateDto.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one field (password or name) is required",
		})
	}
	user, err := UpdateUser(userIdUint64, updateDto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}
	return c.Status(fiber.StatusOK).JSON(user)
}
