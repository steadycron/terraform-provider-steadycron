package client

// ─── Job ─────────────────────────────────────────────────────────────────────

// UpsertJobRequest is used for both POST /api/jobs and PATCH /api/jobs/{id}.
type UpsertJobRequest struct {
	Kind        string `json:"kind"` // "http" | "heartbeat" | "agent"
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// Optional remediation runbook shown inline in failure alert notifications.
	RunbookNotes *string `json:"runbook_notes,omitempty"`
	RunbookUrl   *string `json:"runbook_url,omitempty"`

	// Schedule — exactly one must be set.
	ScheduleKind    string `json:"schedule_kind"` // "cron" | "interval"
	CronExpression  string `json:"cron_expression,omitempty"`
	IntervalSeconds *int64 `json:"interval_seconds,omitempty"`
	Timezone        string `json:"timezone,omitempty"`

	// Ping-driven kinds (heartbeat and agent monitors)
	GraceSeconds          *int64 `json:"grace_seconds,omitempty"`
	StuckRunDetection     *bool  `json:"stuck_run_detection,omitempty"`
	MaxRunDurationSeconds *int64 `json:"max_run_duration_seconds,omitempty"`

	// Agent-monitor-specific. Cost ceilings are USD on the wire (micro-USD in the database).
	//
	// On PATCH the API reads a numeric ceiling as: omitted = unchanged, 0 = clear. Terraform
	// always sends the full desired state, so the resource sends an explicit 0 for an attribute
	// the practitioner removed — otherwise deleting the line from the config would be a no-op.
	//
	// ReportRequired and RuleEmptyResultEnabled are create-only: the API deliberately leaves
	// them off UpdateJobRequest, so the resource marks both RequiresReplace.
	// Every ceiling is a pointer so that an explicit 0 still serializes — `omitempty` on a
	// pointer drops only nil, which is exactly the "leave it alone" case.
	ReportRequired          *bool    `json:"report_required,omitempty"`
	ItemsLabel              *string  `json:"items_label,omitempty"`
	RuleEmptyResultEnabled  *bool    `json:"rule_empty_result_enabled,omitempty"`
	RuleMaxCostUsdPerRun    *float64 `json:"rule_max_cost_usd_per_run,omitempty"`
	RuleMaxCostUsdPerPeriod *float64 `json:"rule_max_cost_usd_per_period,omitempty"`
	RuleCostPeriod          *string  `json:"rule_cost_period,omitempty"` // "day" | "month"
	RuleMaxSteps            *int64   `json:"rule_max_steps,omitempty"`
	RuleMaxToolCalls        *int64   `json:"rule_max_tool_calls,omitempty"`
	RuleMaxDurationMs       *int64   `json:"rule_max_duration_ms,omitempty"`

	// Schedule misfire behaviour: "do_nothing" (default) | "fire_once_now"
	MisfirePolicy *string `json:"misfire_policy,omitempty"`

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

	// JobKey is the stable human-authored key used by code-monitoring SDKs.
	// Optional: when omitted, the server auto-generates a slug from the job name.
	JobKey *string `json:"job_key,omitempty"`
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

	// Optional remediation runbook shown inline in failure alert notifications.
	RunbookNotes *string `json:"runbook_notes"`
	RunbookUrl   *string `json:"runbook_url"`

	ScheduleKind    string  `json:"schedule_kind"`
	CronExpression  *string `json:"cron_expression"`
	IntervalSeconds *int64  `json:"interval_seconds"`
	Timezone        string  `json:"timezone"`

	// Ping-driven kinds (heartbeat and agent monitors)
	GraceSeconds          int64     `json:"grace_seconds"`
	StuckRunDetection     bool      `json:"stuck_run_detection"`
	MaxRunDurationSeconds int64     `json:"max_run_duration_seconds"`
	PingUrls              *PingUrls `json:"ping_urls"`

	// Agent-monitor-specific. The API emits these only for agent monitors — they are null on
	// every other kind, so surfacing them elsewhere would suggest they do something there.
	ReportRequired          *bool    `json:"report_required"`
	ItemsLabel              *string  `json:"items_label"`
	RuleEmptyResultEnabled  *bool    `json:"rule_empty_result_enabled"`
	RuleMaxCostUsdPerRun    *float64 `json:"rule_max_cost_usd_per_run"`
	RuleMaxCostUsdPerPeriod *float64 `json:"rule_max_cost_usd_per_period"`
	RuleCostPeriod          *string  `json:"rule_cost_period"`
	RuleMaxSteps            *int64   `json:"rule_max_steps"`
	RuleMaxToolCalls        *int64   `json:"rule_max_tool_calls"`
	RuleMaxDurationMs       *int64   `json:"rule_max_duration_ms"`

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

	JobKey        *string `json:"job_key"`
	MisfirePolicy string  `json:"misfire_policy"`
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
	ChannelID          string `json:"channel_id"`
	Trigger            string `json:"trigger"`
	Severity           string `json:"severity,omitempty"`
	DedupWindowSeconds *int64 `json:"dedup_window_seconds,omitempty"`
	// Threshold is required for on_n_consecutive; sent as a top-level field (not in params).
	Threshold *int64  `json:"threshold,omitempty"`
	Params    *Params `json:"params,omitempty"`
}

type Params struct {
	Factor             *float64 `json:"factor,omitempty"`
	MinBaselineSamples *int64   `json:"min_baseline_samples,omitempty"`
}

type AlertRuleResponse struct {
	ID                 string `json:"id"`
	JobID              string `json:"job_id"`
	ChannelID          string `json:"channel_id"`
	Trigger            string `json:"trigger"`
	Severity           string `json:"severity"`
	DedupWindowSeconds int64  `json:"dedup_window_seconds"`
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
