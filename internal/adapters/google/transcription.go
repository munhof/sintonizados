package google

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/munhof/sintonizados/internal/domain"
	"net/url"
	"strings"
	"time"
)

// One Gemini Live websocket per session. Raw audio never leaves this adapter as text.
type Transcriber struct{ APIKey, Model, Endpoint string }
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

func (t Transcriber) Run(ctx context.Context, k domain.SessionKnowledge, audio <-chan domain.AudioChunk, emit func(domain.Transcript) error) error {
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
			return fmt.Errorf("gemini final transcript timeout: %w", domain.ErrUnavailable)
		case chunk, ok := <-audio:
			if !ok {
				audio = nil
				ending = true
				if ingress.IsZero() {
					return nil
				}
				if err := write(map[string]any{"realtimeInput": map[string]any{"audioStreamEnd": true}}); err != nil {
					return err
				}
				timer = time.NewTimer(10 * time.Second)
				drain = timer.C
				continue
			}
			if ingress.IsZero() {
				ingress = chunk.IngressAt
				correlation = chunk.CorrelationID
			}
			if err := write(map[string]any{"realtimeInput": map[string]any{"audio": map[string]string{"mimeType": "audio/pcm;rate=16000", "data": base64.StdEncoding.EncodeToString(chunk.Data)}}}); err != nil {
				return err
			}
		case result := <-messages:
			if result.err != nil {
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
				if err := emit(domain.Transcript{Text: content.Interim.Text, IngressAt: ingress, At: time.Now().UTC(), CorrelationID: correlation}); err != nil {
					return err
				}
			}
			if content.Final != nil && strings.TrimSpace(content.Final.Text) != "" {
				if ingress.IsZero() {
					continue
				}
				if err := emit(domain.Transcript{Text: content.Final.Text, Final: true, IngressAt: ingress, At: time.Now().UTC(), CorrelationID: correlation}); err != nil {
					return err
				}
				ingress = time.Time{}
				if ending {
					return nil
				}
			}
			if ending && content.TurnComplete {
				return nil
			}
		}
	}
}
