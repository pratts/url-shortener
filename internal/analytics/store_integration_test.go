package analytics_test

import (
	"context"
	"testing"
	"time"

	"shortener/internal/analytics"
	"shortener/internal/testdb"
)

func TestBatcherWritesToPostgres(t *testing.T) {
	db := testdb.Postgres(t)
	owner := testdb.CreateUser(t, db, "alice@example.com")
	var linkID uint64
	if err := db.Raw("INSERT INTO shortened_urls (created_by, long_url, short_code) VALUES (?, 'https://example.com', 'CLICK01') RETURNING id", owner).Scan(&linkID).Error; err != nil {
		t.Fatal(err)
	}

	b := analytics.NewBatcher(analytics.NewGormStore(db), analytics.BatcherConfig{
		Buffer: 5000, BatchSize: 500, FlushInterval: time.Hour, WriteTimeout: 5 * time.Second,
	})
	for i := 0; i < 1200; i++ {
		b.Record(analytics.Click{ShortURLID: linkID, ShortCode: "CLICK01", CreatedBy: owner, Agent: "agent", IPAddress: "127.0.0.1"})
	}
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Raw("SELECT count(*) FROM url_redirects WHERE short_url_id = ? AND created_at IS NOT NULL AND metadata = '{}'", linkID).Scan(&count)
	if count != 1200 || b.Failed() != 0 || b.Dropped() != 0 {
		t.Fatalf("stored %d of 1200 clicks (failed %d, dropped %d)", count, b.Failed(), b.Dropped())
	}
}
