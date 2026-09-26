// Package analytics records clicks on short links.
package analytics

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// Click is one visit to a short link.
type Click struct {
	ID         uint64 `gorm:"column:id;primaryKey"`
	ShortURLID uint64 `gorm:"column:short_url_id"`
	ShortCode  string
	CreatedBy  uint64
	Agent      string
	IPAddress  string
	Metadata   map[string]interface{} `gorm:"serializer:json"`
	CreatedAt  time.Time
}

func (Click) TableName() string { return "url_redirects" }

// Store writes clicks.
type Store interface {
	Insert(ctx context.Context, clicks []Click) error
}

// GormStore writes clicks to Postgres.
type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

// Insert writes all clicks in a single statement.
func (s *GormStore) Insert(ctx context.Context, clicks []Click) error {
	for i := range clicks {
		if clicks[i].Metadata == nil {
			clicks[i].Metadata = map[string]interface{}{}
		}
	}
	return s.db.WithContext(ctx).CreateInBatches(clicks, len(clicks)).Error
}

// BatcherConfig bounds memory use and database load.
type BatcherConfig struct {
	// Buffer is how many clicks may wait to be written; more are dropped.
	Buffer int
	// BatchSize is the most clicks written in one insert.
	BatchSize int
	// FlushInterval is the longest a click waits before being written.
	FlushInterval time.Duration
	// WriteTimeout bounds each insert.
	WriteTimeout time.Duration
}

// DefaultBatcherConfig suits a single redirect instance.
var DefaultBatcherConfig = BatcherConfig{
	Buffer:        10000,
	BatchSize:     500,
	FlushInterval: time.Second,
	WriteTimeout:  5 * time.Second,
}

// Batcher collects clicks off the request path and writes them in batches
// from a single goroutine. When the buffer is full, clicks are dropped and
// counted rather than blocking redirects or growing without bound.
type Batcher struct {
	store   Store
	cfg     BatcherConfig
	clicks  chan Click
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
	dropped atomic.Uint64
	failed  atomic.Uint64
}

func NewBatcher(store Store, cfg BatcherConfig) *Batcher {
	b := &Batcher{
		store:  store,
		cfg:    cfg,
		clicks: make(chan Click, cfg.Buffer),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go b.run()
	return b
}

// Record queues c without blocking. It returns false if c was dropped.
func (b *Batcher) Record(c Click) bool {
	select {
	case b.clicks <- c:
		return true
	default:
		if n := b.dropped.Add(1); n == 1 || n%1000 == 0 {
			slog.Warn("click buffer full, dropping clicks", "dropped_total", n)
		}
		return false
	}
}

// Dropped returns how many clicks were discarded because the buffer was full.
func (b *Batcher) Dropped() uint64 { return b.dropped.Load() }

// Failed returns how many clicks were lost to failed inserts.
func (b *Batcher) Failed() uint64 { return b.failed.Load() }

// Close stops the batcher after writing every queued click, or when ctx ends.
// Clicks recorded after Close are not written.
func (b *Batcher) Close(ctx context.Context) error {
	b.once.Do(func() { close(b.stop) })
	select {
	case <-b.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Batcher) run() {
	defer close(b.done)
	ticker := time.NewTicker(b.cfg.FlushInterval)
	defer ticker.Stop()
	batch := make([]Click, 0, b.cfg.BatchSize)

	for {
		select {
		case c := <-b.clicks:
			batch = append(batch, c)
			if len(batch) >= b.cfg.BatchSize {
				batch = b.flush(batch)
			}
		case <-ticker.C:
			batch = b.flush(batch)
		case <-b.stop:
			for {
				select {
				case c := <-b.clicks:
					batch = append(batch, c)
					if len(batch) >= b.cfg.BatchSize {
						batch = b.flush(batch)
					}
				default:
					b.flush(batch)
					return
				}
			}
		}
	}
}

// flush writes batch and returns a new empty one; the store may keep the old
// slice. A failed insert is logged and its clicks are lost, since retrying
// could back up without bound.
func (b *Batcher) flush(batch []Click) []Click {
	if len(batch) == 0 {
		return batch
	}
	ctx, cancel := context.WithTimeout(context.Background(), b.cfg.WriteTimeout)
	defer cancel()
	if err := b.store.Insert(ctx, batch); err != nil {
		b.failed.Add(uint64(len(batch)))
		slog.Error("writing clicks failed", "clicks", len(batch), "err", err)
	}
	return make([]Click, 0, b.cfg.BatchSize)
}
