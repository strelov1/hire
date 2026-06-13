package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientGetJSONDecodesAndSendsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"acme"}`))
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), userAgent: "freehire-test"}

	var out struct {
		Name string `json:"name"`
	}
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if out.Name != "acme" {
		t.Errorf("decoded name = %q, want %q", out.Name, "acme")
	}
	if gotUA != "freehire-test" {
		t.Errorf("User-Agent = %q, want %q", gotUA, "freehire-test")
	}
}

func TestClientGetXMLDecodesAndSendsXMLAccept(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<root><name>acme</name></root>`))
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client()}

	var out struct {
		Name string `xml:"name"`
	}
	if err := c.GetXML(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetXML: %v", err)
	}
	if out.Name != "acme" {
		t.Errorf("decoded name = %q, want %q", out.Name, "acme")
	}
	if !strings.Contains(gotAccept, "xml") {
		t.Errorf("Accept = %q, want it to request xml", gotAccept)
	}
}

func TestClientGetJSONRetriesOnServerError(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), maxRetries: 2}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if !out.OK {
		t.Error("expected ok=true after retry")
	}
}

func TestClientGetJSONErrorsOnClientError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client()}

	var out map[string]any
	if err := c.GetJSON(context.Background(), srv.URL, &out); err == nil {
		t.Error("expected error on 404, got nil")
	}
}

func TestClientRetriesOn429ThenSucceeds(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.Header().Set("Retry-After", "0") // ask for an immediate retry
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &Client{httpClient: srv.Client(), maxRetries: 2}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if !out.OK {
		t.Error("expected ok=true after a 429 retry")
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2 (429 then 200)", attempts)
	}
}
