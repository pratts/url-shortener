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

// MemoryCache is an in-memory Cache for tests.
type MemoryCache struct {
	mu      sync.Mutex
	targets map[string]string
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{targets: map[string]string{}}
}

func (c *MemoryCache) Get(_ context.Context, code string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.targets[code]
	return t, ok, nil
}

func (c *MemoryCache) Set(_ context.Context, code, target string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.targets[code] = target
	return nil
}

func (c *MemoryCache) Delete(_ context.Context, code string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.targets, code)
	return nil
}
