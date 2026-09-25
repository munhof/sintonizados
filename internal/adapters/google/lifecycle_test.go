package google

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/munhof/sintonizados/internal/domain"
)

func TestGeminiEOFWithoutFinalDoesNotFailSession(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "trailing_silence", true: "empty_stream"}[empty], func(t *testing.T) {
			var logs bytes.Buffer
			old := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
			defer slog.SetDefault(old)
			ended := make(chan bool, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer c.Close()
				var v map[string]any
				if c.ReadJSON(&v) != nil {
					return
				}
				c.WriteJSON(map[string]any{"setupComplete": map[string]any{}})
				for c.ReadJSON(&v) == nil {
					if input, ok := v["realtimeInput"].(map[string]any); ok && input["audioStreamEnd"] == true {
						ended <- true
					}
				}
			}))
			defer server.Close()
			chunks := make(chan domain.AudioChunk, 1)
			if !empty {
				chunks <- domain.AudioChunk{Data: []byte{0, 0}, IngressAt: time.Now()}
			}
			close(chunks)
			tr := Transcriber{FinalWait: 30 * time.Millisecond, APIKey: "test", Endpoint: "ws" + strings.TrimPrefix(server.URL, "http")}
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			err := tr.Run(ctx, domain.SessionKnowledge{}, chunks, func(domain.Transcript) error { t.Error("unexpected transcript"); return nil })
			if !strings.Contains(logs.String(), `"close_reason":"final_wait_timeout"`) {
				t.Errorf("missing timeout outcome: %s", logs.String())
			}
			if err != nil {
				t.Errorf("EOF with no final must not fail session: %v", err)
			}
			select {
			case <-ended:
			default:
				t.Error("audioStreamEnd was not sent")
			}
		})
	}
}

func TestGeminiDistinguishesProviderErrorAndCancellation(t *testing.T) {
	for _, cancelStream := range []bool{false, true} {
		t.Run(map[bool]string{false: "provider_error", true: "context_cancellation"}[cancelStream], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer c.Close()
				var v map[string]any
				c.ReadJSON(&v)
				c.WriteJSON(map[string]any{"setupComplete": map[string]any{}})
				c.ReadJSON(&v)
				if cancelStream {
					cancel()
					c.ReadJSON(&v)
				} else {
					c.WriteJSON(map[string]any{"error": map[string]any{"code": 500}})
				}
			}))
			defer server.Close()
			chunks := make(chan domain.AudioChunk, 1)
			chunks <- domain.AudioChunk{Data: []byte{0, 0}, IngressAt: time.Now()}
			tr := Transcriber{Endpoint: "ws" + strings.TrimPrefix(server.URL, "http")}
			err := tr.Run(ctx, domain.SessionKnowledge{}, chunks, func(domain.Transcript) error { return nil })
			want := domain.ErrUnavailable
			if cancelStream {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("want %v, got %v", want, err)
			}
		})
	}
}
