package shortlink

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryRepository is an in-memory Repository for tests.
type MemoryRepository struct {
	mu     sync.Mutex
	nextID uint64
	links  map[uint64]Link
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{links: map[uint64]Link{}}
}

func (r *MemoryRepository) Create(_ context.Context, l *Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.links {
		if existing.ShortCode == l.ShortCode {
			return ErrCodeTaken
		}
	}
	r.nextID++
	now := time.Now()
	l.ID, l.CreatedAt, l.UpdatedAt = r.nextID, now, now
	r.links[l.ID] = *l
	return nil
}

func (r *MemoryRepository) Get(_ context.Context, id, owner uint64) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.links[id]
	if !ok || l.CreatedBy != owner {
		return Link{}, ErrNotFound
	}
	return l, nil
}

func (r *MemoryRepository) ByCode(_ context.Context, code string) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.links {
		if l.ShortCode == code {
			return l, nil
		}
	}
	return Link{}, ErrNotFound
}

func (r *MemoryRepository) List(_ context.Context, owner uint64, limit int, cursor uint64) ([]Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Link
	for _, l := range r.links {
		if l.CreatedBy == owner && (cursor == 0 || l.ID < cursor) {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryRepository) UpdateTarget(_ context.Context, id, owner uint64, target string) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.links[id]
	if !ok || l.CreatedBy != owner {
		return Link{}, ErrNotFound
	}
	l.LongURL, l.UpdatedAt = target, time.Now()
	r.links[id] = l
	return l, nil
}

func (r *MemoryRepository) Delete(_ context.Context, id, owner uint64) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.links[id]
	if !ok || l.CreatedBy != owner {
		return Link{}, ErrNotFound
	}
	delete(r.links, id)
	return l, nil
}

// MemoryCache is an in-memory Cache for tests. Entries never expire.
type MemoryCache struct {
	mu      sync.Mutex
	entries map[string]Entry
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{entries: map[string]Entry{}}
}

func (c *MemoryCache) Get(_ context.Context, code string) (Entry, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[code]
	return e, ok, nil
}

func (c *MemoryCache) Set(_ context.Context, code string, e Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[code] = e
	return nil
}

func (c *MemoryCache) Delete(_ context.Context, code string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, code)
	return nil
}
