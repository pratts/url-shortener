package user

import (
	"context"
	"sync"
	"time"
)

// MemoryRepository is an in-memory Repository for tests.
type MemoryRepository struct {
	mu     sync.Mutex
	nextID uint64
	users  map[uint64]User
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{users: map[uint64]User{}}
}

func (r *MemoryRepository) Create(_ context.Context, u *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Email == u.Email {
			return ErrEmailTaken
		}
	}
	r.nextID++
	now := time.Now()
	u.ID, u.CreatedAt, u.UpdatedAt = r.nextID, now, now
	r.users[u.ID] = *u
	return nil
}

func (r *MemoryRepository) ByID(_ context.Context, id uint64) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (r *MemoryRepository) ByEmail(_ context.Context, email string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (r *MemoryRepository) Update(_ context.Context, id uint64, fields map[string]interface{}) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	if v, ok := fields["name"].(string); ok {
		u.Name = v
	}
	if v, ok := fields["password"].(string); ok {
		u.Password = v
	}
	u.UpdatedAt = time.Now()
	r.users[id] = u
	return u, nil
}
