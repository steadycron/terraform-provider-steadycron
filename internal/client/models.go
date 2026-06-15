package client

// ─── Job ─────────────────────────────────────────────────────────────────────

// UpsertJobRequest is used for both POST /api/jobs and PATCH /api/jobs/{id}.
type UpsertJobRequest struct {
	Kind        string `json:"kind"` // "http" | "heartbeat"
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// Schedule — exactly one must be set.
	ScheduleKind    string `json:"schedule_kind"`        // "cron" | "interval"
	CronExpression  string `json:"cron_expression,omitempty"`
	IntervalSeconds *int64 `json:"interval_seconds,omitempty"`
	Timezone        string `json:"timezone,omitempty"`

	// Heartbeat-specific
	GraceSeconds          *int64 `json:"grace_seconds,omitempty"`
	StuckRunDetection     *bool  `json:"stuck_run_detection,omitempty"`
	MaxRunDurationSeconds *int64 `json:"max_run_duration_seconds,omitempty"`

	// HTTP-specific
	Method              string            `json:"http_method,omitempty"`
	URL                 string            `json:"http_url,omitempty"`
	Headers             map[string]string `json:"http_headers,omitempty"`
	Body                string            `json:"http_body,omitempty"`
	TimeoutSeconds      *int64            `json:"timeout_seconds,omitempty"`
	MaxRetries          *int64            `json:"max_retries,omitempty"`
	RetryBackoffSeconds *int64            `json:"retry_backoff_seconds,omitempty"`
	SkipIfRunning       *bool             `json:"skip_if_running,omitempty"`

	// Tags — list of tag UUIDs.
	Tags []string `json:"tags,omitempty"`

	// ManifestKey is the stable human-authored key used by code-monitoring SDKs.
	// Optional: when omitted, the server auto-generates a slug from the job name.
	ManifestKey *string `json:"manifest_key,omitempty"`
}

// PingUrls holds the three heartbeat ping endpoints.
type PingUrls struct {
	Success string `json:"success"`
	Start   string `json:"start"`
	Fail    string `json:"fail"`
}

// JobTagInfo is a tag as returned inside a job response.
type JobTagInfo struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Color string `json:"color"`
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
	GraceSeconds          int64     `json:"grace_seconds"`
	StuckRunDetection     bool      `json:"stuck_run_detection"`
	MaxRunDurationSeconds int64     `json:"max_run_duration_seconds"`
	PingUrls              *PingUrls `json:"ping_urls"`

	// HTTP-specific
	Method              *string           `json:"http_method"`
	URL                 *string           `json:"http_url"`
	Headers             map[string]string `json:"http_headers"`
	Body                *string           `json:"http_body"`
	TimeoutSeconds      *int64            `json:"timeout_seconds"`
	MaxRetries          *int64            `json:"max_retries"`
	RetryBackoffSeconds *int64            `json:"retry_backoff_seconds"`
	SkipIfRunning       bool              `json:"skip_if_running"`

	// Derived status (read-only)
	Status     *string      `json:"status"`
	NextFireAt *string      `json:"next_fire_at"`
	LastFireAt *string      `json:"last_fire_at"`
	BadgeURL   string       `json:"badge_url"`
	Tags       []JobTagInfo `json:"tags"`
	CreatedAt  string       `json:"created_at"`
	UpdatedAt  string       `json:"updated_at"`

	// ManifestKey is the stable human-authored key used by code-monitoring SDKs.
	// Null when not yet set (e.g. legacy jobs created before SPEC-18).
	ManifestKey *string `json:"manifest_key"`
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
	Name   string            `json:"name"`
	Kind   string            `json:"kind"` // email|slack|discord|webhook|telegram
	Config map[string]string `json:"config"`
}

type AlertChannelResponse struct {
	ID        string            `json:"id"`
	AccountID string            `json:"account_id"`
	Name      string            `json:"name"`
	Kind      string            `json:"kind"`
	Config    map[string]string `json:"config"`
	CreatedAt string            `json:"created_at"`
}

// ─── Alert Rule ──────────────────────────────────────────────────────────────

type UpsertAlertRuleRequest struct {
	ChannelID          string  `json:"channel_id"`
	Trigger            string  `json:"trigger"`
	Severity           string  `json:"severity,omitempty"`
	DedupWindowSeconds *int64  `json:"dedup_window_seconds,omitempty"`
	// Threshold is required for on_n_consecutive; sent as a top-level field (not in params).
	Threshold *int64  `json:"threshold,omitempty"`
	Params    *Params `json:"params,omitempty"`
}

type Params struct {
	Factor             *float64 `json:"factor,omitempty"`
	MinBaselineSamples *int64   `json:"min_baseline_samples,omitempty"`
}

type AlertRuleResponse struct {
	ID                 string  `json:"id"`
	JobID              string  `json:"job_id"`
	ChannelID          string  `json:"channel_id"`
	Trigger            string  `json:"trigger"`
	Severity           string  `json:"severity"`
	DedupWindowSeconds int64   `json:"dedup_window_seconds"`
	// Threshold is the consecutive-failure count for on_n_consecutive rules.
	Threshold *int64  `json:"threshold"`
	Params    *Params `json:"params"`
	CreatedAt string  `json:"created_at"`
}

// ─── Template Variable ───────────────────────────────────────────────────────

type UpsertTemplateVariableRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TemplateVariableResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// jobListResponse is the paginated envelope returned by GET /api/jobs.
type jobListResponse struct {
	Items      []JobResponse `json:"items"`
	TotalCount int           `json:"total_count"`
}

// ─── Error responses ─────────────────────────────────────────────────────────

// APIError represents a structured error returned by the API.
type APIError struct {
	StatusCode int
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}
