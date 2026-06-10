// Package client wraps the SteadyCron REST API with authentication,
// User-Agent, and 429/5xx-aware retry with exponential backoff + jitter.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEndpoint = "https://api.steadycron.com"
	maxRetries      = 5
	baseBackoff     = 500 * time.Millisecond
	maxBackoff      = 30 * time.Second
)

// Client is a thin, authenticated HTTP client for the SteadyCron API.
type Client struct {
	endpoint   string
	apiKey     string
	userAgent  string
	httpClient *http.Client
}

// New creates a Client with the given endpoint, API key, and provider version.
// endpoint may be empty, in which case defaultEndpoint is used.
func New(endpoint, apiKey, version string) *Client {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	endpoint = strings.TrimRight(endpoint, "/")
	return &Client{
		endpoint:  endpoint,
		apiKey:    apiKey,
		userAgent: "terraform-provider-steadycron/" + version,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ─── Generic request helpers ────────────────────────────────────────────────

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	var (
		attempt int
		backoff = baseBackoff
	)
	for {
		attempt++
		req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bodyReader)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("User-Agent", c.userAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("executing request: %w", err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Retry on 429 and 5xx (up to maxRetries).
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt <= maxRetries {
			wait := retryAfter(resp, backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
			// reset bodyReader for retries
			if body != nil {
				b, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(b)
			}
			backoff = min(time.Duration(float64(backoff)*1.5)+jitter(100*time.Millisecond), maxBackoff)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return &APIError{StatusCode: resp.StatusCode, Code: "unauthorized", Message: "invalid or expired API key — check STEADYCRON_API_KEY"}
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return &APIError{StatusCode: resp.StatusCode, Code: "rate_limited", Message: "API rate limit exceeded after retries; consider reducing parallelism"}
		}
		if resp.StatusCode >= 400 {
			var apiErr APIError
			apiErr.StatusCode = resp.StatusCode
			if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr != nil {
				apiErr.Message = string(respBody)
			}
			return &apiErr
		}

		if out != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
		}
		return nil
	}
}

func retryAfter(resp *http.Response, fallback time.Duration) time.Duration {
	if h := resp.Header.Get("Retry-After"); h != "" {
		if secs, err := strconv.Atoi(h); err == nil {
			return time.Duration(secs)*time.Second + jitter(200*time.Millisecond)
		}
	}
	return fallback + jitter(100*time.Millisecond)
}

func jitter(max time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(max)))
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// ─── Jobs ────────────────────────────────────────────────────────────────────

func (c *Client) ListJobs(ctx context.Context) ([]JobResponse, error) {
	var out jobListResponse
	if err := c.do(ctx, http.MethodGet, "/api/jobs?page=1&pageSize=100", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) CreateJob(ctx context.Context, req UpsertJobRequest) (*JobResponse, error) {
	var out JobResponse
	if err := c.do(ctx, http.MethodPost, "/api/jobs", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetJob(ctx context.Context, id string) (*JobResponse, error) {
	var out JobResponse
	if err := c.do(ctx, http.MethodGet, "/api/jobs/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateJob(ctx context.Context, id string, req UpsertJobRequest) (*JobResponse, error) {
	var out JobResponse
	if err := c.do(ctx, http.MethodPatch, "/api/jobs/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteJob(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/jobs/"+id, nil, nil)
}

// ─── Tags ────────────────────────────────────────────────────────────────────

func (c *Client) ListTags(ctx context.Context) ([]TagResponse, error) {
	var out []TagResponse
	if err := c.do(ctx, http.MethodGet, "/api/tags", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTag fetches a single tag by ID using the list endpoint (no GET-by-ID endpoint exists).
func (c *Client) GetTag(ctx context.Context, id string) (*TagResponse, error) {
	tags, err := c.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tags {
		if tags[i].ID == id {
			return &tags[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Code: "not_found", Message: "tag not found"}
}

func (c *Client) CreateTag(ctx context.Context, req UpsertTagRequest) (*TagResponse, error) {
	var out TagResponse
	if err := c.do(ctx, http.MethodPost, "/api/tags", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateTag(ctx context.Context, id string, req UpsertTagRequest) (*TagResponse, error) {
	var out TagResponse
	if err := c.do(ctx, http.MethodPatch, "/api/tags/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteTag(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/tags/"+id, nil, nil)
}

// ─── Alert Channels ──────────────────────────────────────────────────────────

func (c *Client) ListAlertChannels(ctx context.Context) ([]AlertChannelResponse, error) {
	var out []AlertChannelResponse
	if err := c.do(ctx, http.MethodGet, "/api/alert-channels", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAlertChannel fetches a single channel by ID using the list endpoint (no GET-by-ID endpoint exists).
func (c *Client) GetAlertChannel(ctx context.Context, id string) (*AlertChannelResponse, error) {
	channels, err := c.ListAlertChannels(ctx)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		if channels[i].ID == id {
			return &channels[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Code: "not_found", Message: "alert channel not found"}
}

func (c *Client) CreateAlertChannel(ctx context.Context, req UpsertAlertChannelRequest) (*AlertChannelResponse, error) {
	var out AlertChannelResponse
	if err := c.do(ctx, http.MethodPost, "/api/alert-channels", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateAlertChannel(ctx context.Context, id string, req UpsertAlertChannelRequest) (*AlertChannelResponse, error) {
	var out AlertChannelResponse
	if err := c.do(ctx, http.MethodPatch, "/api/alert-channels/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteAlertChannel(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/alert-channels/"+id, nil, nil)
}

// ─── Alert Rules ─────────────────────────────────────────────────────────────

// CreateAlertRule creates an alert rule nested under the given job.
func (c *Client) CreateAlertRule(ctx context.Context, jobID string, req UpsertAlertRuleRequest) (*AlertRuleResponse, error) {
	var out AlertRuleResponse
	if err := c.do(ctx, http.MethodPost, "/api/jobs/"+jobID+"/alert-rules", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListAlertRules(ctx context.Context, jobID string) ([]AlertRuleResponse, error) {
	var out []AlertRuleResponse
	if err := c.do(ctx, http.MethodGet, "/api/jobs/"+jobID+"/alert-rules", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAlertRule fetches a single rule by ID using the job-scoped list endpoint.
func (c *Client) GetAlertRule(ctx context.Context, jobID, ruleID string) (*AlertRuleResponse, error) {
	rules, err := c.ListAlertRules(ctx, jobID)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].ID == ruleID {
			return &rules[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Code: "not_found", Message: "alert rule not found"}
}

// FindAlertRuleByID scans all jobs to locate a rule by ID.
// Used for ImportState where the job ID is not known upfront.
func (c *Client) FindAlertRuleByID(ctx context.Context, ruleID string) (*AlertRuleResponse, error) {
	jobs, err := c.ListJobs(ctx)
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		rules, err := c.ListAlertRules(ctx, job.ID)
		if err != nil {
			continue
		}
		for i := range rules {
			if rules[i].ID == ruleID {
				return &rules[i], nil
			}
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Code: "not_found", Message: "alert rule not found"}
}

func (c *Client) UpdateAlertRule(ctx context.Context, jobID, ruleID string, req UpsertAlertRuleRequest) (*AlertRuleResponse, error) {
	var out AlertRuleResponse
	if err := c.do(ctx, http.MethodPatch, "/api/jobs/"+jobID+"/alert-rules/"+ruleID, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteAlertRule(ctx context.Context, ruleID string) error {
	return c.do(ctx, http.MethodDelete, "/api/alert-rules/"+ruleID, nil, nil)
}

// ─── Template Variables ──────────────────────────────────────────────────────

func (c *Client) ListTemplateVariables(ctx context.Context) ([]TemplateVariableResponse, error) {
	var out []TemplateVariableResponse
	if err := c.do(ctx, http.MethodGet, "/api/variables", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTemplateVariable fetches a single variable by ID using the list endpoint.
func (c *Client) GetTemplateVariable(ctx context.Context, id string) (*TemplateVariableResponse, error) {
	vars, err := c.ListTemplateVariables(ctx)
	if err != nil {
		return nil, err
	}
	for i := range vars {
		if vars[i].ID == id {
			return &vars[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Code: "not_found", Message: "template variable not found"}
}

func (c *Client) CreateTemplateVariable(ctx context.Context, req UpsertTemplateVariableRequest) (*TemplateVariableResponse, error) {
	var out TemplateVariableResponse
	if err := c.do(ctx, http.MethodPost, "/api/variables", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateTemplateVariable(ctx context.Context, id string, req UpsertTemplateVariableRequest) (*TemplateVariableResponse, error) {
	var out TemplateVariableResponse
	if err := c.do(ctx, http.MethodPatch, "/api/variables/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteTemplateVariable(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/variables/"+id, nil, nil)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// IsNotFound returns true when err is a 404 APIError.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
