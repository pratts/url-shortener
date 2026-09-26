package migrations_test

import (
	"context"
	"errors"
	"testing"

	"shortener/internal/analytics"
	"shortener/internal/platform"
	"shortener/internal/testdb"
	"shortener/migrations"
)

func TestUpDownAndEnsureCurrent(t *testing.T) {
	db, err := platform.OpenPostgres(testdb.Config(t))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	ctx := context.Background()

	if _, err := migrations.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := migrations.DownAll(ctx, sqlDB); err != nil {
		t.Fatalf("down: %v", err)
	}
	if err := migrations.EnsureCurrent(ctx, sqlDB); !errors.Is(err, migrations.ErrPending) {
		t.Fatalf("empty schema: got %v, want ErrPending", err)
	}
	version, err := migrations.Up(ctx, sqlDB)
	if err != nil || version < 1 {
		t.Fatalf("up: version %d, %v", version, err)
	}
	if err := migrations.EnsureCurrent(ctx, sqlDB); err != nil {
		t.Fatalf("after up: %v", err)
	}
	if again, err := migrations.Up(ctx, sqlDB); err != nil || again != version {
		t.Fatalf("second up should be a no-op: %d, %v", again, err)
	}
}

func TestSchemaMatchesModels(t *testing.T) {
	db := testdb.Postgres(t)
	ctx := context.Background()
	owner := testdb.CreateUser(t, db, "alice@example.com")
	var linkID uint64
	if err := db.Raw("INSERT INTO shortened_urls (created_by, long_url, short_code) VALUES (?, 'https://example.com', 'CLICK01') RETURNING id", owner).Scan(&linkID).Error; err != nil {
		t.Fatal(err)
	}

	rec := analytics.NewRecorder(db)
	if err := rec.Record(ctx, analytics.Click{ShortURLID: linkID, ShortCode: "CLICK01", CreatedBy: owner, Agent: "test", IPAddress: "127.0.0.1"}); err != nil {
		t.Fatalf("recording a click: %v", err)
	}
	var metadata string
	var hasTime bool
	db.Raw("SELECT metadata::text, created_at IS NOT NULL FROM url_redirects").Row().Scan(&metadata, &hasTime)
	if metadata != "{}" || !hasTime {
		t.Fatalf("click stored with metadata %q and created_at set %v", metadata, hasTime)
	}

	if err := db.Exec("DELETE FROM shortened_urls WHERE id = ?", linkID).Error; err != nil {
		t.Fatal(err)
	}
	var clicks int
	db.Raw("SELECT count(*) FROM url_redirects").Scan(&clicks)
	if clicks != 0 {
		t.Fatalf("deleting a link left %d clicks behind", clicks)
	}
}
