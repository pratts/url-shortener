package shortlink

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

const base = "https://tidylnk.com"

func newTestService(t *testing.T) (*Service, *MemoryRepository, *MemoryCache) {
	t.Helper()
	repo, cache := NewMemoryRepository(), NewMemoryCache()
	svc, err := NewService(repo, cache, base)
	if err != nil {
		t.Fatal(err)
	}
	return svc, repo, cache
}

// fixedCodes makes the service generate the given codes in order.
func fixedCodes(svc *Service, codes ...string) *int {
	calls := 0
	svc.newCode = func() (string, error) {
		c := codes[calls]
		calls++
		return c, nil
	}
	return &calls
}

func TestValidateTarget(t *testing.T) {
	svc, _, _ := newTestService(t)
	valid := []string{
		"https://example.com",
		"http://example.com/path?q=1#frag",
		"  https://example.com/trimmed  ",
		"HTTPS://Example.com",
	}
	for _, u := range valid {
		if _, err := svc.ValidateTarget(u); err != nil {
			t.Errorf("expected %q to be valid, got %v", u, err)
		}
	}
	invalid := []string{
		"", "   ",
		"javascript:alert(1)", "JavaScript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd", "ftp://example.com",
		"//example.com", "example.com", "https://",
		"https://user:pass@example.com", "https://example.com@evil.com",
		"https://tidylnk.com/abc1234", "https://TIDYLNK.com/abc1234",
		"https://example.com/" + strings.Repeat("a", maxURLLength),
	}
	for _, u := range invalid {
		if _, err := svc.ValidateTarget(u); !errors.Is(err, ErrInvalidTarget) {
			t.Errorf("expected %q to be rejected, got %v", u, err)
		}
	}
}

func TestGenerateCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 10000; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != codeLength {
			t.Fatalf("code %q has length %d, want %d", code, len(code), codeLength)
		}
		for _, c := range code {
			if !strings.ContainsRune(base62, c) {
				t.Fatalf("code %q contains non-base62 character %q", code, c)
			}
		}
		if seen[code] {
			t.Fatalf("duplicate code %q in 10000 draws", code)
		}
		seen[code] = true
	}
}

func TestCreateRetriesOnCollision(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	calls := fixedCodes(svc, "SAME001", "SAME001", "SAME001", "FRESH01")

	first, err := svc.Create(ctx, 1, "https://example.com/1")
	if err != nil || first.ShortCode != "SAME001" {
		t.Fatalf("first create: %+v, %v", first, err)
	}
	second, err := svc.Create(ctx, 2, "https://example.com/2")
	if err != nil || second.ShortCode != "FRESH01" || *calls != 4 {
		t.Fatalf("second create: %+v, %v after %d codes; want FRESH01 after 4", second, err, *calls)
	}
	if target, _ := svc.Resolve(ctx, "SAME001"); target != "https://example.com/1" {
		t.Fatalf("original link changed to %q", target)
	}
}

func TestCreateGivesUpAfterMaxAttempts(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	svc.newCode = func() (string, error) { return "STUCK01", nil }
	if _, err := svc.Create(ctx, 1, "https://example.com/a"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, 1, "https://example.com/b"); !errors.Is(err, ErrCodesExhausted) {
		t.Fatalf("got %v, want ErrCodesExhausted", err)
	}
}

func TestCreateRejectsInvalidTarget(t *testing.T) {
	svc, repo, _ := newTestService(t)
	if _, err := svc.Create(context.Background(), 1, "javascript:alert(1)"); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("got %v, want ErrInvalidTarget", err)
	}
	if links, _ := repo.List(context.Background(), 1, 10, 0); len(links) != 0 {
		t.Fatal("invalid link was stored")
	}
}

func TestView(t *testing.T) {
	svc, _, _ := newTestService(t)
	ist := time.FixedZone("IST", 5*3600+1800)
	got := svc.View(Link{
		ID: 7, CreatedBy: 3, LongURL: "https://example.com", ShortCode: "abc1234",
		CreatedAt: time.Date(2026, 9, 26, 10, 0, 0, 0, ist),
		UpdatedAt: time.Date(2026, 9, 26, 11, 30, 0, 0, ist),
	})
	want := View{
		ID: 7, URL: "https://example.com", ShortCode: "abc1234",
		ShortURL:  "https://tidylnk.com/abc1234",
		CreatedAt: "2026-09-26T04:30:00Z", UpdatedAt: "2026-09-26T06:00:00Z",
		CreatedBy: 3,
	}
	if got != want {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestListPages(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	for i := 0; i < 7; i++ {
		if _, err := svc.Create(ctx, 1, "https://example.com/mine"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Create(ctx, 2, "https://example.com/theirs"); err != nil {
		t.Fatal(err)
	}

	var ids []uint64
	var cursor uint64
	for pages := 1; ; pages++ {
		page, err := svc.List(ctx, 1, 3, cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range page.Items {
			if v.CreatedBy != 1 {
				t.Fatalf("listed another user's link: %+v", v)
			}
			ids = append(ids, v.ID)
		}
		if page.NextCursor == nil {
			if pages != 3 {
				t.Fatalf("got %d pages, want 3", pages)
			}
			break
		}
		if pages > 3 {
			t.Fatal("pagination did not terminate")
		}
		cursor = mustParse(t, *page.NextCursor)
	}
	if len(ids) != 7 || ids[0] <= ids[6] {
		t.Fatalf("got ids %v, want 7 ids newest first", ids)
	}

	empty, err := svc.List(ctx, 3, 50, 0)
	if err != nil || empty.Items == nil || len(empty.Items) != 0 || empty.NextCursor != nil {
		t.Fatalf("empty list: %+v, %v; want non-nil empty items and nil cursor", empty, err)
	}
}

func TestUpdateAndDeleteInvalidateCache(t *testing.T) {
	svc, _, cache := newTestService(t)
	ctx := context.Background()
	created, _ := svc.Create(ctx, 1, "https://example.com/old")

	if target, _ := svc.Resolve(ctx, created.ShortCode); target != "https://example.com/old" {
		t.Fatalf("resolve: %q", target)
	}
	if _, ok, _ := cache.Get(ctx, created.ShortCode); !ok {
		t.Fatal("resolve did not populate the cache")
	}

	if _, err := svc.Update(ctx, created.ID, 2, "https://evil.example"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update by another user: got %v, want ErrNotFound", err)
	}
	updated, err := svc.Update(ctx, created.ID, 1, "https://example.com/new")
	if err != nil || updated.URL != "https://example.com/new" {
		t.Fatalf("update: %+v, %v", updated, err)
	}
	if _, ok, _ := cache.Get(ctx, created.ShortCode); ok {
		t.Fatal("update did not invalidate the cache")
	}
	if target, _ := svc.Resolve(ctx, created.ShortCode); target != "https://example.com/new" {
		t.Fatalf("resolve after update: %q", target)
	}

	if err := svc.Delete(ctx, created.ID, 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete by another user: got %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, created.ID, 1); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := cache.Get(ctx, created.ShortCode); ok {
		t.Fatal("delete did not invalidate the cache")
	}
	if _, err := svc.Resolve(ctx, created.ShortCode); !errors.Is(err, ErrNotFound) {
		t.Fatalf("resolve after delete: got %v, want ErrNotFound", err)
	}
}

func TestResolveRefusesUnsafeStoredTarget(t *testing.T) {
	svc, repo, cache := newTestService(t)
	ctx := context.Background()
	repo.Create(ctx, &Link{CreatedBy: 1, LongURL: "javascript:alert(1)", ShortCode: "EVIL001"})
	if _, err := svc.Resolve(ctx, "EVIL001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
	if _, ok, _ := cache.Get(ctx, "EVIL001"); ok {
		t.Fatal("unsafe target was cached")
	}
}

func TestNewServiceRejectsBadBase(t *testing.T) {
	if _, err := NewService(NewMemoryRepository(), NewMemoryCache(), "not a url"); err == nil {
		t.Fatal("expected an error")
	}
}

func mustParse(t *testing.T, s string) uint64 {
	t.Helper()
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		t.Fatalf("cursor %q is not numeric", s)
	}
	return n
}
