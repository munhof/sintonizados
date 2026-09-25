package google

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/munhof/sintonizados/internal/domain"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"
)

// One Gemini Live websocket per session. Raw audio never leaves this adapter as text.
type Transcriber struct {
	APIKey, Model, Endpoint string
	FinalWait               time.Duration
}
type liveMessage struct {
	SetupComplete *struct{} `json:"setupComplete"`
	Error         *struct {
		Code int `json:"code"`
	} `json:"error"`
	GoAway        *struct{} `json:"goAway"`
	ServerContent *struct {
		Interim *struct {
			Text string `json:"text"`
		} `json:"interimInputTranscription"`
		Final *struct {
			Text string `json:"text"`
		} `json:"inputTranscription"`
		TurnComplete bool `json:"turnComplete"`
	} `json:"serverContent"`
}

func (t Transcriber) Run(ctx context.Context, k domain.SessionKnowledge, audio <-chan domain.AudioChunk, emit func(domain.Transcript) error) (runErr error) {
	var lastAudio, lastTranscript, streamClose time.Time
	reason := "normal_stream_completion"
	defer func() {
		if runErr != nil {
			reason = "provider_error"
			if errors.Is(runErr, context.Canceled) {
				reason = "context_cancellation"
			}
			if errors.Is(runErr, context.DeadlineExceeded) {
				reason = "provider_timeout"
			}
		}
		wait := time.Duration(0)
		if !streamClose.IsZero() {
			wait = time.Since(streamClose)
		}
		slog.Info("gemini_stream_closed", "session_id", k.Metadata["session_id"], "last_audio_timestamp", lastAudio, "last_transcript_timestamp", lastTranscript, "stream_close_timestamp", streamClose, "final_wait_duration", wait.String(), "close_reason", reason)
	}()
	ctx, cancel := context.WithTimeout(ctx, 9*time.Minute)
	defer cancel()
	endpoint := t.Endpoint
	if endpoint == "" {
		endpoint = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return domain.ErrUnavailable
	}
	q := u.Query()
	q.Set("key", t.APIKey)
	u.RawQuery = q.Encode()
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	c, resp, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("gemini handshake failed: %w", domain.ErrUnavailable)
	}
	defer c.Close()
	c.SetReadLimit(2 << 20)
	stopClose := context.AfterFunc(ctx, func() { c.Close() })
	defer stopClose()
	write := func(v any) error {
		c.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.WriteJSON(v); err != nil {
			return fmt.Errorf("gemini write failed: %w", domain.ErrUnavailable)
		}
		return nil
	}
	model := t.Model
	if model == "" {
		model = "gemini-3.5-transcribe-live"
	}
	if !strings.HasPrefix(model, "models/") {
		model = "models/" + model
	}
	if err := write(map[string]any{"setup": map[string]any{"model": model, "generationConfig": map[string]any{"responseModalities": []string{"TEXT"}}, "inputAudioTranscription": map[string]any{"languageCodes": []string{"en-US"}}}}); err != nil {
		return err
	}
	c.SetReadDeadline(time.Now().Add(15 * time.Second))
	var setup liveMessage
	if err := c.ReadJSON(&setup); err != nil || setup.SetupComplete == nil {
		return fmt.Errorf("gemini setup rejected: %w", domain.ErrUnavailable)
	}
	type received struct {
		message liveMessage
		err     error
	}
	messages := make(chan received, 16)
	readctx, readcancel := context.WithCancel(ctx)
	defer readcancel()
	go func() {
		for {
			c.SetReadDeadline(time.Now().Add(45 * time.Second))
			var m liveMessage
			err := c.ReadJSON(&m)
			select {
			case messages <- received{m, err}:
			case <-readctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	var ingress time.Time
	var correlation string
	var drain <-chan time.Time
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	ending := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-drain:
			reason = "final_wait_timeout"
			return nil
		case chunk, ok := <-audio:
			if !ok {
				audio = nil
				ending = true
				streamClose = time.Now().UTC()
				if err := write(map[string]any{"realtimeInput": map[string]any{"audioStreamEnd": true}}); err != nil {
					return err
				}
				wait := t.FinalWait
				if wait <= 0 {
					wait = 10 * time.Second
				}
				timer = time.NewTimer(wait)
				drain = timer.C
				continue
			}
			lastAudio = chunk.IngressAt
			if ingress.IsZero() {
				ingress = chunk.IngressAt
				correlation = chunk.CorrelationID
			}
			if err := write(map[string]any{"realtimeInput": map[string]any{"audio": map[string]string{"mimeType": "audio/pcm;rate=16000", "data": base64.StdEncoding.EncodeToString(chunk.Data)}}}); err != nil {
				return err
			}
		case result := <-messages:
			if result.err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if ending && websocket.IsCloseError(result.err, websocket.CloseNormalClosure) {
					return nil
				}
				var timeout net.Error
				if errors.As(result.err, &timeout) && timeout.Timeout() {
					return fmt.Errorf("gemini read timeout: %w", context.DeadlineExceeded)
				}
				return fmt.Errorf("gemini read failed: %w", domain.ErrUnavailable)
			}
			m := result.message
			if m.Error != nil || m.GoAway != nil {
				return fmt.Errorf("gemini session closed by provider: %w", domain.ErrUnavailable)
			}
			if m.ServerContent == nil {
				continue
			}
			content := m.ServerContent
			if content.Interim != nil && content.Interim.Text != "" {
				lastTranscript = time.Now().UTC()
				if err := emit(domain.Transcript{Text: content.Interim.Text, IngressAt: ingress, At: time.Now().UTC(), CorrelationID: correlation}); err != nil {
					return err
				}
			}
			if content.Final != nil && strings.TrimSpace(content.Final.Text) != "" {
				lastTranscript = time.Now().UTC()
				if ingress.IsZero() {
					ingress = lastAudio
				}
				if err := emit(domain.Transcript{Text: content.Final.Text, Final: true, IngressAt: ingress, At: time.Now().UTC(), CorrelationID: correlation}); err != nil {
					return err
				}
				ingress = time.Time{}
			}
			if ending && content.TurnComplete {
				return nil
			}
		}
	}
}
