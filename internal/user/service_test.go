package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func newTestService() (*Service, *MemoryRepository) {
	repo := NewMemoryRepository()
	return NewService(repo, WithPasswordCost(bcrypt.MinCost)), repo
}

func validInput() RegisterInput {
	return RegisterInput{Email: "alice@example.com", Name: "Alice", Password: "correct horse"}
}

func TestRegisterNormalizesAndHashes(t *testing.T) {
	// Uses the production cost to check the stored hash.
	repo := NewMemoryRepository()
	svc := NewService(repo)
	profile, err := svc.Register(context.Background(), RegisterInput{
		Email: " Alice@Example.COM ", Name: "  Alice  ", Password: "correct horse",
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Email != "alice@example.com" || profile.Name != "Alice" || profile.Verified {
		t.Fatalf("not normalized: %+v", profile)
	}
	stored, _ := repo.ByID(context.Background(), profile.ID)
	cost, err := bcrypt.Cost([]byte(stored.Password))
	if err != nil || cost != bcryptCost {
		t.Fatalf("password not stored as bcrypt cost %d: %q", bcryptCost, stored.Password)
	}
}

func TestRegisterRejects(t *testing.T) {
	cases := []struct {
		field string
		edit  func(*RegisterInput)
	}{
		{"email", func(in *RegisterInput) { in.Email = "" }},
		{"email", func(in *RegisterInput) { in.Email = "not-an-email" }},
		{"email", func(in *RegisterInput) { in.Email = "Alice <alice@example.com>" }},
		{"email", func(in *RegisterInput) { in.Email = "a@b.c, d@e.f" }},
		{"email", func(in *RegisterInput) { in.Email = strings.Repeat("a", 250) + "@x.io" }},
		{"name", func(in *RegisterInput) { in.Name = "   " }},
		{"name", func(in *RegisterInput) { in.Name = strings.Repeat("é", 101) }},
		{"password", func(in *RegisterInput) { in.Password = "short" }},
		{"password", func(in *RegisterInput) { in.Password = strings.Repeat("a", 73) }},
	}
	for _, c := range cases {
		svc, _ := newTestService()
		in := validInput()
		c.edit(&in)
		_, err := svc.Register(context.Background(), in)
		var fields ValidationErrors
		if !errors.As(err, &fields) || len(fields) != 1 || fields[c.field] == "" {
			t.Errorf("%+v: expected only %q to fail, got %v", in, c.field, err)
		}
	}
}

func TestRegisterReportsAllFields(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{})
	var fields ValidationErrors
	if !errors.As(err, &fields) || len(fields) != 3 {
		t.Fatalf("expected 3 field errors, got %v", err)
	}
}

func TestNameLengthCountsCharacters(t *testing.T) {
	svc, _ := newTestService()
	in := validInput()
	in.Name = strings.Repeat("é", 100) // 200 bytes, 100 characters
	if _, err := svc.Register(context.Background(), in); err != nil {
		t.Fatalf("100-character name rejected: %v", err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, validInput()); err != nil {
		t.Fatal(err)
	}
	dup := validInput()
	dup.Email = "ALICE@example.com"
	if _, err := svc.Register(ctx, dup); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("got %v, want ErrEmailTaken", err)
	}
}

func TestAuthenticate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	registered, _ := svc.Register(ctx, validInput())

	got, err := svc.Authenticate(ctx, "ALICE@example.com", "correct horse")
	if err != nil || got.ID != registered.ID {
		t.Fatalf("login with different case: %+v, %v", got, err)
	}
	if _, err := svc.Authenticate(ctx, "alice@example.com", "wrong horse"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}
	if _, err := svc.Authenticate(ctx, "nobody@example.com", "correct horse"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email: got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	u, _ := svc.Register(ctx, validInput())

	var fields ValidationErrors
	if _, err := svc.Update(ctx, u.ID, UpdateInput{}); !errors.As(err, &fields) {
		t.Fatalf("empty update: got %v, want ValidationErrors", err)
	}
	if _, err := svc.Update(ctx, u.ID, UpdateInput{Name: "   "}); !errors.As(err, &fields) || fields["name"] == "" {
		t.Fatalf("blank name: got %v", err)
	}
	if p, err := svc.Update(ctx, u.ID, UpdateInput{Name: "  Alice B  "}); err != nil || p.Name != "Alice B" {
		t.Fatalf("name update: %+v, %v", p, err)
	}
	if _, err := svc.Update(ctx, u.ID, UpdateInput{Password: "new password"}); !errors.As(err, &fields) || fields["current_password"] == "" {
		t.Fatalf("password without current: got %v", err)
	}
	if _, err := svc.Update(ctx, u.ID, UpdateInput{Password: "new password", CurrentPassword: "nope nope"}); !errors.Is(err, ErrCurrentPassword) {
		t.Fatalf("wrong current password: got %v", err)
	}
	if _, err := svc.Update(ctx, u.ID, UpdateInput{Password: "short", CurrentPassword: "correct horse"}); !errors.As(err, &fields) || fields["password"] == "" {
		t.Fatalf("short new password: got %v", err)
	}
	if _, err := svc.Update(ctx, u.ID, UpdateInput{Password: "new password", CurrentPassword: "correct horse"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, u.Email, "correct horse"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("old password still works")
	}
	if _, err := svc.Authenticate(ctx, u.Email, "new password"); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}
	if _, err := svc.Update(ctx, 999, UpdateInput{Name: "X"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown user: got %v", err)
	}
}
