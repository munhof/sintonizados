package laya

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/munhof/sintonizados/internal/domain"
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

type staticIdentityToken string

func (s staticIdentityToken) IDToken(context.Context, string) (string, error) {
	return string(s), nil
}

func TestCloudRunIdentityAndLayaAPIKeyAreBothSent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Serverless-Authorization"); got != "Bearer google-id-token" {
			t.Errorf("Cloud Run identity header = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer laya-api-key" {
			t.Errorf("Laya API key header = %q", got)
		}
		w.Write([]byte(`{"answers":{"language":{"choice":"en","probabilities":{"en":0.94}}}}`))
	}))
	defer server.Close()

	d := LayaDecisionEngine{
		URL:             server.URL,
		APIKey:          "laya-api-key",
		CloudAudience:   "https://sintonizados-laya.example.run.app",
		IDTokenProvider: staticIdentityToken("google-id-token"),
	}
	_, err := d.Decide(context.Background(), domain.DecisionRequest{Stage: "classify", Segment: domain.TranscriptSegment{Text: "hello"}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetadataIDTokenProviderUsesAudienceAndCachesUntilExpiry(t *testing.T) {
	var calls atomic.Int32
	expires := time.Now().Add(10 * time.Minute).Unix()
	payload, err := json.Marshal(map[string]int64{"exp": expires})
	if err != nil {
		t.Fatal(err)
	}
	token := "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/identity" || r.URL.Query().Get("audience") != "https://laya.example.run.app" || r.URL.Query().Get("format") != "full" {
			t.Errorf("identity request URL = %s", r.URL.String())
		}
		if r.Header.Get("Metadata-Flavor") != "Google" {
			t.Errorf("metadata header = %q", r.Header.Get("Metadata-Flavor"))
		}
		w.Write([]byte(token))
	}))
	defer server.Close()

	provider := &MetadataIDTokenProvider{Endpoint: server.URL + "/identity", Client: server.Client()}
	for range 2 {
		got, err := provider.IDToken(context.Background(), "https://laya.example.run.app")
		if err != nil || got != token {
			t.Fatalf("token = %q, err = %v", got, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("metadata calls = %d, want one cached request", calls.Load())
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
