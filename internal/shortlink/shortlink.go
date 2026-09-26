// Package shortlink creates, resolves and manages short links.
package shortlink

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("link not found")
	ErrCodeTaken      = errors.New("short code already exists")
	ErrCodesExhausted = errors.New("could not generate a unique short code")
	ErrInvalidTarget  = errors.New("url must be an absolute http(s) URL of at most 2048 characters, without credentials, and not pointing to this service")
)

// Link is a stored short link.
type Link struct {
	ID        uint64 `gorm:"column:id;primaryKey"`
	CreatedBy uint64
	LongURL   string
	ShortCode string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Link) TableName() string { return "shortened_urls" }

// Repository persists links. Every method taking an owner only matches links
// that user created.
type Repository interface {
	// Create stores l and sets its ID; it returns ErrCodeTaken on a code clash.
	Create(ctx context.Context, l *Link) error
	Get(ctx context.Context, id, owner uint64) (Link, error)
	ByCode(ctx context.Context, code string) (Link, error)
	// List returns up to limit links, newest first, with IDs below cursor
	// (0 means from the newest).
	List(ctx context.Context, owner uint64, limit int, cursor uint64) ([]Link, error)
	UpdateTarget(ctx context.Context, id, owner uint64, target string) (Link, error)
	Delete(ctx context.Context, id, owner uint64) (Link, error)
}

// Cache holds resolved targets by short code.
type Cache interface {
	Get(ctx context.Context, code string) (target string, ok bool, err error)
	Set(ctx context.Context, code, target string) error
	Delete(ctx context.Context, code string) error
}

// Input is the body of POST and PUT /urls.
type Input struct {
	URL string `json:"url"`
}

// View is a link as returned by the API.
type View struct {
	ID        uint64 `json:"id"`
	URL       string `json:"url"`
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	CreatedBy uint64 `json:"created_by"`
}

// Page is one page of GET /urls. NextCursor is null on the last page.
type Page struct {
	Items      []View  `json:"items"`
	NextCursor *string `json:"next_cursor"`
}
