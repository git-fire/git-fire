package executor

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestEventBuffer_AppendsAndTrims(t *testing.T) {
	buf := NewEventBuffer(3)
	for i := 0; i < 5; i++ {
		buf.Append(LogEntry{Action: fmt.Sprintf("a-%d", i)})
	}

	entries := buf.Snapshot()
	if len(entries) != 3 {
		t.Fatalf("len=%d want 3", len(entries))
	}
	if entries[0].Action != "a-2" || entries[2].Action != "a-4" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestEventBuffer_ConcurrentAppendAndSnapshot(t *testing.T) {
	buf := NewEventBuffer(16)
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				buf.Append(LogEntry{Action: "append"})
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				if n := len(buf.Snapshot()); n > 16 {
					t.Errorf("snapshot len=%d exceeds max 16", n)
					return
				}
			}
		}()
	}
	wg.Wait()
	if n := len(buf.Snapshot()); n != 16 {
		t.Fatalf("final len=%d want 16", n)
	}
}

func TestLogger_SubscribeReceivesWrittenEntry(t *testing.T) {
	logger, err := NewLogger(t.TempDir())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	got := make(chan LogEntry, 1)
	logger.Subscribe(func(e LogEntry) {
		got <- e
	})

	logger.Info("repo", "scan", "repo discovered")

	select {
	case e := <-got:
		if e.Action != "scan" {
			t.Fatalf("action=%q want scan", e.Action)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no subscriber event received")
	}
}
