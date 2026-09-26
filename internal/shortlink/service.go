package shortlink

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
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

// reservedCodes are valid-looking codes the redirect service routes itself.
var reservedCodes = map[string]bool{"healthz": true}

// ValidCode reports whether code could be a generated short code, so requests
// for anything else can be answered without touching the cache or database.
func ValidCode(code string) bool {
	if len(code) != codeLength || reservedCodes[code] {
		return false
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
			return false
		}
	}
	return true
}

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
	for {
		code, err := randomCode()
		if err != nil || !reservedCodes[code] {
			return code, err
		}
	}
}

func randomCode() (string, error) {
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
			// Clear any cached "not found" for this code from before it existed.
			s.invalidate(ctx, code)
			return s.View(l), nil
		}
		if !errors.Is(err, ErrCodeTaken) {
			return View{}, err
		}
		slog.Warn("short code collision, retrying", "attempt", attempt)
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

// Resolve returns what the redirect path needs for code, using the cache when
// possible. Malformed codes, unknown codes and stored targets that fail
// validation all return ErrNotFound; misses are cached briefly.
func (s *Service) Resolve(ctx context.Context, code string) (Entry, error) {
	if !ValidCode(code) {
		return Entry{}, ErrNotFound
	}
	if e, ok, err := s.cache.Get(ctx, code); err != nil {
		slog.WarnContext(ctx, "cache read failed, using the database", "code", code, "err", err)
	} else if ok {
		if e.Missing {
			return Entry{}, ErrNotFound
		}
		return e, nil
	}

	l, err := s.repo.ByCode(ctx, code)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Entry{}, err
	}
	e := Entry{Missing: true}
	if err == nil {
		if _, invalid := s.ValidateTarget(l.LongURL); invalid == nil {
			e = Entry{LinkID: l.ID, Owner: l.CreatedBy, Target: l.LongURL}
		}
	}
	if err := s.cache.Set(ctx, code, e); err != nil {
		slog.WarnContext(ctx, "cache write failed", "code", code, "err", err)
	}
	if e.Missing {
		return Entry{}, ErrNotFound
	}
	return e, nil
}

func (s *Service) invalidate(ctx context.Context, code string) {
	if err := s.cache.Delete(ctx, code); err != nil {
		slog.WarnContext(ctx, "cache invalidation failed; the entry expires after its TTL", "code", code, "err", err)
	}
}
