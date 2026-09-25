package main

import (
	"context"
	"errors"
	"github.com/munhof/sintonizados/internal/adapters/demo"
	"github.com/munhof/sintonizados/internal/adapters/google"
	httpadapter "github.com/munhof/sintonizados/internal/adapters/http"
	"github.com/munhof/sintonizados/internal/adapters/laya"
	"github.com/munhof/sintonizados/internal/adapters/memory"
	"github.com/munhof/sintonizados/internal/application"
	"github.com/munhof/sintonizados/internal/domain"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func env(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	mode := env("PROVIDER_MODE", "demo")
	token := os.Getenv("OPERATOR_TOKEN")
	if token == "" {
		slog.Error("OPERATOR_TOKEN is required")
		os.Exit(1)
	}
	limit, err := strconv.Atoi(env("MAX_SESSIONS", "16"))
	if err != nil || limit < 1 || limit > 1000 {
		slog.Error("MAX_SESSIONS must be 1..1000")
		os.Exit(1)
	}
	var transcriber domain.Transcriber
	var translator domain.Translator
	switch mode {
	case "demo":
		transcriber = demo.Transcriber{}
		translator = demo.Translator{}
	case "google":
		if os.Getenv("GEMINI_API_KEY") == "" || os.Getenv("GOOGLE_TRANSLATION_API_KEY") == "" {
			slog.Error("Google mode requires GEMINI_API_KEY and GOOGLE_TRANSLATION_API_KEY")
			os.Exit(1)
		}
		transcriber = google.Transcriber{APIKey: os.Getenv("GEMINI_API_KEY"), Model: env("GEMINI_MODEL", "gemini-3.5-transcribe-live")}
		translator = google.Translator{APIKey: os.Getenv("GOOGLE_TRANSLATION_API_KEY"), Client: &http.Client{Timeout: 15 * time.Second}}
	default:
		slog.Error("PROVIDER_MODE must be demo or google")
		os.Exit(1)
	}
	var decision domain.DecisionEngine = domain.DeterministicDecisionEngine{}
	switch env("DECISION_ENGINE", "deterministic") {
	case "deterministic":
	case "laya":
		audience := os.Getenv("LAYA_CLOUD_AUDIENCE")
		var identityTokens laya.IDTokenProvider
		if audience != "" {
			identityTokens = &laya.MetadataIDTokenProvider{}
		}
		decision = laya.LayaDecisionEngine{
			URL: env("LAYA_URL", "http://sintonizados-laya:8000"), APIKey: os.Getenv("LAYA_API_KEY"),
			CloudAudience: audience, IDTokenProvider: identityTokens,
		}
	default:
		slog.Error("DECISION_ENGINE must be deterministic or laya")
		os.Exit(1)
	}
	service := application.New(memory.NewSessionStore(), memory.NewEventBus(), transcriber, translator, decision, mode, limit)
	server := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: httpadapter.New(service, token), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	stopped := make(chan struct{})
	go func() {
		<-ctx.Done()
		service.Close()
		shutdown, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
		close(stopped)
	}()
	slog.Info("server_started", "address", server.Addr, "mode", mode, "max_sessions", limit)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server_failed", "error", err)
		stop()
	}
	<-stopped
}
