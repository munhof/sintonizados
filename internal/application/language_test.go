package application

import (
	"context"
	"errors"
	"github.com/munhof/sintonizados/internal/adapters/demo"
	"github.com/munhof/sintonizados/internal/adapters/memory"
	"github.com/munhof/sintonizados/internal/domain"
	"testing"
	"time"
)

type decisions map[string]string

func (d decisions) Decide(_ context.Context, r domain.DecisionRequest) (domain.DecisionResult, error) {
	l, ok := d[r.Segment.Text]
	if !ok {
		return domain.DecisionResult{}, errors.New("offline")
	}
	return domain.DecisionResult{Language: l, LanguageConfidence: .9, Provider: "test", RequiresTranslation: l != "es"}, nil
}

type recordingTranslator struct {
	requests []domain.TranslationRequest
	detected string
}

func (r *recordingTranslator) Translate(_ context.Context, q domain.TranslationRequest) (domain.Translation, error) {
	r.requests = append(r.requests, q)
	return domain.Translation{Text: "traducido", DetectedSourceLanguage: r.detected}, nil
}
func TestDecidesPerSegmentAndPreservesSpanish(t *testing.T) {
	tr := &recordingTranslator{}
	s := New(memory.NewSessionStore(), memory.NewEventBus(), demo.Transcriber{}, tr, decisions{"Hola": "es", "Hello": "en", "Bueno hello": "mixed"}, "test", 4)
	defer s.Close()
	s.Store.Create(domain.Session{ID: "a", Language: "en"})
	for _, text := range []string{"Hola", "Hello", "Bueno hello", "?"} {
		now := time.Now()
		if err := s.translate(context.Background(), "a", domain.Transcript{Text: text, At: now, IngressAt: now, Final: true}); err != nil {
			t.Fatal(err)
		}
	}
	snap, _ := s.Snapshot("a")
	if len(snap.Subtitles) != 4 || snap.Subtitles[0].Spanish != "Hola" || snap.Subtitles[0].RequiresTranslation {
		t.Fatalf("%+v", snap.Subtitles)
	}
	if len(tr.requests) != 3 || tr.requests[0].Source != "en" || tr.requests[1].Source != "" || tr.requests[2].Source != "" {
		t.Fatalf("%+v", tr.requests)
	}
	for i, lang := range []string{"es", "en", "mixed", "unknown"} {
		sub := snap.Subtitles[i]
		if sub.Language != lang || sub.SegmentID == "" {
			t.Fatalf("%+v", sub)
		}
	}
	if snap.Subtitles[3].DecisionFallback == "" {
		t.Fatal("fallback must be visible")
	}
}
func TestMixedSubdivisionPreservesOrderAndSpanish(t *testing.T) {
	tr := &recordingTranslator{}
	s := New(memory.NewSessionStore(), memory.NewEventBus(), demo.Transcriber{}, tr, decisions{"Bueno, ahora we're going to talk about Kubernetes.": "mixed", "Bueno,": "es", "ahora we're going to talk about Kubernetes.": "mixed"}, "test", 4)
	defer s.Close()
	s.Store.Create(domain.Session{ID: "a"})
	now := time.Now()
	err := s.translate(context.Background(), "a", domain.Transcript{Text: "Bueno, ahora we're going to talk about Kubernetes.", At: now, IngressAt: now})
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot("a")
	if len(snap.Subtitles) != 2 || snap.Subtitles[0].Spanish != "Bueno," || len(tr.requests) != 1 || snap.Subtitles[1].ParentSegmentID == "" {
		t.Fatalf("%+v requests=%+v", snap.Subtitles, tr.requests)
	}
}

func TestUnknownWithGoogleDetectedSpanishIsPreserved(t *testing.T) {
	tr := &recordingTranslator{detected: "es"}
	s := New(memory.NewSessionStore(), memory.NewEventBus(), demo.Transcriber{}, tr, domain.DeterministicDecisionEngine{}, "test", 4)
	defer s.Close()
	s.Store.Create(domain.Session{ID: "a"})
	now := time.Now()
	if err := s.translate(context.Background(), "a", domain.Transcript{Text: "Hola, ¿cómo estamos?", At: now, IngressAt: now}); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot("a")
	sub := snap.Subtitles[0]
	if sub.Language != "es" || sub.RequiresTranslation || sub.Spanish != sub.Original || sub.DecisionFallback != "google_detected_es" {
		t.Fatalf("%+v", sub)
	}
}
