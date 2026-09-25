package httpadapter_test

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/munhof/sintonizados/internal/adapters/demo"
	httpadapter "github.com/munhof/sintonizados/internal/adapters/http"
	"github.com/munhof/sintonizados/internal/adapters/memory"
	"github.com/munhof/sintonizados/internal/application"
	"github.com/munhof/sintonizados/internal/domain"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (*application.Service, *httptest.Server) {
	t.Helper()
	s := application.New(memory.NewSessionStore(), memory.NewEventBus(), demo.Transcriber{}, demo.Translator{}, domain.DeterministicDecisionEngine{}, "demo", 4)
	server := httptest.NewServer(httpadapter.New(s, "secret"))
	t.Cleanup(func() { server.Close(); s.Close() })
	return s, server
}
func TestHTTPContractAndAuthorization(t *testing.T) {
	_, server := fixture(t)
	for _, tc := range []struct {
		path, body, token string
		status            int
	}{
		{"/api/sessions", `{"session_id":"a","title":"A","language":"en"}`, "", 401},
		{"/api/sessions", `{"session_id":"a","title":"A","language":"en"}`, "secret", 201},
		{"/api/sessions", `{"session_id":"a","title":"A","language":"en"}`, "secret", 409},
		{"/api/sessions", `{"session_id":"es","title":"ES","language":"es"}`, "secret", 201},
		{"/api/sessions", `{"session_id":"auto","title":"Auto","language":"auto"}`, "secret", 201},
		{"/api/sessions", `{"session_id":"bad id","title":"A","language":"en"}`, "secret", 400},
		{"/api/sessions", `{"session_id":"b","title":"A","language":"fr"}`, "secret", 400},
		{"/api/sessions", `{"session_id":"b","title":"A","language":"en"} {}`, "secret", 400},
	} {
		req, _ := http.NewRequest("POST", server.URL+tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+tc.token)
		req.Header.Set("Content-Type", "application/json")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		if r.StatusCode != tc.status {
			t.Fatalf("%s: %d %s", tc.body, r.StatusCode, body)
		}
	}
	r, err := http.Get(server.URL + "/api/sessions/a")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var v map[string]any
	json.NewDecoder(r.Body).Decode(&v)
	if v["session_id"] != "a" || v["transcription_status"] != "waiting" {
		t.Fatal(v)
	}
}

func TestOperatorPageExposesLiveMicrophoneCapture(t *testing.T) {
	_, server := fixture(t)
	r, err := http.Get(server.URL + "/operator")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Fatalf("operator page: got %d, want %d: %s", r.StatusCode, http.StatusOK, body)
	}
	if !strings.Contains(string(body), "/assets/operator.js") {
		t.Fatalf("operator page does not load the microphone client")
	}

	r, err = http.Get(server.URL + "/assets/operator.js")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	client, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != http.StatusOK || !strings.Contains(string(client), "navigator.mediaDevices.getUserMedia") || !strings.Contains(string(client), "X-Audio-Sequence") {
		t.Fatalf("microphone client: got HTTP %d with %q", r.StatusCode, client)
	}

	r, err = http.Get(server.URL + "/assets/mic-worklet.js")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	worklet, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != http.StatusOK || !strings.Contains(string(worklet), "registerProcessor") || !strings.Contains(string(worklet), "16000") {
		t.Fatalf("microphone worklet: got HTTP %d with %q", r.StatusCode, worklet)
	}
}

func TestOBSOverlayProvidesTransparentLiveSubtitles(t *testing.T) {
	s, server := fixture(t)
	if _, err := s.Create(context.Background(), "charla-obs", "Charla OBS", "en"); err != nil {
		t.Fatal(err)
	}
	r, err := http.Get(server.URL + "/obs/charla-obs")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	page := string(body)
	if r.StatusCode != http.StatusOK {
		t.Fatalf("OBS overlay: got %d, want %d: %s", r.StatusCode, http.StatusOK, page)
	}
	for _, expected := range []string{"background:transparent", "/api/sessions/charla-obs/events", "Original", "English", "Español", "subtitle.english"} {
		if !strings.Contains(page, expected) {
			t.Errorf("OBS overlay missing %q", expected)
		}
	}
	talk, err := http.Get(server.URL + "/talks/charla-obs")
	if err != nil {
		t.Fatal(err)
	}
	defer talk.Body.Close()
	viewer, err := io.ReadAll(talk.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"English", `id="english"`, "s.english"} {
		if !strings.Contains(string(viewer), expected) {
			t.Errorf("talk viewer missing %q", expected)
		}
	}
}

func TestSSEReplayAndSessionIsolation(t *testing.T) {
	s, server := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Create(ctx, "a", "A", "en")
	s.Create(ctx, "b", "B", "en")
	s.Ingest("a", 1, []byte{0, 0})
	s.Ingest("a", 2, []byte{0, 0})
	s.End(ctx, "a")
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/sessions/a/events", nil)
	req.Header.Set("Last-Event-ID", "1")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	scanner := bufio.NewScanner(r.Body)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var sub domain.Subtitle
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &sub); err != nil {
				t.Fatal(err)
			}
			if sub.ID != 2 || sub.SessionID != "a" {
				t.Fatal(line)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing replay")
	}
}
func TestPCMValidation(t *testing.T) {
	s, server := fixture(t)
	s.Create(context.Background(), "a", "A", "en")
	for _, tc := range []struct {
		body string
		seq  string
		want int
	}{{"x", "1", 400}, {"xx", "0", 400}, {strings.Repeat("x", 32002), "1", 413}, {"xx", "1", 202}, {"xx", "1", 409}} {
		req, _ := http.NewRequest("POST", server.URL+"/api/sessions/a/audio", strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer secret")
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("X-Audio-Sequence", tc.seq)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != tc.want {
			t.Fatalf("got %d want %d", r.StatusCode, tc.want)
		}
	}
}
