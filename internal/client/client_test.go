package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

func newTestServer(t *testing.T, handler http.Handler) (*client.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := client.New(srv.URL, "sc_test", "test")
	return c, srv
}

func TestGetJob_OK(t *testing.T) {
	c, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer sc_test" {
			t.Errorf("missing auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.JobResponse{
			ID:   "abc-123",
			Kind: "http",
			Name: "my-job",
		})
	}))

	job, err := c.GetJob(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID != "abc-123" {
		t.Errorf("expected ID abc-123, got %s", job.ID)
	}
}

func TestGetJob_NotFound(t *testing.T) {
	c, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"code": "not_found", "message": "job not found"})
	}))

	_, err := c.GetJob(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected not-found error, got %T: %v", err, err)
	}
}

func TestClient_Unauthorized(t *testing.T) {
	c, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	_, err := c.GetJob(context.Background(), "some-id")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
}

func TestClient_RetryOn429(t *testing.T) {
	calls := 0
	c, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.JobResponse{ID: "ok"})
	}))

	job, err := c.GetJob(context.Background(), "ok")
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if job.ID != "ok" {
		t.Errorf("expected ID ok, got %s", job.ID)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 calls (1 retry), got %d", calls)
	}
}

func TestClient_UserAgent(t *testing.T) {
	c, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua == "" || !contains(ua, "terraform-provider-steadycron") {
			t.Errorf("unexpected User-Agent: %q", ua)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.JobResponse{ID: "x"})
	}))
	c.GetJob(context.Background(), "x") //nolint:errcheck
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
