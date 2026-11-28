package onlineconf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const MaxSnapshotBytes int64 = 4 << 20

type SnapshotSource interface {
	Fetch(context.Context) (*Snapshot, error)
}

type HTTPClient struct {
	client *http.Client
	url    string
	token  string
}

func NewHTTPClient(client *http.Client, baseURL, snapshotPath, token string) (*HTTPClient, error) {
	if client == nil {
		return nil, errors.New("onlineconf http client is nil")
	}
	base, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid onlineconf url %q", baseURL)
	}
	if snapshotPath == "" || snapshotPath[0] != '/' {
		return nil, errors.New("onlineconf snapshot path must start with /")
	}
	return &HTTPClient{client: client, url: base.String() + snapshotPath, token: token}, nil
}

func (c *HTTPClient) Fetch(ctx context.Context) (*Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("build snapshot request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch snapshot: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("fetch snapshot: unexpected status %d", resp.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, fmt.Errorf("fetch snapshot: invalid content type %q", resp.Header.Get("Content-Type"))
	}

	limited := io.LimitReader(resp.Body, MaxSnapshotBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	if int64(len(body)) > MaxSnapshotBytes {
		return nil, fmt.Errorf("snapshot exceeds %d bytes", MaxSnapshotBytes)
	}

	var raw *struct {
		snapshotDTO
		Revision *int64 `json:"revision"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, errors.New("decode snapshot: trailing JSON data")
	}
	if raw == nil || raw.Revision == nil || raw.GeneratedAt.IsZero() || raw.Configs == nil || raw.Flags == nil {
		return nil, errors.New("snapshot must contain generated_at, configs and flags")
	}
	raw.snapshotDTO.Revision = *raw.Revision
	return buildSnapshot(raw.snapshotDTO)
}

type snapshotDTO struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Revision    int64       `json:"revision"`
	Configs     []configDTO `json:"configs"`
	Flags       []flagDTO   `json:"flags"`
}

type configDTO struct {
	Scope string          `json:"scope"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type flagDTO struct {
	ID              int64           `json:"id"`
	Salt            string          `json:"salt"`
	Scope           string          `json:"scope"`
	Key             string          `json:"key"`
	Enabled         bool            `json:"enabled"`
	Rollout         *int            `json:"rollout_percent"`
	IncludedUserIDs []domain.UserID `json:"force_on_user_ids"`
	ExcludedUserIDs []domain.UserID `json:"force_off_user_ids"`
}
