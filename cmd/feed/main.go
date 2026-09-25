// feed plays raw PCM16 mono 16 kHz at real-time speed into one session.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	base := flag.String("url", "http://127.0.0.1:8080", "gateway URL")
	sid := flag.String("session", "", "new stable session ID")
	title := flag.String("title", "Charla", "talk title")
	file := flag.String("file", "", "raw PCM16 LE mono 16kHz file")
	demo := flag.Bool("demo", false, "send 30 silent chunks (scripted demo mode only)")
	flag.Parse()
	if *sid == "" || (*file == "" && !*demo) || (*file != "" && *demo) {
		return fmt.Errorf("use -session ID and exactly one of -file /data/audio.pcm or -demo")
	}
	token := os.Getenv("OPERATOR_TOKEN")
	if token == "" {
		return fmt.Errorf("OPERATOR_TOKEN is required")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	*base = strings.TrimRight(*base, "/")
	request := func(path, content string, body []byte, seq int64) (int, []byte, error) {
		req, err := http.NewRequest("POST", *base+path, bytes.NewReader(body))
		if err != nil {
			return 0, nil, err
		}
		req.Header.Set("Content-Type", content)
		req.Header.Set("Authorization", "Bearer "+token)
		if seq > 0 {
			req.Header.Set("X-Audio-Sequence", fmt.Sprint(seq))
		}
		r, err := client.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer r.Body.Close()
		b, err := io.ReadAll(io.LimitReader(r.Body, 8192))
		return r.StatusCode, b, err
	}
	if *demo {
		r, err := client.Get(*base + "/health")
		if err != nil {
			return err
		}
		var h struct {
			Mode string `json:"mode"`
		}
		err = json.NewDecoder(r.Body).Decode(&h)
		r.Body.Close()
		if err != nil || h.Mode != "demo" {
			return fmt.Errorf("-demo requires gateway PROVIDER_MODE=demo")
		}
	}
	var source io.Reader
	if *demo {
		source = bytes.NewReader(make([]byte, 3200*30))
	} else {
		f, err := os.Open(*file)
		if err != nil {
			return err
		}
		defer f.Close()
		source = f
	}
	body, _ := json.Marshal(map[string]string{"session_id": *sid, "title": *title, "language": "en"})
	status, b, err := request("/api/sessions", "application/json", body, 0)
	if err != nil {
		return err
	}
	if status != 201 {
		return fmt.Errorf("create: HTTP %d %s", status, b)
	}
	path := "/api/sessions/" + *sid
	buffer := make([]byte, 3200)
	var sequence int64
	for {
		n, readerr := io.ReadFull(source, buffer)
		if n == 0 && readerr == io.EOF {
			break
		}
		if readerr != nil && readerr != io.ErrUnexpectedEOF {
			return readerr
		}
		if n%2 != 0 {
			return fmt.Errorf("PCM file must contain an even number of bytes")
		}
		sequence++
		accepted := false
		for attempt := 0; attempt < 10; attempt++ {
			status, b, err = request(path+"/audio", "application/octet-stream", buffer[:n], sequence)
			if err != nil {
				return fmt.Errorf("ingest uncertain delivery; inspect session before retry: %w", err)
			}
			if status == 202 {
				accepted = true
				break
			}
			if status != 429 {
				return fmt.Errorf("ingest: HTTP %d %s", status, b)
			}
			time.Sleep(200 * time.Millisecond)
		}
		if !accepted {
			return fmt.Errorf("audio queue remained full")
		}
		time.Sleep(time.Duration(n) * time.Second / 32000)
	}
	status, b, err = request(path+"/end", "application/json", nil, 0)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("end: HTTP %d %s", status, b)
	}
	fmt.Printf("Session %s ended: %d chunks sent\n", *sid, sequence)
	return nil
}
