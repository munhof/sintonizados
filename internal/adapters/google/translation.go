package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/munhof/sintonizados/internal/domain"
	"html"
	"io"
	"net/http"
)

type Translator struct {
	APIKey, Endpoint string
	Client           *http.Client
}

func (t Translator) Translate(ctx context.Context, r domain.TranslationRequest) (domain.Translation, error) {
	endpoint := t.Endpoint
	if endpoint == "" {
		endpoint = "https://translation.googleapis.com/language/translate/v2"
	}
	body, _ := json.Marshal(map[string]string{"q": r.Text, "source": r.Source, "target": r.Target, "format": "text"})
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return domain.Translation{}, domain.ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", t.APIKey)
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return domain.Translation{}, fmt.Errorf("translation transport failed: %w", domain.ErrUnavailable)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return domain.Translation{}, fmt.Errorf("translation HTTP %d: %w", resp.StatusCode, domain.ErrUnavailable)
	}
	var out struct {
		Data struct {
			Translations []struct {
				Text string `json:"translatedText"`
			} `json:"translations"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil || len(out.Data.Translations) != 1 || out.Data.Translations[0].Text == "" {
		return domain.Translation{}, domain.ErrUnavailable
	}
	return domain.Translation{Text: html.UnescapeString(out.Data.Translations[0].Text), Provider: "google-translation-basic"}, nil
}
