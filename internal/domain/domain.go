package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("session not found")
	ErrConflict    = errors.New("session state or audio sequence conflict")
	ErrBusy        = errors.New("capacity exhausted; retry later")
	ErrInvalid     = errors.New("invalid input")
	ErrUnavailable = errors.New("provider unavailable")
)

type SessionKnowledge struct {
	RecentTranscript      []string
	Language, Topic       string
	Terminology, Glossary map[string]string
	Entities              []string
	Metadata              map[string]string
}
type Session struct {
	ID                                                        string
	Title, Language, Status, TranscriptionStatus, Mode, Error string
	CreatedAt                                                 time.Time
	SubtitleCount                                             int64
	Knowledge                                                 SessionKnowledge
}
type Subtitle struct {
	ID              int64     `json:"id"`
	SessionID       string    `json:"session_id"`
	CorrelationID   string    `json:"correlation_id"`
	Original        string    `json:"original"`
	Spanish         string    `json:"spanish"`
	Final           bool      `json:"final"`
	AudioIngressAt  time.Time `json:"audio_ingress_at"`
	TranscriptAt    time.Time `json:"transcript_at"`
	TranslationAt   time.Time `json:"translation_at"`
	PublishedAt     time.Time `json:"published_at"`
	TranscriptionMS float64   `json:"transcription_ms"`
	TranslationMS   float64   `json:"translation_ms"`
	EndToEndMS      float64   `json:"end_to_end_ms"`
}
type Snapshot struct {
	Session   Session
	Subtitles []Subtitle
	Partial   string
}
type SessionStore interface {
	Create(Session) error
	Get(string) (Snapshot, error)
	List() []Session
	Update(string, func(*Snapshot)) error
}
type EventKind string

const (
	SessionStarted        EventKind = "SessionStarted"
	AudioChunkReceived    EventKind = "AudioChunkReceived"
	TranscriptPartial     EventKind = "TranscriptPartial"
	TranscriptFinal       EventKind = "TranscriptFinal"
	TranslationRequested  EventKind = "TranslationRequested"
	TranslationProduced   EventKind = "TranslationProduced"
	SubtitlePublished     EventKind = "SubtitlePublished"
	LatencyObserved       EventKind = "LatencyObserved"
	SessionEnded          EventKind = "SessionEnded"
	TranscriptionComplete EventKind = "TranscriptionComplete"
)

type Event struct {
	ID, SessionID, CorrelationID string
	Timestamp                    time.Time
	Kind                         EventKind
	Payload                      any
}

// Reliable subscriptions exert bounded backpressure. Observers may drop notifications.
// Consumers must subscribe before publishing; this MVP bus has no durable delivery.
type EventBus interface {
	Subscribe(string, EventKind, ...bool) (<-chan Event, func())
	Publish(context.Context, Event) error
}
type AudioChunk struct {
	Data          []byte
	Sequence      int64
	IngressAt     time.Time
	CorrelationID string
}
type Transcript struct {
	Text          string
	Final         bool
	IngressAt, At time.Time
	CorrelationID string
}
type Transcriber interface {
	Run(context.Context, SessionKnowledge, <-chan AudioChunk, func(Transcript) error) error
}
type TranslationRequest struct {
	Text, Source, Target string
	Knowledge            SessionKnowledge
}
type Translation struct {
	Text     string
	Provider string
}
type Translator interface {
	Translate(context.Context, TranslationRequest) (Translation, error)
}
type ReasoningEngine interface {
	Enrich(context.Context, SessionKnowledge, Transcript) (SessionKnowledge, error)
}
type Strategy struct {
	Translator    string
	NeedReasoning bool
}
type GateResult struct{ Publish, Retry, Escalate bool }
type DecisionEngine interface {
	Before(context.Context, TranslationRequest) (Strategy, error)
	After(context.Context, TranslationRequest, Translation) (GateResult, error)
}
type DeterministicDecisionEngine struct{}

func (DeterministicDecisionEngine) Before(context.Context, TranslationRequest) (Strategy, error) {
	return Strategy{Translator: "standard"}, nil
}
func (DeterministicDecisionEngine) After(context.Context, TranslationRequest, Translation) (GateResult, error) {
	return GateResult{Publish: true}, nil
}
