package httpadapter

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/munhof/sintonizados/internal/application"
	"github.com/munhof/sintonizados/internal/domain"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"time"
)

//go:embed web/*
var assets embed.FS
var pages = template.Must(template.ParseFS(assets, "web/*.html"))

type sessionDTO struct {
	ID                  string    `json:"session_id"`
	Title               string    `json:"title"`
	Language            string    `json:"language"`
	Status              string    `json:"status"`
	TranscriptionStatus string    `json:"transcription_status"`
	Mode                string    `json:"mode"`
	CreatedAt           time.Time `json:"created_at"`
	SubtitleCount       int64     `json:"subtitle_count"`
	Error               string    `json:"error,omitempty"`
}

func dto(s domain.Session) sessionDTO {
	return sessionDTO{s.ID, s.Title, s.Language, s.Status, s.TranscriptionStatus, s.Mode, s.CreatedAt, s.SubtitleCount, s.Error}
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, name, msg string) {
	w.Header().Set("X-Amzn-Errortype", name)
	respond(w, status, map[string]string{"message": msg})
}
func failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		problem(w, 404, "NotFound", err.Error())
	case errors.Is(err, domain.ErrConflict):
		problem(w, 409, "Conflict", err.Error())
	case errors.Is(err, domain.ErrBusy):
		w.Header().Set("Retry-After", "1")
		problem(w, 429, "Busy", err.Error())
	case errors.Is(err, domain.ErrInvalid):
		problem(w, 400, "BadRequest", err.Error())
	default:
		problem(w, 503, "Unavailable", "processing unavailable")
	}
}
func New(s *application.Service, token string) http.Handler {
	mux := http.NewServeMux()
	protect := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
				problem(w, 401, "Unauthorized", "operator bearer token required")
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]string{"status": "ok", "mode": s.Mode})
	})
	mux.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		v := []sessionDTO{}
		for _, x := range s.Store.List() {
			v = append(v, dto(x))
		}
		respond(w, 200, map[string]any{"sessions": v})
	})
	mux.HandleFunc("POST /api/sessions", protect(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			failure(w, domain.ErrInvalid)
			return
		}
		var input struct {
			ID       string `json:"session_id"`
			Title    string `json:"title"`
			Language string `json:"language"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		d.DisallowUnknownFields()
		if err := d.Decode(&input); err != nil {
			failure(w, domain.ErrInvalid)
			return
		}
		if d.Decode(&struct{}{}) != io.EOF {
			failure(w, domain.ErrInvalid)
			return
		}
		v, err := s.Create(r.Context(), input.ID, input.Title, input.Language)
		if err != nil {
			failure(w, err)
			return
		}
		respond(w, 201, dto(v))
	}))
	mux.HandleFunc("GET /api/sessions/{session_id}", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Snapshot(r.PathValue("session_id"))
		if err != nil {
			failure(w, err)
			return
		}
		respond(w, 200, dto(v.Session))
	})
	mux.HandleFunc("POST /api/sessions/{session_id}/audio", protect(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			failure(w, domain.ErrInvalid)
			return
		}
		n, err := strconv.ParseInt(r.Header.Get("X-Audio-Sequence"), 10, 64)
		if err != nil {
			failure(w, domain.ErrInvalid)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32000))
		if err != nil {
			var max *http.MaxBytesError
			if errors.As(err, &max) {
				problem(w, 413, "TooLarge", "maximum chunk size is 32000 bytes")
			} else {
				failure(w, domain.ErrInvalid)
			}
			return
		}
		if err := s.Ingest(r.PathValue("session_id"), n, body); err != nil {
			failure(w, err)
			return
		}
		respond(w, 202, map[string]int64{"sequence": n})
	}))
	mux.HandleFunc("POST /api/sessions/{session_id}/end", protect(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		sid := r.PathValue("session_id")
		if err := s.End(ctx, sid); err != nil {
			failure(w, err)
			return
		}
		v, _ := s.Snapshot(sid)
		respond(w, 200, dto(v.Session))
	}))
	mux.HandleFunc("GET /api/sessions/{session_id}/subtitles", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Snapshot(r.PathValue("session_id"))
		if err != nil {
			failure(w, err)
			return
		}
		respond(w, 200, map[string]any{"subtitles": v.Subtitles})
	})
	mux.HandleFunc("GET /api/sessions/{session_id}/events", func(w http.ResponseWriter, r *http.Request) { stream(s, w, r) })
	mux.HandleFunc("GET /metrics", protect(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintln(w, "# HELP sintonizados_subtitles_total Published subtitles since process start.\n# TYPE sintonizados_subtitles_total counter")
		for _, v := range s.Store.List() {
			fmt.Fprintf(w, "sintonizados_subtitles_total{session_id=%q} %d\n", v.ID, v.SubtitleCount)
		}
		for _, metric := range []string{"transcription", "translation", "end_to_end"} {
			fmt.Fprintf(w, "# HELP sintonizados_%s_seconds Most recent subtitle latency in seconds.\n# TYPE sintonizados_%s_seconds gauge\n", metric, metric)
			for _, v := range s.Store.List() {
				x, _ := s.Snapshot(v.ID)
				if len(x.Subtitles) == 0 {
					continue
				}
				sub := x.Subtitles[len(x.Subtitles)-1]
				ms := sub.EndToEndMS
				if metric == "transcription" {
					ms = sub.TranscriptionMS
				}
				if metric == "translation" {
					ms = sub.TranslationMS
				}
				fmt.Fprintf(w, "sintonizados_%s_seconds{session_id=%q} %g\n", metric, v.ID, ms/1000)
			}
		}
	}))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pages.ExecuteTemplate(w, "home.html", map[string]any{"Sessions": s.Store.List(), "Mode": s.Mode})
	})
	mux.HandleFunc("GET /operator", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pages.ExecuteTemplate(w, "operator.html", map[string]any{"Mode": s.Mode})
	})
	mux.HandleFunc("GET /assets/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name != "operator.js" && name != "mic-worklet.js" {
			http.NotFound(w, r)
			return
		}
		body, err := assets.ReadFile("web/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(body)
	})
	mux.HandleFunc("GET /talks/{session_id}", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Snapshot(r.PathValue("session_id"))
		if err != nil {
			failure(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pages.ExecuteTemplate(w, "talk.html", v.Session)
	})
	mux.HandleFunc("GET /obs/{session_id}", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Snapshot(r.PathValue("session_id"))
		if err != nil {
			failure(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		pages.ExecuteTemplate(w, "obs.html", v.Session)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'")
		mux.ServeHTTP(w, r)
	})
}
func stream(s *application.Service, w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("session_id")
	snap, err := s.Snapshot(sid)
	if err != nil {
		failure(w, err)
		return
	}
	var last int64
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		last, err = strconv.ParseInt(v, 10, 64)
		if err != nil || last < 0 {
			failure(w, domain.ErrInvalid)
			return
		}
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	ctl := http.NewResponseController(w)
	send := func(kind, seq string, v any) error {
		ctl.SetWriteDeadline(time.Now().Add(5 * time.Second))
		b, _ := json.Marshal(v)
		if seq != "" {
			if _, err := fmt.Fprintf(w, "id: %s\n", seq); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, b); err != nil {
			return err
		}
		return ctl.Flush()
	}
	// Poll durable snapshots: observer drops cannot lose subtitles within retention.
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	partial := ""
	for {
		if last > snap.Session.SubtitleCount || (len(snap.Subtitles) > 0 && last < snap.Subtitles[0].ID-1) {
			last = 0
			if err := send("reset", "0", map[string]string{"reason": "history outside retention; replaying retained subtitles"}); err != nil {
				return
			}
		}
		for _, sub := range snap.Subtitles {
			if sub.ID > last {
				if err := send("subtitle", strconv.FormatInt(sub.ID, 10), sub); err != nil {
					return
				}
				last = sub.ID
			}
		}
		if snap.Partial != partial {
			partial = snap.Partial
			if err := send("partial", "", map[string]string{"text": partial}); err != nil {
				return
			}
		}
		if snap.Session.Status != "active" {
			send("session", "", dto(snap.Session))
			return
		}
		// Flush headers even before the first subtitle arrives.
		ctl.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := ctl.Flush(); err != nil {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			ctl.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := io.WriteString(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if err := ctl.Flush(); err != nil {
				return
			}
		case <-tick.C:
		}
		snap, err = s.Snapshot(sid)
		if err != nil {
			return
		}
	}
}
