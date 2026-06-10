package client

import "encoding/json"

// ─── Job ─────────────────────────────────────────────────────────────────────

// UpsertJobRequest is used for both POST /api/jobs and PATCH /api/jobs/{id}.
type UpsertJobRequest struct {
	Kind        string `json:"kind"` // "http" | "heartbeat"
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// Schedule — exactly one must be set.
	ScheduleKind    string `json:"schedule_kind"`         // "cron" | "interval"
	CronExpression  string `json:"cron_expression,omitempty"`
	IntervalSeconds *int64 `json:"interval_seconds,omitempty"`
	Timezone        string `json:"timezone,omitempty"`

	// Heartbeat-specific
	GraceSeconds          *int64 `json:"grace_seconds,omitempty"`
	StuckRunDetection     *bool  `json:"stuck_run_detection,omitempty"`
	MaxRunDurationSeconds *int64 `json:"max_run_duration_seconds,omitempty"`

	// HTTP-specific
	Method              string            `json:"method,omitempty"`
	URL                 string            `json:"http_url,omitempty"`
	Headers             map[string]string `json:"headers,omitempty"`
	Body                string            `json:"body,omitempty"`
	TimeoutSeconds      *int64            `json:"timeout_seconds,omitempty"`
	MaxRetries          *int64            `json:"max_retries,omitempty"`
	RetryBackoffSeconds *int64            `json:"retry_backoff_seconds,omitempty"`
	SkipIfRunning       *bool             `json:"skip_if_running,omitempty"`

	// Tags — list of tag UUIDs.
	Tags []string `json:"tags,omitempty"`
}

// JobResponse is returned by GET /api/jobs/{id} and POST /api/jobs.
type JobResponse struct {
	ID          string  `json:"id"`
	AccountID   string  `json:"account_id"`
	Kind        string  `json:"kind"`
	Name        string  `json:"name"`
	Description *string `json:"description"`

	ScheduleKind    string  `json:"schedule_kind"`
	CronExpression  *string `json:"cron_expression"`
	IntervalSeconds *int64  `json:"interval_seconds"`
	Timezone        string  `json:"timezone"`

	// Heartbeat-specific
	GraceSeconds          int64  `json:"grace_seconds"`
	StuckRunDetection     bool   `json:"stuck_run_detection"`
	MaxRunDurationSeconds int64  `json:"max_run_duration_seconds"`
	PingURL               string `json:"ping_url"`
	Token                 string `json:"token"`
	LastSuccessAt         string `json:"last_success_at"`
	LastStartAt           string `json:"last_start_at"`
	LastFailAt            string `json:"last_fail_at"`

	// HTTP-specific
	Method              *string           `json:"method"`
	URL                 *string           `json:"http_url"`
	Headers             map[string]string `json:"headers"`
	Body                *string           `json:"body"`
	TimeoutSeconds      *int64            `json:"timeout_seconds"`
	MaxRetries          *int64            `json:"max_retries"`
	RetryBackoffSeconds *int64            `json:"retry_backoff_seconds"`
	SkipIfRunning       bool              `json:"skip_if_running"`

	// Derived status (read-only)
	Status     *string `json:"status"`
	NextFireAt *string `json:"next_fire_at"`
	LastFireAt *string `json:"last_fire_at"`
	BadgeURL   string  `json:"badge_url"`

	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// ─── Tag ─────────────────────────────────────────────────────────────────────

type UpsertTagRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Color string `json:"color,omitempty"`
}

type TagResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Color     string `json:"color"`
	CreatedAt string `json:"created_at"`
}

// ─── Alert Channel ───────────────────────────────────────────────────────────

type UpsertAlertChannelRequest struct {
	Name   string          `json:"name"`
	Kind   string          `json:"kind"` // email|slack|discord|webhook|telegram
	Config json.RawMessage `json:"config"`
}

// Per-kind config shapes (used when building UpsertAlertChannelRequest.Config).

type EmailConfig struct {
	To string `json:"to"`
}

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type WebhookConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type AlertChannelResponse struct {
	ID        string          `json:"id"`
	AccountID string          `json:"account_id"`
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Config    json.RawMessage `json:"config"`
	CreatedAt string          `json:"created_at"`
}

// ─── Alert Rule ──────────────────────────────────────────────────────────────

type UpsertAlertRuleRequest struct {
	JobID              string  `json:"job_id"`
	ChannelID          string  `json:"channel_id"`
	Trigger            string  `json:"trigger"`
	Threshold          *int64  `json:"threshold,omitempty"`
	Severity           string  `json:"severity,omitempty"`
	DedupWindowSeconds *int64  `json:"dedup_window_seconds,omitempty"`
	Params             *Params `json:"params,omitempty"`
}

type Params struct {
	Factor              *float64 `json:"factor,omitempty"`
	MinBaselineSamples  *int64   `json:"min_baseline_samples,omitempty"`
}

type AlertRuleResponse struct {
	ID                 string  `json:"id"`
	JobID              string  `json:"job_id"`
	ChannelID          string  `json:"channel_id"`
	Trigger            string  `json:"trigger"`
	Threshold          *int64  `json:"threshold"`
	Severity           string  `json:"severity"`
	DedupWindowSeconds int64   `json:"dedup_window_seconds"`
	Params             *Params `json:"params"`
	CreatedAt          string  `json:"created_at"`
}

// ─── Template Variable ───────────────────────────────────────────────────────

type UpsertTemplateVariableRequest struct {
	Name string `json:"name"`
}

type TemplateVariableResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	// Value is intentionally omitted — it is write-only server-side.
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ─── Error responses ─────────────────────────────────────────────────────────

// APIError represents a structured error returned by the API.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
	// Details carries per-field validation errors when present.
	Details map[string]string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}
