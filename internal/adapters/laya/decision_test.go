package laya

import (
	"context"
	"encoding/json"
	"github.com/munhof/sintonizados/internal/domain"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestOfficialMultilingualProtocol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/systemone" {
			t.Error(r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "multilingual" || body["state"] != "Hello world" {
			t.Error(body)
		}
		q := body["questions"].(map[string]any)["language"].(map[string]any)
		if q["type"] != "choice" || len(q["criteria"].(map[string]any)) != 4 {
			t.Error(q)
		}
		w.Write([]byte(`{"routing":{"model":"multilingual"},"answers":{"language":{"choice":"en","confidence":0.65,"probabilities":{"en":0.94,"es":0.02,"mixed":0.02,"unknown":0.02}}}}`))
	}))
	defer server.Close()
	d := LayaDecisionEngine{URL: server.URL, Timeout: time.Second}
	result, err := d.Decide(context.Background(), domain.DecisionRequest{Stage: "classify", Segment: domain.TranscriptSegment{Text: "Hello world"}})
	if err != nil || result.Language != "en" || result.LanguageConfidence != 0.94 || !result.RequiresTranslation {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestInvalidOrUnavailableLayaReturnsError(t *testing.T) {
	for _, body := range []string{`{}`, `{"answers":{"language":{"choice":"fr","probabilities":{"fr":1}}}}`, `{"answers":{"language":{"choice":"en","probabilities":{"en":1.1}}}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		_, err := (LayaDecisionEngine{URL: server.URL}).Decide(context.Background(), domain.DecisionRequest{Stage: "classify"})
		server.Close()
		if err == nil {
			t.Fatalf("accepted invalid decision: %s", body)
		}
	}
}

// Opt-in integration test: official model service, never substituted by a fake.
func TestRealMultilingual(t *testing.T) {
	endpoint := os.Getenv("LAYA_TEST_URL")
	if endpoint == "" {
		t.Skip("run ./scripts/dev laya-smoke against the official service")
	}
	d := LayaDecisionEngine{URL: endpoint, Timeout: 5 * time.Second}
	for _, tc := range []struct{ text, language string }{
		{"Hola, ¿cómo estamos? ¿Todo bien?", "es"},
		{"We are going to talk about neural networks and scientific research.", "en"},
		{"Bueno, ahora we're going to talk about Kubernetes.", "mixed"},
	} {
		start := time.Now()
		got, err := d.Decide(context.Background(), domain.DecisionRequest{Stage: "classify", Segment: domain.TranscriptSegment{Text: tc.text}})
		t.Logf("expected=%s language=%s probability=%.4f elapsed=%s", tc.language, got.Language, got.LanguageConfidence, time.Since(start))
		if err != nil || got.Language != tc.language {
			t.Errorf("classification: %+v err=%v", got, err)
		}
	}
}
func TestTimeoutAndHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("unused") == "" {
			time.Sleep(30 * time.Millisecond)
		}
		w.WriteHeader(503)
	}))
	defer server.Close()
	for _, timeout := range []time.Duration{time.Millisecond, time.Second} {
		_, err := (LayaDecisionEngine{URL: server.URL, Timeout: timeout}).Decide(context.Background(), domain.DecisionRequest{Stage: "classify"})
		if err == nil {
			t.Fatal("expected failure")
		}
	}
}
