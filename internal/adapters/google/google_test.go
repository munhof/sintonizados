package google

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/munhof/sintonizados/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTranslationUsesStructuredRequestAndDecodesEntities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goog-Api-Key") != "test" {
			t.Error("missing key")
		}
		var v map[string]any
		json.NewDecoder(r.Body).Decode(&v)
		if v["source"] != "en" || v["target"] != "es" || v["format"] != "text" {
			t.Error(v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"translations":[{"translatedText":"Go &amp; personas","detectedSourceLanguage":"en"}]}}`))
	}))
	defer server.Close()
	tr := Translator{APIKey: "test", Endpoint: server.URL, Client: server.Client()}
	got, err := tr.Translate(context.Background(), domain.TranslationRequest{Text: "Go & people", Source: "en", Target: "es"})
	if err != nil || got.Text != "Go & personas" || got.DetectedSourceLanguage != "en" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestTranslationSupportsSpanishToEnglish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			t.Error(err)
			return
		}
		if v["source"] != "es" || v["target"] != "en" {
			t.Errorf("expected Spanish-to-English request, got %v", v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"translations":[{"translatedText":"Hello"}]}}`))
	}))
	defer server.Close()
	got, err := (Translator{Endpoint: server.URL, Client: server.Client()}).Translate(context.Background(), domain.TranslationRequest{Text: "Hola", Source: "es", Target: "en"})
	if err != nil || got.Text != "Hello" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestGeminiStreamsPCMAndWaitsForFinal(t *testing.T) {
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		var v map[string]any
		if err := c.ReadJSON(&v); err != nil {
			t.Error(err)
			return
		}
		if v["setup"] == nil {
			t.Fatal(v)
		}
		config := v["setup"].(map[string]any)["inputAudioTranscription"].(map[string]any)
		if codes, ok := config["languageCodes"].([]any); !ok || len(codes) != 0 {
			t.Errorf("expected automatic language detection, got %v", config)
		}
		c.WriteJSON(map[string]any{"setupComplete": map[string]any{}})
		if err := c.ReadJSON(&v); err != nil {
			t.Error(err)
			return
		}
		if v["realtimeInput"] == nil {
			t.Error(v)
		}
		c.WriteJSON(map[string]any{"serverContent": map[string]any{"interimInputTranscription": map[string]any{"text": "Hello"}}})
		if err := c.ReadJSON(&v); err != nil {
			t.Error(err)
			return
		}
		c.WriteJSON(map[string]any{"serverContent": map[string]any{"inputTranscription": map[string]any{"text": "Hello world."}, "turnComplete": true}})
	}))
	defer server.Close()
	tr := Transcriber{APIKey: "test", Model: "models/test", Endpoint: "ws" + strings.TrimPrefix(server.URL, "http")}
	chunks := make(chan domain.AudioChunk, 1)
	chunks <- domain.AudioChunk{Data: []byte{0, 0}, IngressAt: time.Now(), CorrelationID: "c"}
	close(chunks)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var got []domain.Transcript
	err := tr.Run(ctx, domain.SessionKnowledge{Language: "en"}, chunks, func(v domain.Transcript) error { got = append(got, v); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Final || !got[1].Final || got[1].CorrelationID != "c" {
		t.Fatalf("%+v", got)
	}
}
func TestTranslationFailureDoesNotLeakKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403); w.Write([]byte("secret-test")) }))
	defer server.Close()
	tr := Translator{APIKey: "secret-test", Endpoint: server.URL, Client: server.Client()}
	_, err := tr.Translate(context.Background(), domain.TranslationRequest{})
	if err == nil || strings.Contains(err.Error(), "secret-test") {
		t.Fatal(err)
	}
}

func TestTranslationAutodetectOmitsSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		json.NewDecoder(r.Body).Decode(&v)
		if _, present := v["source"]; present {
			t.Error("unknown/mixed must omit source, not send an empty or fixed language")
		}
		w.Write([]byte(`{"data":{"translations":[{"translatedText":"Hola"}]}}`))
	}))
	defer server.Close()
	_, err := (Translator{Endpoint: server.URL}).Translate(context.Background(), domain.TranslationRequest{Text: "Hello", Target: "es"})
	if err != nil {
		t.Fatal(err)
	}
}
