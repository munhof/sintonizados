package application_test

import (
	"context"
	"github.com/munhof/sintonizados/internal/adapters/demo"
	"github.com/munhof/sintonizados/internal/adapters/memory"
	"github.com/munhof/sintonizados/internal/application"
	"github.com/munhof/sintonizados/internal/domain"
	"sync"
	"testing"
	"time"
)

func newService(t *testing.T) *application.Service {
	t.Helper()
	s := application.New(memory.NewSessionStore(), memory.NewEventBus(), demo.Transcriber{}, demo.Translator{}, domain.DeterministicDecisionEngine{}, "demo", 8)
	t.Cleanup(s.Close)
	return s
}
func TestTwoSessionsKeepIndependentSubtitles(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	for _, id := range []string{"a", "b"} {
		if _, err := s.Create(ctx, id, id, "en"); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for _, id := range []string{"a", "b"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			for n := int64(1); n <= 3; n++ {
				if err := s.Ingest(id, n, make([]byte, 3200)); err != nil {
					t.Error(err)
				}
			}
			if err := s.End(ctx, id); err != nil {
				t.Error(err)
			}
		}(id)
	}
	wg.Wait()
	for _, id := range []string{"a", "b"} {
		snap, err := s.Snapshot(id)
		if err != nil {
			t.Fatal(err)
		}
		if len(snap.Subtitles) != 3 {
			t.Fatalf("%s: %d subtitles", id, len(snap.Subtitles))
		}
		for i, sub := range snap.Subtitles {
			if sub.SessionID != id || sub.ID != int64(i+1) || sub.Spanish == "" {
				t.Fatalf("mixed or missing subtitle: %+v", sub)
			}
			if sub.PublishedAt.Before(sub.AudioIngressAt) || sub.EndToEndMS < 0 {
				t.Fatal("invalid latency")
			}
		}
	}
}
func TestRejectDuplicateAndEndedAudio(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	s.Create(ctx, "a", "Talk", "en")
	if err := s.Ingest("a", 1, []byte{0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := s.Ingest("a", 1, []byte{0, 0}); err != domain.ErrConflict {
		t.Fatalf("duplicate: %v", err)
	}
	if err := s.Ingest("a", 3, []byte{0, 0}); err != domain.ErrConflict {
		t.Fatalf("gap: %v", err)
	}
	if err := s.End(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Ingest("a", 2, []byte{0, 0}); err != domain.ErrConflict {
		t.Fatalf("ended: %v", err)
	}
}
func TestSlowSubscriberDoesNotBlockProcessing(t *testing.T) {
	s := newService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Create(ctx, "a", "Talk", "en")
	_, stop := s.Bus.Subscribe("a", domain.SubtitlePublished)
	defer stop()
	for n := int64(1); n <= 80; n++ {
		for {
			err := s.Ingest("a", n, []byte{0, 0})
			if err == nil {
				break
			}
			if err != domain.ErrBusy {
				t.Fatal(err)
			}
			time.Sleep(time.Millisecond)
		}
	}
	if err := s.End(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot("a")
	if len(snap.Subtitles) != 80 {
		t.Fatal(len(snap.Subtitles))
	}
}
