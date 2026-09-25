// smoke exercises the HTTP server with two simultaneous scripted sources.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/munhof/sintonizados/internal/adapters/demo"
	httpadapter "github.com/munhof/sintonizados/internal/adapters/http"
	"github.com/munhof/sintonizados/internal/adapters/memory"
	"github.com/munhof/sintonizados/internal/application"
	"github.com/munhof/sintonizados/internal/domain"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	service := application.New(memory.NewSessionStore(), memory.NewEventBus(), demo.Transcriber{}, demo.Translator{}, domain.DeterministicDecisionEngine{}, "demo", 4)
	defer service.Close()
	server := httptest.NewServer(httpadapter.New(service, "smoke"))
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	post := func(path string, body []byte, seq int) error {
		req, _ := http.NewRequest("POST", server.URL+path, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer smoke")
		req.Header.Set("Content-Type", "application/json")
		if seq > 0 {
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set("X-Audio-Sequence", fmt.Sprint(seq))
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return fmt.Errorf("%s: %d %s", path, resp.StatusCode, b)
		}
		return nil
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, sid := range []string{"charla-a", "charla-b"} {
		wg.Add(1)
		go func(sid string) {
			defer wg.Done()
			b, _ := json.Marshal(map[string]string{"session_id": sid, "title": sid, "language": "en"})
			if err := post("/api/sessions", b, 0); err != nil {
				errs <- err
				return
			}
			for n := 1; n <= 10; n++ {
				if err := post("/api/sessions/"+sid+"/audio", make([]byte, 3200), n); err != nil {
					errs <- err
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
			errs <- post("/api/sessions/"+sid+"/end", nil, 0)
		}(sid)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	for _, sid := range []string{"charla-a", "charla-b"} {
		r, err := client.Get(server.URL + "/api/sessions/" + sid + "/subtitles")
		if err != nil {
			return err
		}
		var v struct {
			Subtitles []domain.Subtitle `json:"subtitles"`
		}
		err = json.NewDecoder(r.Body).Decode(&v)
		r.Body.Close()
		if err != nil {
			return err
		}
		if len(v.Subtitles) != 10 {
			return fmt.Errorf("%s: expected 10 subtitles", sid)
		}
		for _, sub := range v.Subtitles {
			if sub.SessionID != sid || sub.Spanish == "" || sub.EndToEndMS < 0 {
				return fmt.Errorf("invalid subtitle")
			}
		}
	}
	if err := service.End(context.Background(), "charla-a"); err != nil {
		return err
	}
	fmt.Println("PASS: two simultaneous HTTP audio sessions, 20 translated demo subtitles, isolated history and latency.")
	return nil
}
