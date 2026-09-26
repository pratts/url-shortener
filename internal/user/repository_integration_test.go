package user_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"shortener/internal/testdb"
	"shortener/internal/user"

	"golang.org/x/crypto/bcrypt"
)

func TestGormRepository(t *testing.T) {
	db := testdb.Postgres(t)
	repo := user.NewGormRepository(db)
	ctx := context.Background()

	u := user.User{Email: "alice@example.com", Password: "hash", Name: "Alice"}
	if err := repo.Create(ctx, &u); err != nil || u.ID == 0 || u.CreatedAt.IsZero() {
		t.Fatalf("create: %+v, %v", u, err)
	}
	dup := user.User{Email: "alice@example.com", Password: "hash", Name: "A2"}
	if err := repo.Create(ctx, &dup); !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("duplicate email: got %v, want ErrEmailTaken", err)
	}
	if got, err := repo.ByEmail(ctx, "alice@example.com"); err != nil || got.ID != u.ID || got.Verified {
		t.Fatalf("by email: %+v, %v", got, err)
	}
	if _, err := repo.ByID(ctx, 999999); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("unknown id: %v", err)
	}
	updated, err := repo.Update(ctx, u.ID, map[string]interface{}{"name": "Alice B"})
	if err != nil || updated.Name != "Alice B" || updated.Email != u.Email {
		t.Fatalf("update should return the full row: %+v, %v", updated, err)
	}
	if _, err := repo.Update(ctx, 999999, map[string]interface{}{"name": "X"}); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("update unknown: %v", err)
	}
}

func TestConcurrentRegistrationOfOneEmail(t *testing.T) {
	db := testdb.Postgres(t)
	svc := user.NewService(user.NewGormRepository(db), user.WithPasswordCost(bcrypt.MinCost))

	var wg sync.WaitGroup
	results := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Register(context.Background(), user.RegisterInput{
				Email: "racer@example.com", Name: "R", Password: "correct horse",
			})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	ok, taken := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, user.ErrEmailTaken):
			taken++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || taken != 4 {
		t.Fatalf("got %d successes and %d conflicts, want 1 and 4", ok, taken)
	}
}
