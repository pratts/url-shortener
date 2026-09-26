package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost        = 12
	minPasswordLength = 8
	// bcrypt only uses the first 72 bytes of its input.
	maxPasswordBytes = 72
	maxEmailLength   = 254
	maxNameLength    = 100

	passwordRule = "must be between 8 and 72 bytes"
	nameRule     = "must be 1-100 characters"
)

// ValidationErrors maps a request field to what is wrong with it.
type ValidationErrors map[string]string

func (v ValidationErrors) Error() string {
	return "validation failed"
}

// Service implements account operations on top of a Repository.
type Service struct {
	repo Repository
	cost int
	// dummyHash is compared against when the email is unknown so that login
	// takes the same time whether or not the account exists.
	dummyHash []byte
}

// Option configures a Service.
type Option func(*Service)

// WithPasswordCost sets the bcrypt cost. Only tests should lower it.
func WithPasswordCost(cost int) Option {
	return func(s *Service) { s.cost = cost }
}

func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{repo: repo, cost: bcryptCost}
	for _, opt := range opts {
		opt(s)
	}
	s.dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), s.cost)
	return s
}

// Register validates input and creates an account. It returns
// ValidationErrors or ErrEmailTaken for bad input.
func (s *Service) Register(ctx context.Context, in RegisterInput) (Profile, error) {
	in, err := normalizeRegistration(in)
	if err != nil {
		return Profile{}, err
	}
	hash, err := s.hashPassword(in.Password)
	if err != nil {
		return Profile{}, err
	}
	u := User{Email: in.Email, Password: hash, Name: in.Name}
	// The unique constraint, not a pre-check, rejects duplicates, so
	// concurrent registrations for one email cannot both succeed.
	if err := s.repo.Create(ctx, &u); err != nil {
		return Profile{}, err
	}
	return u.Profile(), nil
}

// Authenticate checks an email and password and returns the account.
func (s *Service) Authenticate(ctx context.Context, email, password string) (Profile, error) {
	u, err := s.repo.ByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ErrNotFound) {
		bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return Profile{}, ErrInvalidCredentials
	}
	if err != nil {
		return Profile{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return Profile{}, ErrInvalidCredentials
	}
	return u.Profile(), nil
}

func (s *Service) Get(ctx context.Context, id uint64) (Profile, error) {
	u, err := s.repo.ByID(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	return u.Profile(), nil
}

// Update changes the name and/or password. Changing the password requires the
// current one; it returns ErrCurrentPassword if that is wrong.
func (s *Service) Update(ctx context.Context, id uint64, in UpdateInput) (Profile, error) {
	fields := map[string]interface{}{}
	if in.Name != "" {
		name, ok := validName(in.Name)
		if !ok {
			return Profile{}, ValidationErrors{"name": nameRule}
		}
		fields["name"] = name
	}
	if in.Password != "" {
		if in.CurrentPassword == "" {
			return Profile{}, ValidationErrors{"current_password": "is required to change password"}
		}
		current, err := s.repo.ByID(ctx, id)
		if err != nil {
			return Profile{}, err
		}
		if bcrypt.CompareHashAndPassword([]byte(current.Password), []byte(in.CurrentPassword)) != nil {
			return Profile{}, ErrCurrentPassword
		}
		hash, err := s.hashPassword(in.Password)
		if err != nil {
			return Profile{}, err
		}
		fields["password"] = hash
	}
	if len(fields) == 0 {
		return Profile{}, ValidationErrors{"name": "or password is required"}
	}
	u, err := s.repo.Update(ctx, id, fields)
	if err != nil {
		return Profile{}, err
	}
	return u.Profile(), nil
}

func (s *Service) hashPassword(password string) (string, error) {
	if !validPassword(password) {
		return "", ValidationErrors{"password": passwordRule}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func validPassword(password string) bool {
	return len(password) >= minPasswordLength && len(password) <= maxPasswordBytes
}

// validName trims name and checks its length in characters.
func validName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	return name, n >= 1 && n <= maxNameLength
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// normalizeRegistration trims and lowercases the input, then checks every
// field, returning all problems at once.
func normalizeRegistration(in RegisterInput) (RegisterInput, error) {
	errs := ValidationErrors{}
	in.Email = normalizeEmail(in.Email)
	if len(in.Email) > maxEmailLength {
		errs["email"] = "must be at most 254 characters"
	} else if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email || addr.Name != "" {
		// Only a bare address is accepted, not forms like "Alice <a@b.c>".
		errs["email"] = "must be a valid email address"
	}
	var ok bool
	if in.Name, ok = validName(in.Name); !ok {
		errs["name"] = nameRule
	}
	if !validPassword(in.Password) {
		errs["password"] = passwordRule
	}
	if len(errs) > 0 {
		return in, errs
	}
	return in, nil
}
