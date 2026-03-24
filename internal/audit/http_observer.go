package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// HTTPObserver - отправляет события аудита во внешний HTTP endpoint.
type HTTPObserver struct {
	targetURL string
	client    *http.Client
}

// NewHTTPObserver - создает HTTP-приемник аудита.
func NewHTTPObserver(targetURL string, client *http.Client) (*HTTPObserver, error) {
	if _, err := url.ParseRequestURI(targetURL); err != nil {
		return nil, fmt.Errorf("invalid audit url: %w", err)
	}

	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	return &HTTPObserver{
		targetURL: targetURL,
		client:    client,
	}, nil
}

// Notify - отправляет событие аудита HTTP POST-запросом.
func (o *HTTPObserver) Notify(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	// Внешний приёмник получает то же событие, но уже обычным HTTP POST.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.targetURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("post audit event: %w", err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("read audit response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("post audit event: unexpected status %s", resp.Status)
	}

	return nil
}
