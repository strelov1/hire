package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPClient is the narrow transport an adapter needs: fetch a URL and decode its
// JSON (or XML) body into v. PostJSON sends a JSON request body (Workday's listing
// API is POST-only). Adapters depend on this interface so tests inject a fake and
// never touch the network; the real client is Client below.
type HTTPClient interface {
	GetJSON(ctx context.Context, url string, v any) error
	GetXML(ctx context.Context, url string, v any) error
	PostJSON(ctx context.Context, url string, body, v any) error
}

// Client is the real HTTPClient: a timeout-bounded GET with a project User-Agent and
// a bounded retry-with-backoff on transient (5xx / network) failures. A 4xx is not
// retried — it will not recover on its own.
type Client struct {
	httpClient *http.Client
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

// NewClient builds the default ingest HTTP client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		userAgent:  "freehire/0.1 (+https://freehire.dev)",
		maxRetries: 2,
		retryDelay: 500 * time.Millisecond,
	}
}

// GetJSON fetches url and decodes its JSON body into v.
func (c *Client) GetJSON(ctx context.Context, url string, v any) error {
	return c.do(ctx, http.MethodGet, url, nil, "application/json", func(r io.Reader) error {
		return json.NewDecoder(r).Decode(v)
	})
}

// GetXML fetches url and decodes its XML body into v (used by adapters whose platform
// publishes an XML feed, e.g. Personio).
func (c *Client) GetXML(ctx context.Context, url string, v any) error {
	return c.do(ctx, http.MethodGet, url, nil, "application/xml", func(r io.Reader) error {
		return xml.NewDecoder(r).Decode(v)
	})
}

// PostJSON marshals body to JSON, POSTs it to url, and decodes the JSON response into
// v (used by adapters whose listing API is POST-only, e.g. Workday).
func (c *Client) PostJSON(ctx context.Context, url string, body, v any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("sources: marshal request %s: %w", url, err)
	}
	return c.do(ctx, http.MethodPost, url, payload, "application/json", func(r io.Reader) error {
		return json.NewDecoder(r).Decode(v)
	})
}

// do issues an HTTP request (optionally with a JSON body) and applies decode to a
// successful response body, retrying transient failures (5xx / network / 429 rate
// limit) up to maxRetries times. The backoff is a fixed delay, except a 429 honors
// the server's Retry-After hint — busy ATS APIs (SmartRecruiters) throttle by IP
// under the concurrent crawl and recover on a brief wait. Other 4xx are not retried.
// A non-nil body is re-sent on each attempt.
func (c *Client) do(ctx context.Context, method, url string, body []byte, accept string, decode func(io.Reader) error) error {
	var lastErr error
	delay := c.retryDelay
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 && delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		delay = c.retryDelay

		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return fmt.Errorf("sources: build request %s: %w", url, err)
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		req.Header.Set("Accept", accept)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue // network error — transient
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			err := decode(resp.Body)
			resp.Body.Close()
			if err != nil {
				return fmt.Errorf("sources: decode %s: %w", url, err)
			}
			return nil
		case resp.StatusCode == http.StatusTooManyRequests:
			delay = retryAfter(resp, c.retryDelay) // honor the rate-limit hint
			resp.Body.Close()
			lastErr = fmt.Errorf("sources: GET %s: status %d", url, resp.StatusCode)
			continue // rate limited — transient
		case resp.StatusCode >= 500:
			resp.Body.Close()
			lastErr = fmt.Errorf("sources: GET %s: status %d", url, resp.StatusCode)
			continue // server error — transient
		default:
			resp.Body.Close()
			return fmt.Errorf("sources: GET %s: status %d", url, resp.StatusCode)
		}
	}
	return fmt.Errorf("sources: GET %s failed after %d attempts: %w", url, c.maxRetries+1, lastErr)
}

// retryAfter is how long to wait before retrying a 429, honoring the response's
// Retry-After header (delta-seconds) when present and sane, else the fallback. It
// is capped so one rate-limited board cannot stall the whole crawl.
func retryAfter(resp *http.Response, fallback time.Duration) time.Duration {
	const max = 30 * time.Second
	if v := strings.TrimSpace(resp.Header.Get("Retry-After")); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
			d := time.Duration(secs) * time.Second
			if d > max {
				return max
			}
			return d
		}
	}
	return fallback
}
