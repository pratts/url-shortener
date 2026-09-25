package users

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	"shortener/models"
)

const (
	maxEmailLength = 254
	maxNameLength  = 100
)

// ValidationErrors maps a request field to what is wrong with it.
type ValidationErrors map[string]string

func (v ValidationErrors) Error() string {
	return "validation failed"
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NormalizeRegistration trims and lowercases the input, then checks every
// field, returning all problems at once.
func NormalizeRegistration(dto models.UserCreateDto) (models.UserCreateDto, error) {
	dto.Email = normalizeEmail(dto.Email)
	dto.Name = strings.TrimSpace(dto.Name)

	errs := ValidationErrors{}
	if len(dto.Email) > maxEmailLength {
		errs["email"] = "must be at most 254 characters"
	} else if addr, err := mail.ParseAddress(dto.Email); err != nil || addr.Address != dto.Email || addr.Name != "" {
		// Reject display-name forms like "Alice <a@b.c>"; only a bare address is accepted.
		errs["email"] = "must be a valid email address"
	}

	if n := utf8.RuneCountInString(dto.Name); n < 1 || n > maxNameLength {
		errs["name"] = "must be 1-100 characters"
	}

	if err := validatePassword(dto.Password); err != nil {
		errs["password"] = "must be between 8 and 72 bytes"
	}

	if len(errs) > 0 {
		return dto, errs
	}
	return dto, nil
}
