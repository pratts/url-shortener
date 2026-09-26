package analytics

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeStore records inserted batches; it can be made to block or fail.
type fakeStore struct {
	mu      sync.Mutex
	batches [][]Click
	block   chan struct{}
	err     error
}

func (s *fakeStore) Insert(_ context.Context, clicks []Click) error {
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.batches = append(s.batches, append([]Click(nil), clicks...))
	return nil
}

func (s *fakeStore) sizes() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []int
	for _, b := range s.batches {
		out = append(out, len(b))
	}
	return out
}

func (s *fakeStore) total() int {
	n := 0
	for _, size := range s.sizes() {
		n += size
	}
	return n
}

func cfg(buffer, batch int, interval time.Duration) BatcherConfig {
	return BatcherConfig{Buffer: buffer, BatchSize: batch, FlushInterval: interval, WriteTimeout: time.Second}
}

func TestBatchesBySize(t *testing.T) {
	store := &fakeStore{}
	b := NewBatcher(store, cfg(100, 10, time.Hour))
	for i := 0; i < 25; i++ {
		b.Record(Click{ShortCode: "CODE001"})
	}
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := store.sizes(); len(got) != 3 || got[0] != 10 || got[1] != 10 || got[2] != 5 {
		t.Fatalf("batch sizes %v, want [10 10 5]", got)
	}
}

func TestFlushesOnInterval(t *testing.T) {
	store := &fakeStore{}
	b := NewBatcher(store, cfg(100, 1000, 20*time.Millisecond))
	defer b.Close(context.Background())
	b.Record(Click{})
	b.Record(Click{})
	deadline := time.Now().Add(2 * time.Second)
	for store.total() < 2 {
		if time.Now().After(deadline) {
			t.Fatal("clicks were not flushed on the interval")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestDropsWhenFullWithoutBlocking(t *testing.T) {
	store := &fakeStore{block: make(chan struct{})}
	b := NewBatcher(store, cfg(5, 1, time.Hour))

	// The first click is taken by the worker, which then blocks in Insert;
	// five more fill the buffer; the rest must be dropped immediately.
	start := time.Now()
	accepted := 0
	for i := 0; i < 50; i++ {
		if b.Record(Click{}) {
			accepted++
		}
		if i == 0 {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if time.Since(start) > time.Second {
		t.Fatal("Record blocked while the buffer was full")
	}
	if accepted != 6 || b.Dropped() != 44 {
		t.Fatalf("accepted %d, dropped %d; want 6 and 44", accepted, b.Dropped())
	}
	close(store.block)
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.total() != 6 {
		t.Fatalf("wrote %d clicks, want the 6 accepted", store.total())
	}
}

func TestCloseDrainsQueue(t *testing.T) {
	store := &fakeStore{}
	b := NewBatcher(store, cfg(1000, 100, time.Hour))
	for i := 0; i < 250; i++ {
		b.Record(Click{})
	}
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.total() != 250 {
		t.Fatalf("wrote %d of 250 clicks on close", store.total())
	}
	if err := b.Close(context.Background()); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestCloseRespectsContext(t *testing.T) {
	store := &fakeStore{block: make(chan struct{})}
	defer close(store.block)
	b := NewBatcher(store, cfg(10, 1, time.Hour))
	b.Record(Click{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := b.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want DeadlineExceeded", err)
	}
}

func TestFailedInsertsAreCounted(t *testing.T) {
	store := &fakeStore{err: errors.New("db down")}
	b := NewBatcher(store, cfg(100, 10, time.Hour))
	for i := 0; i < 15; i++ {
		b.Record(Click{})
	}
	b.Close(context.Background())
	if b.Failed() != 15 {
		t.Fatalf("failed %d, want 15", b.Failed())
	}
}
