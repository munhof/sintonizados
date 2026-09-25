// Package demo supplies deliberately scripted results, never speech recognition.
package demo

import (
	"context"
	"github.com/munhof/sintonizados/internal/domain"
	"time"
)

var originals = []string{"Welcome to Sintonizados.", "Each talk has its own context.", "Accessible conferences connect people."}
var translations = []string{"Bienvenidos a Sintonizados.", "Cada charla tiene su propio contexto.", "Las conferencias accesibles conectan personas."}

type Transcriber struct{}

func (Transcriber) Run(ctx context.Context, k domain.SessionKnowledge, in <-chan domain.AudioChunk, emit func(domain.Transcript) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			if err := emit(domain.Transcript{Text: originals[(chunk.Sequence-1)%3], Final: true, IngressAt: chunk.IngressAt, At: time.Now().UTC(), CorrelationID: chunk.CorrelationID}); err != nil {
				return err
			}
		}
	}
}

type Translator struct{}

func (Translator) Translate(ctx context.Context, r domain.TranslationRequest) (domain.Translation, error) {
	for i, s := range originals {
		if r.Text == s {
			return domain.Translation{Text: translations[i], Provider: "scripted-demo"}, nil
		}
	}
	return domain.Translation{}, domain.ErrInvalid
}
