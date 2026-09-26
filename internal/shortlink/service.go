package shortlink

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	base62          = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeLength      = 7
	maxURLLength    = 2048
	maxCodeAttempts = 5
)

// Service implements link operations on top of a Repository and Cache.
type Service struct {
	repo  Repository
	cache Cache
	// baseURL is the redirect service's public URL, used to build short URLs.
	baseURL string
	ownHost string
	newCode func() (string, error)
}

// NewService builds a Service. baseURL is the redirect service's public URL;
// links pointing at its host are refused.
func NewService(repo Repository, cache Cache, baseURL string) (*Service, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid short URL base %q", baseURL)
	}
	return &Service{
		repo:    repo,
		cache:   cache,
		baseURL: strings.TrimRight(baseURL, "/"),
		ownHost: u.Hostname(),
		newCode: GenerateCode,
	}, nil
}

// GenerateCode returns a random base62 code from crypto/rand. Bytes >= 248
// are rejected so every character is equally likely (248 = 4 * 62).
func GenerateCode() (string, error) {
	code := make([]byte, 0, codeLength)
	buf := make([]byte, codeLength*2)
	for len(code) < codeLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if b >= 248 {
				continue
			}
			code = append(code, base62[int(b)%len(base62)])
			if len(code) == codeLength {
				break
			}
		}
	}
	return string(code), nil
}

// ValidateTarget checks that raw is safe to redirect to and returns it with
// surrounding whitespace removed.
func (s *Service) ValidateTarget(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxURLLength {
		return "", ErrInvalidTarget
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ErrInvalidTarget
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Hostname() == "" || u.User != nil {
		return "", ErrInvalidTarget
	}
	if strings.EqualFold(u.Hostname(), s.ownHost) {
		return "", ErrInvalidTarget
	}
	return raw, nil
}

// View converts a stored link to its API representation.
func (s *Service) View(l Link) View {
	return View{
		ID:        l.ID,
		URL:       l.LongURL,
		ShortCode: l.ShortCode,
		ShortURL:  s.baseURL + "/" + l.ShortCode,
		CreatedAt: l.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: l.UpdatedAt.UTC().Format(time.RFC3339),
		CreatedBy: l.CreatedBy,
	}
}

// Create stores target under a new random code, retrying on collisions.
func (s *Service) Create(ctx context.Context, owner uint64, target string) (View, error) {
	target, err := s.ValidateTarget(target)
	if err != nil {
		return View{}, err
	}
	for attempt := 1; attempt <= maxCodeAttempts; attempt++ {
		code, err := s.newCode()
		if err != nil {
			return View{}, err
		}
		l := Link{CreatedBy: owner, LongURL: target, ShortCode: code}
		err = s.repo.Create(ctx, &l)
		if err == nil {
			return s.View(l), nil
		}
		if !errors.Is(err, ErrCodeTaken) {
			return View{}, err
		}
		log.Printf("short code collision on attempt %d, retrying", attempt)
	}
	return View{}, ErrCodesExhausted
}

func (s *Service) Get(ctx context.Context, id, owner uint64) (View, error) {
	l, err := s.repo.Get(ctx, id, owner)
	if err != nil {
		return View{}, err
	}
	return s.View(l), nil
}

// List returns one page of the owner's links, newest first. cursor is the
// NextCursor of the previous page, or empty for the first page.
func (s *Service) List(ctx context.Context, owner uint64, limit int, cursor uint64) (Page, error) {
	// Fetch one extra row to learn whether another page exists.
	links, err := s.repo.List(ctx, owner, limit+1, cursor)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: make([]View, 0, limit)}
	if len(links) > limit {
		links = links[:limit]
		next := strconv.FormatUint(links[limit-1].ID, 10)
		page.NextCursor = &next
	}
	for _, l := range links {
		page.Items = append(page.Items, s.View(l))
	}
	return page, nil
}

// Update changes a link's target and drops its cached copy.
func (s *Service) Update(ctx context.Context, id, owner uint64, target string) (View, error) {
	target, err := s.ValidateTarget(target)
	if err != nil {
		return View{}, err
	}
	l, err := s.repo.UpdateTarget(ctx, id, owner, target)
	if err != nil {
		return View{}, err
	}
	s.invalidate(ctx, l.ShortCode)
	return s.View(l), nil
}

// Delete removes a link and drops its cached copy.
func (s *Service) Delete(ctx context.Context, id, owner uint64) error {
	l, err := s.repo.Delete(ctx, id, owner)
	if err != nil {
		return err
	}
	s.invalidate(ctx, l.ShortCode)
	return nil
}

// Resolve returns the target for code, using the cache when possible. A
// stored target that fails validation is treated as not found.
func (s *Service) Resolve(ctx context.Context, code string) (string, error) {
	if target, ok, err := s.cache.Get(ctx, code); err == nil && ok {
		return target, nil
	} else if err != nil {
		log.Printf("cache read for %s failed, falling back to the database: %v", code, err)
	}
	l, err := s.repo.ByCode(ctx, code)
	if err != nil {
		return "", err
	}
	if _, err := s.ValidateTarget(l.LongURL); err != nil {
		return "", ErrNotFound
	}
	if err := s.cache.Set(ctx, code, l.LongURL); err != nil {
		log.Printf("cache write for %s failed: %v", code, err)
	}
	return l.LongURL, nil
}

// ByCode returns the stored link for code, bypassing the cache.
func (s *Service) ByCode(ctx context.Context, code string) (Link, error) {
	return s.repo.ByCode(ctx, code)
}

func (s *Service) invalidate(ctx context.Context, code string) {
	if err := s.cache.Delete(ctx, code); err != nil {
		log.Printf("failed to invalidate cache for %s, it will expire after the TTL: %v", code, err)
	}
}
