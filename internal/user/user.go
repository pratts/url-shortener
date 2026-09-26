// Package user manages accounts: registration, login and profile updates.
package user

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("user not found")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrCurrentPassword    = errors.New("current password is incorrect")
)

// User is a stored account. Email is the login ID and is stored lowercase.
type User struct {
	ID        uint64 `gorm:"column:id;primaryKey"`
	Email     string
	Password  string // bcrypt hash
	Name      string
	Verified  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Repository persists users.
type Repository interface {
	// Create stores u and sets its ID; it returns ErrEmailTaken on conflict.
	Create(ctx context.Context, u *User) error
	ByID(ctx context.Context, id uint64) (User, error)
	ByEmail(ctx context.Context, email string) (User, error)
	// Update sets the given columns and returns the updated user.
	Update(ctx context.Context, id uint64, fields map[string]interface{}) (User, error)
}

// RegisterInput is the body of POST /users/register.
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginInput is the body of POST /users/login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateInput is the body of PATCH /users/me. CurrentPassword is required
// when Password is set.
type UpdateInput struct {
	Password        string `json:"password"`
	CurrentPassword string `json:"current_password"`
	Name            string `json:"name"`
}

// Profile is a user as returned by the API.
type Profile struct {
	ID       uint64 `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Verified bool   `json:"verified"`
}

// Session is the response to a successful login.
type Session struct {
	Profile
	Token string `json:"token"`
}

func (u User) Profile() Profile {
	return Profile{ID: u.ID, Email: u.Email, Name: u.Name, Verified: u.Verified}
}
