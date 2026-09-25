package laya

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const cloudRunIdentityEndpoint = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity"

type cachedIDToken struct {
	value   string
	expires time.Time
}

// MetadataIDTokenProvider uses the Cloud Run metadata server and caches tokens
// per audience until one minute before the expiry in the signed token payload.
type MetadataIDTokenProvider struct {
	Client   *http.Client
	Endpoint string

	mu     sync.Mutex
	tokens map[string]cachedIDToken
}

func (p *MetadataIDTokenProvider) IDToken(ctx context.Context, audience string) (string, error) {
	if audience == "" {
		return "", errors.New("identity token audience is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if cached, ok := p.tokens[audience]; ok && cached.expires.After(now.Add(time.Minute)) {
		return cached.value, nil
	}

	endpoint := p.Endpoint
	if endpoint == "" {
		endpoint = cloudRunIdentityEndpoint
	}
	identityURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	query := identityURL.Query()
	query.Set("audience", audience)
	query.Set("format", "full")
	identityURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, identityURL.String(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Metadata-Flavor", "Google")
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("metadata server rejected identity token request")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 32<<10))
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	expires, err := tokenExpiry(token)
	if err != nil || !expires.After(now.Add(time.Minute)) {
		return "", errors.New("metadata server returned an invalid or expiring identity token")
	}
	if p.tokens == nil {
		p.tokens = make(map[string]cachedIDToken)
	}
	p.tokens[audience] = cachedIDToken{value: token, expires: expires}
	return token, nil
}

func tokenExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, errors.New("malformed identity token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	var claims struct {
		Expires int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Expires == 0 {
		return time.Time{}, errors.New("identity token has no expiry")
	}
	return time.Unix(claims.Expires, 0), nil
}
