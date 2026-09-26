package shortlink_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"shortener/internal/cache"
	"shortener/internal/shortlink"
	"shortener/internal/testdb"
)

func TestGormRepository(t *testing.T) {
	db := testdb.Postgres(t)
	repo := shortlink.NewGormRepository(db)
	ctx := context.Background()
	alice := testdb.CreateUser(t, db, "alice@example.com")
	bob := testdb.CreateUser(t, db, "bob@example.com")

	link := shortlink.Link{CreatedBy: alice, LongURL: "https://example.com/1", ShortCode: "CODE001"}
	if err := repo.Create(ctx, &link); err != nil || link.ID == 0 || link.CreatedAt.IsZero() {
		t.Fatalf("create: %+v, %v", link, err)
	}
	dup := shortlink.Link{CreatedBy: bob, LongURL: "https://example.com/2", ShortCode: "CODE001"}
	if err := repo.Create(ctx, &dup); !errors.Is(err, shortlink.ErrCodeTaken) {
		t.Fatalf("duplicate code: got %v, want ErrCodeTaken", err)
	}
	orphan := shortlink.Link{CreatedBy: 999999, LongURL: "https://example.com", ShortCode: "ORPHAN1"}
	if err := repo.Create(ctx, &orphan); err == nil {
		t.Fatal("link with a nonexistent owner was accepted; the foreign key is missing")
	}

	if _, err := repo.Get(ctx, link.ID, bob); !errors.Is(err, shortlink.ErrNotFound) {
		t.Fatalf("get by another user: %v", err)
	}
	if got, err := repo.ByCode(ctx, "CODE001"); err != nil || got.ID != link.ID {
		t.Fatalf("by code: %+v, %v", got, err)
	}
	if _, err := repo.ByCode(ctx, "NOPE000"); !errors.Is(err, shortlink.ErrNotFound) {
		t.Fatalf("unknown code: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	updated, err := repo.UpdateTarget(ctx, link.ID, alice, "https://example.com/new")
	if err != nil || updated.LongURL != "https://example.com/new" || updated.ShortCode != "CODE001" ||
		updated.CreatedBy != alice || !updated.UpdatedAt.After(link.UpdatedAt) {
		t.Fatalf("update should return the full, updated row: %+v, %v", updated, err)
	}
	if _, err := repo.UpdateTarget(ctx, link.ID, bob, "https://evil.example"); !errors.Is(err, shortlink.ErrNotFound) {
		t.Fatalf("update by another user: %v", err)
	}

	for i := 2; i <= 5; i++ {
		l := shortlink.Link{CreatedBy: alice, LongURL: "https://example.com", ShortCode: "CODE00" + string(rune('0'+i))}
		if err := repo.Create(ctx, &l); err != nil {
			t.Fatal(err)
		}
	}
	first, _ := repo.List(ctx, alice, 3, 0)
	rest, _ := repo.List(ctx, alice, 3, first[len(first)-1].ID)
	if len(first) != 3 || len(rest) != 2 || first[0].ID < first[2].ID || rest[0].ID >= first[2].ID {
		t.Fatalf("list pages: %v then %v", ids(first), ids(rest))
	}
	if none, _ := repo.List(ctx, bob, 10, 0); len(none) != 0 {
		t.Fatalf("bob sees alice's links: %v", ids(none))
	}

	if _, err := repo.Delete(ctx, link.ID, bob); !errors.Is(err, shortlink.ErrNotFound) {
		t.Fatalf("delete by another user: %v", err)
	}
	deleted, err := repo.Delete(ctx, link.ID, alice)
	if err != nil || deleted.ShortCode != "CODE001" {
		t.Fatalf("delete should return the deleted row: %+v, %v", deleted, err)
	}
	if _, err := repo.Delete(ctx, link.ID, alice); !errors.Is(err, shortlink.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}

func TestServiceWithRedisCache(t *testing.T) {
	db := testdb.Postgres(t)
	rdb := testdb.Redis(t)
	ctx := context.Background()
	linkCache := cache.NewLinkCache(rdb, time.Minute)
	svc, err := shortlink.NewService(shortlink.NewGormRepository(db), linkCache, "https://tidylnk.com")
	if err != nil {
		t.Fatal(err)
	}
	owner := testdb.CreateUser(t, db, "alice@example.com")

	created, err := svc.Create(ctx, owner, "https://example.com/old")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { linkCache.Delete(ctx, created.ShortCode) })
	if e, err := svc.Resolve(ctx, created.ShortCode); err != nil || e.Target != "https://example.com/old" || e.LinkID != created.ID {
		t.Fatalf("resolve: %+v, %v", e, err)
	}
	if ttl := rdb.TTL(ctx, "link:"+created.ShortCode).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("cached entry TTL %v, want (0, 1m]", ttl)
	}
	cached, ok, err := linkCache.Get(ctx, created.ShortCode)
	if err != nil || !ok || cached.Owner != owner || cached.Target != "https://example.com/old" {
		t.Fatalf("cache should hold the full entry: %+v, %v, %v", cached, ok, err)
	}
	if _, err := svc.Update(ctx, created.ID, owner, "https://example.com/new"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := linkCache.Get(ctx, created.ShortCode); ok {
		t.Fatal("update left a stale cache entry")
	}
	if err := svc.Delete(ctx, created.ID, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve(ctx, created.ShortCode); !errors.Is(err, shortlink.ErrNotFound) {
		t.Fatalf("resolve after delete: %v", err)
	}
	if e, ok, _ := linkCache.Get(ctx, created.ShortCode); !ok || !e.Missing {
		t.Fatalf("a miss should be cached: %+v, %v", e, ok)
	}
	if ttl := rdb.TTL(ctx, "link:"+created.ShortCode).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("miss TTL %v, want (0, 1m]", ttl)
	}
}

func ids(links []shortlink.Link) []uint64 {
	out := make([]uint64, len(links))
	for i, l := range links {
		out[i] = l.ID
	}
	return out
}
