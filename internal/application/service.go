package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/munhof/sintonizados/internal/domain"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type runtime struct {
	mu       sync.Mutex
	sequence int64
	ending   bool
	audio    chan domain.AudioChunk
	done     chan struct{}
	cancel   context.CancelFunc
}
type Service struct {
	Store       domain.SessionStore
	Bus         domain.EventBus
	transcriber domain.Transcriber
	translator  domain.Translator
	decision    domain.DecisionEngine
	Mode        string
	limit       int
	mu          sync.Mutex
	sessions    map[string]*runtime
	ctx         context.Context
	cancel      context.CancelFunc
}

func New(store domain.SessionStore, bus domain.EventBus, t domain.Transcriber, tr domain.Translator, d domain.DecisionEngine, mode string, limit int) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{Store: store, Bus: bus, transcriber: t, translator: tr, decision: d, Mode: mode, limit: limit, sessions: map[string]*runtime{}, ctx: ctx, cancel: cancel}
}
func id() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func (s *Service) emit(ctx context.Context, sid, cid string, kind domain.EventKind, payload any) error {
	return s.Bus.Publish(ctx, domain.Event{ID: id(), SessionID: sid, CorrelationID: cid, Timestamp: time.Now().UTC(), Kind: kind, Payload: payload})
}

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func (s *Service) Create(ctx context.Context, sid, title, language string) (domain.Session, error) {
	if !validID.MatchString(sid) || strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > 200 || language != "en" {
		return domain.Session{}, domain.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx.Err() != nil {
		return domain.Session{}, domain.ErrUnavailable
	}
	if _, exists := s.sessions[sid]; exists {
		return domain.Session{}, domain.ErrConflict
	}
	// Bound total retained sessions as well as concurrent providers.
	if len(s.sessions) >= s.limit {
		return domain.Session{}, domain.ErrBusy
	}
	v := domain.Session{ID: sid, Title: title, Language: language, Status: "active", TranscriptionStatus: "waiting", Mode: s.Mode, CreatedAt: time.Now().UTC(), Knowledge: domain.SessionKnowledge{Language: language}}
	if err := s.Store.Create(v); err != nil {
		return v, err
	}
	runctx, cancel := context.WithCancel(s.ctx)
	r := &runtime{audio: make(chan domain.AudioChunk, 32), done: make(chan struct{}), cancel: cancel}
	s.sessions[sid] = r
	transcripts, unsubscribe := s.Bus.Subscribe(sid, domain.TranscriptFinal, true)
	completed := make(chan error, 1)
	go func() {
		completed <- s.transcriber.Run(runctx, v.Knowledge, r.audio, func(t domain.Transcript) error {
			if !t.Final {
				s.Store.Update(sid, func(x *domain.Snapshot) { x.Partial = t.Text })
				return s.emit(runctx, sid, t.CorrelationID, domain.TranscriptPartial, t)
			}
			return s.emit(runctx, sid, t.CorrelationID, domain.TranscriptFinal, t)
		})
	}()
	go s.process(runctx, sid, r, transcripts, completed, unsubscribe)
	s.emit(runctx, sid, "", domain.SessionStarted, v)
	return v, nil
}
func (s *Service) Ingest(sid string, sequence int64, data []byte) error {
	if len(data) < 2 || len(data) > 32000 || len(data)%2 != 0 || sequence < 1 {
		return domain.ErrInvalid
	}
	s.mu.Lock()
	r, ok := s.sessions[sid]
	s.mu.Unlock()
	if !ok {
		return domain.ErrNotFound
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ending || sequence != r.sequence+1 {
		return domain.ErrConflict
	}
	chunk := domain.AudioChunk{Data: append([]byte{}, data...), Sequence: sequence, IngressAt: time.Now().UTC(), CorrelationID: id()}
	select {
	case r.audio <- chunk:
		r.sequence = sequence
		s.Store.Update(sid, func(x *domain.Snapshot) { x.Session.TranscriptionStatus = "streaming" })
		s.emit(s.ctx, sid, chunk.CorrelationID, domain.AudioChunkReceived, sequence)
		return nil
	default:
		return domain.ErrBusy
	}
}
func (s *Service) process(ctx context.Context, sid string, r *runtime, transcripts <-chan domain.Event, completed <-chan error, unsubscribe func()) {
	defer close(r.done)
	defer r.cancel()
	defer unsubscribe()
	var failure error
	defer func() {
		r.mu.Lock()
		r.ending = true
		r.mu.Unlock()
		s.Store.Update(sid, func(x *domain.Snapshot) {
			if failure != nil {
				x.Session.Status = "failed"
				x.Session.TranscriptionStatus = "error"
				x.Session.Error = "provider processing failed"
			} else {
				x.Session.Status = "ended"
				x.Session.TranscriptionStatus = "complete"
			}
			x.Partial = ""
		})
		if failure != nil {
			slog.Error("session_failed", "session_id", sid, "error", failure.Error())
		}
		s.emit(context.Background(), sid, "", domain.SessionEnded, nil)
	}()
	for {
		select {
		case <-ctx.Done():
			failure = ctx.Err()
			return
		case e := <-transcripts:
			if err := s.translate(ctx, sid, e.Payload.(domain.Transcript)); err != nil {
				failure = err
				return
			}
		case err := <-completed:
			// Run has returned, so all transcript events are already queued.
			if err != nil {
				failure = err
				return
			}
			for {
				select {
				case e := <-transcripts:
					if err := s.translate(ctx, sid, e.Payload.(domain.Transcript)); err != nil {
						failure = err
						return
					}
				default:
					return
				}
			}
		}
	}
}
func (s *Service) translate(ctx context.Context, sid string, t domain.Transcript) error {
	snap, err := s.Store.Get(sid)
	if err != nil {
		return err
	}
	req := domain.TranslationRequest{Text: t.Text, Source: snap.Session.Language, Target: "es", Knowledge: snap.Session.Knowledge}
	strategy, err := s.decision.Before(ctx, req)
	if err != nil {
		return err
	}
	if strategy.Translator != "standard" || strategy.NeedReasoning {
		return domain.ErrUnavailable
	}
	s.emit(ctx, sid, t.CorrelationID, domain.TranslationRequested, req)
	start := time.Now()
	callctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := s.translator.Translate(callctx, req)
	if err != nil {
		return err
	}
	translated := time.Now().UTC()
	gate, err := s.decision.After(ctx, req, result)
	if err != nil {
		return err
	}
	if !gate.Publish || gate.Retry || gate.Escalate {
		return domain.ErrUnavailable
	}
	s.emit(ctx, sid, t.CorrelationID, domain.TranslationProduced, result)
	now := time.Now().UTC()
	sub := domain.Subtitle{SessionID: sid, CorrelationID: t.CorrelationID, Original: t.Text, Spanish: result.Text, Final: true, AudioIngressAt: t.IngressAt, TranscriptAt: t.At, TranslationAt: translated, PublishedAt: now, TranscriptionMS: float64(t.At.Sub(t.IngressAt).Microseconds()) / 1000, TranslationMS: float64(translated.Sub(start).Microseconds()) / 1000, EndToEndMS: float64(now.Sub(t.IngressAt).Microseconds()) / 1000}
	s.Store.Update(sid, func(x *domain.Snapshot) {
		x.Session.SubtitleCount++
		sub.ID = x.Session.SubtitleCount
		x.Subtitles = append(x.Subtitles, sub)
		if len(x.Subtitles) > 200 {
			x.Subtitles = append([]domain.Subtitle{}, x.Subtitles[len(x.Subtitles)-200:]...)
		}
		k := &x.Session.Knowledge
		k.RecentTranscript = append(k.RecentTranscript, t.Text)
		if len(k.RecentTranscript) > 20 {
			k.RecentTranscript = append([]string{}, k.RecentTranscript[len(k.RecentTranscript)-20:]...)
		}
		x.Partial = ""
	})
	s.emit(ctx, sid, t.CorrelationID, domain.SubtitlePublished, sub)
	s.emit(ctx, sid, t.CorrelationID, domain.LatencyObserved, sub.EndToEndMS)
	slog.Info("subtitle_published", "session_id", sid, "correlation_id", t.CorrelationID, "subtitle_id", sub.ID, "transcription_ms", sub.TranscriptionMS, "translation_ms", sub.TranslationMS, "end_to_end_ms", sub.EndToEndMS)
	return nil
}
func (s *Service) End(ctx context.Context, sid string) error {
	s.mu.Lock()
	r, ok := s.sessions[sid]
	s.mu.Unlock()
	if !ok {
		return domain.ErrNotFound
	}
	r.mu.Lock()
	if !r.ending {
		r.ending = true
		close(r.audio)
	}
	r.mu.Unlock()
	select {
	case <-r.done:
		snap, _ := s.Store.Get(sid)
		if snap.Session.Status == "failed" {
			return domain.ErrUnavailable
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Service) Snapshot(id string) (domain.Snapshot, error) { return s.Store.Get(id) }
func (s *Service) Close() {
	s.cancel()
	s.mu.Lock()
	rs := []*runtime{}
	for _, r := range s.sessions {
		rs = append(rs, r)
	}
	s.mu.Unlock()
	for _, r := range rs {
		<-r.done
	}
}
