package users

import (
	"errors"
	"shortener/db"
	"shortener/models"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	bcryptCost        = 12
	minPasswordLength = 8
	// bcrypt only uses the first 72 bytes of its input.
	maxPasswordBytes = 72
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidPassword    = errors.New("password must be between 8 and 72 bytes")
	ErrCurrentPassword    = errors.New("current password is incorrect")
	ErrEmailTaken         = errors.New("email is already registered")
)

// dummyHash is compared against when the email is unknown so that login takes
// the same time whether or not the account exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)

func validatePassword(password string) error {
	if len(password) < minPasswordLength || len(password) > maxPasswordBytes {
		return ErrInvalidPassword
	}
	return nil
}

func hashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation on a constraint whose name contains column.
func isUniqueViolation(err error, column string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, column)
}

func ValidateUser(email string, password string) (models.UserLoginResponseDto, error) {
	var user models.User
	err := db.DBObj.Where("email = ?", normalizeEmail(email)).First(&user).Error
	if err != nil {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.UserLoginResponseDto{}, ErrInvalidCredentials
		}
		return models.UserLoginResponseDto{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return models.UserLoginResponseDto{}, ErrInvalidCredentials
	}

	userDto := models.UserLoginResponseDto{
		Id:    user.Id,
		Email: user.Email,
		Name:  user.Name,
	}
	return userDto, nil
}

func GetUserById(id uint64) (models.UserDto, error) {
	var user models.User
	if err := db.DBObj.Where("id = ?", id).First(&user).Error; err != nil {
		return models.UserDto{}, err
	}
	return toUserDto(user), nil
}

func toUserDto(user models.User) models.UserDto {
	return models.UserDto{
		Id:       user.Id,
		Email:    user.Email,
		Verified: user.Verified,
		Name:     user.Name,
	}
}

// CreateUser validates and normalizes dto, then stores the user with a hashed
// password. It returns ValidationErrors or ErrEmailTaken for bad input.
func CreateUser(dto models.UserCreateDto) (models.UserDto, error) {
	dto, err := NormalizeRegistration(dto)
	if err != nil {
		return models.UserDto{}, err
	}
	hash, err := hashPassword(dto.Password)
	if err != nil {
		return models.UserDto{}, err
	}
	user := models.User{
		Email:    dto.Email,
		Password: hash,
		Name:     dto.Name,
		Verified: false,
	}
	// Rely on the unique constraint rather than a pre-check, so concurrent
	// registrations for the same email cannot both succeed.
	if err := db.DBObj.Create(&user).Error; err != nil {
		if isUniqueViolation(err, "email") {
			return models.UserDto{}, ErrEmailTaken
		}
		return models.UserDto{}, err
	}
	return toUserDto(user), nil
}

func UpdateUser(id uint64, update models.UserUpdateDto) (models.UserDto, error) {
	updates := map[string]interface{}{}
	if update.Name != "" {
		updates["name"] = update.Name
	}
	if update.Password != "" {
		var user models.User
		if err := db.DBObj.Where("id = ?", id).First(&user).Error; err != nil {
			return models.UserDto{}, err
		}
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(update.CurrentPassword)) != nil {
			return models.UserDto{}, ErrCurrentPassword
		}
		hash, err := hashPassword(update.Password)
		if err != nil {
			return models.UserDto{}, err
		}
		updates["password"] = hash
	}

	res := db.DBObj.Model(&models.User{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return models.UserDto{}, res.Error
	}

	return GetUserById(id)
}
