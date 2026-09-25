package users

import (
	"errors"
	"shortener/models"
	"strings"
	"testing"
)

func validDto() models.UserCreateDto {
	return models.UserCreateDto{
		Email:    "alice@example.com",
		Name:     "Alice",
		Password: "correct horse",
	}
}

func TestNormalizeRegistrationAcceptsAndNormalizes(t *testing.T) {
	dto := validDto()
	dto.Email = " Alice@Example.COM "
	dto.Name = "  Alice  "
	got, err := NormalizeRegistration(dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "alice@example.com" || got.Name != "Alice" {
		t.Fatalf("not normalized: %+v", got)
	}
}

func TestNormalizeRegistrationRejects(t *testing.T) {
	cases := []struct {
		field string
		edit  func(*models.UserCreateDto)
	}{
		{"email", func(d *models.UserCreateDto) { d.Email = "" }},
		{"email", func(d *models.UserCreateDto) { d.Email = "not-an-email" }},
		{"email", func(d *models.UserCreateDto) { d.Email = "Alice <alice@example.com>" }},
		{"email", func(d *models.UserCreateDto) { d.Email = "a@b.c, d@e.f" }},
		{"email", func(d *models.UserCreateDto) { d.Email = strings.Repeat("a", 250) + "@x.io" }},
		{"name", func(d *models.UserCreateDto) { d.Name = "   " }},
		{"name", func(d *models.UserCreateDto) { d.Name = strings.Repeat("é", 101) }},
		{"password", func(d *models.UserCreateDto) { d.Password = "short" }},
		{"password", func(d *models.UserCreateDto) { d.Password = strings.Repeat("a", 73) }},
	}
	for _, c := range cases {
		dto := validDto()
		c.edit(&dto)
		_, err := NormalizeRegistration(dto)
		var verrs ValidationErrors
		if !errors.As(err, &verrs) {
			t.Errorf("%+v: expected ValidationErrors, got %v", dto, err)
			continue
		}
		if _, ok := verrs[c.field]; !ok || len(verrs) != 1 {
			t.Errorf("%+v: expected only %q to fail, got %v", dto, c.field, verrs)
		}
	}
}

func TestNormalizeRegistrationReportsAllFields(t *testing.T) {
	_, err := NormalizeRegistration(models.UserCreateDto{})
	var verrs ValidationErrors
	if !errors.As(err, &verrs) || len(verrs) != 3 {
		t.Fatalf("expected 3 field errors, got %v", err)
	}
}

func TestNameLengthCountsCharactersNotBytes(t *testing.T) {
	dto := validDto()
	dto.Name = strings.Repeat("é", 100) // 200 bytes, 100 characters
	if _, err := NormalizeRegistration(dto); err != nil {
		t.Fatalf("100-character name rejected: %v", err)
	}
}
