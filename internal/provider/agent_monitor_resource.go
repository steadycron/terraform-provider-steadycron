package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ resource.Resource = &AgentMonitorResource{}
var _ resource.ResourceWithImportState = &AgentMonitorResource{}

func NewAgentMonitorResource() resource.Resource {
	return &AgentMonitorResource{}
}

type AgentMonitorResource struct {
	client *client.Client
}

// agentMonitorModel is the Terraform state model for steadycron_agent_monitor.
//
// stuck_run_detection is deliberately absent: the API forces it true on every agent monitor,
// because enforcing the /start ping is what the kind is for. Exposing an attribute the server
// overrides would produce permanent plan drift.
type agentMonitorModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	RunbookNotes          types.String `tfsdk:"runbook_notes"`
	RunbookUrl            types.String `tfsdk:"runbook_url"`
	CronExpression        types.String `tfsdk:"cron_expression"`
	IntervalSeconds       types.Int64  `tfsdk:"interval_seconds"`
	Timezone              types.String `tfsdk:"timezone"`
	GraceSeconds          types.Int64  `tfsdk:"grace_seconds"`
	MaxRunDurationSeconds types.Int64  `tfsdk:"max_run_duration_seconds"`
	MisfirePolicy         types.String `tfsdk:"misfire_policy"`
	Tags                  types.Set    `tfsdk:"tags"`
	Key                   types.String `tfsdk:"key"`

	// Outcome rules
	ReportRequired          types.Bool    `tfsdk:"report_required"`
	ItemsLabel              types.String  `tfsdk:"items_label"`
	RuleEmptyResultEnabled  types.Bool    `tfsdk:"rule_empty_result_enabled"`
	RuleMaxCostUsdPerRun    types.Float64 `tfsdk:"rule_max_cost_usd_per_run"`
	RuleMaxCostUsdPerPeriod types.Float64 `tfsdk:"rule_max_cost_usd_per_period"`
	RuleCostPeriod          types.String  `tfsdk:"rule_cost_period"`
	RuleMaxSteps            types.Int64   `tfsdk:"rule_max_steps"`
	RuleMaxToolCalls        types.Int64   `tfsdk:"rule_max_tool_calls"`
	RuleMaxDurationMs       types.Int64   `tfsdk:"rule_max_duration_ms"`

	// Computed
	PingURL      types.String `tfsdk:"ping_url"`
	StartPingURL types.String `tfsdk:"start_ping_url"`
	FailPingURL  types.String `tfsdk:"fail_ping_url"`
	Token        types.String `tfsdk:"token"`
	Status       types.String `tfsdk:"status"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *AgentMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_monitor"
}

func (r *AgentMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SteadyCron AI agent monitor — like a heartbeat, but each run reports a JSON payload " +
			"that SteadyCron evaluates against the outcome rules below, so a bare exit-0 is no longer taken as proof of work.\n\n" +
			"**A run is an ordered pair of calls.** Ping `start_ping_url` when the run begins, then POST the run report to " +
			"`ping_url` when it ends (or to `fail_ping_url` on failure). `/start` is mandatory — a completion with no run " +
			"open is rejected with `422`.\n\n" +
			"```jsonc\n" +
			"// POST {ping_url}   Content-Type: application/json\n" +
			"{\n" +
			"  \"itemsProduced\": 412,   // what the empty-result rule reads — the field that matters\n" +
			"  \"steps\": 12,\n" +
			"  \"toolCalls\": 31,\n" +
			"  \"model\": \"claude-opus-5\",\n" +
			"  \"tokensIn\": 41200,\n" +
			"  \"tokensOut\": 3100,\n" +
			"  \"costUsd\": 0.84,       // wins over anything derived from tokens\n" +
			"  \"traceUrl\": \"https://cloud.langfuse.com/…\",\n" +
			"  \"summary\": \"42 tickets triaged, 3 escalated\"\n" +
			"}\n" +
			"```\n\n" +
			"Every report field is optional. Exactly one of `cron_expression` or `interval_seconds` must be set.\n\n" +
			"**Two clocks, not one:** `grace_seconds` bounds when the run must *start*; `max_run_duration_seconds` bounds how " +
			"long it may then take. Sizing one grace period to cover both is what made agent runs look missed every night.\n\n" +
			"**Token stability:** renaming a monitor is an in-place update; the ping URLs and token are preserved.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned UUID for this monitor.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name. Renaming is an in-place update — the ping URLs and token are unchanged.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Optional free-text description.",
			},
			"runbook_notes": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Markdown remediation notes shown inline in failure alert notifications (Slack, Telegram, Email). Max 4000 characters.",
			},
			"runbook_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional link to an external runbook (e.g. Notion, Confluence). Max 2048 characters.",
			},
			"cron_expression": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Expected schedule as a cron expression. Mutually exclusive with `interval_seconds`.",
			},
			"interval_seconds": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "Expected run interval in seconds. Mutually exclusive with `cron_expression`. " +
					"Interval agents anchor the next window on the previous `/start`, not the previous completion, so " +
					"\"start every hour\" does not drift by the run duration each cycle.",
			},
			"timezone": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
				MarkdownDescription: "IANA timezone name. Also the timezone the per-period spend bucket is evaluated in, so \"this month\" means what you think it means. Defaults to `UTC`.",
			},
			"grace_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
				MarkdownDescription: "How long after the scheduled time the run may still `/start` before the schedule window counts as missed. Defaults to `60`.",
			},
			"max_run_duration_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(120),
				MarkdownDescription: "How long a started run may take before it is considered abandoned. Also the upper bound backing the slow-run rule — a run that finished beyond it was slow by your own definition. Defaults to `120`.",
			},
			"misfire_policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("do_nothing"),
				MarkdownDescription: "What to do when a scheduled window is missed (e.g. the scheduler was down). `do_nothing` skips it; `fire_once_now` fires once immediately. Defaults to `do_nothing`.",
				Validators: []validator.String{
					stringvalidator.OneOf("do_nothing", "fire_once_now"),
				},
			},
			"tags": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				MarkdownDescription: "Set of tag IDs to attach. Use `steadycron_tag` resources and reference their `id`.",
			},
			"key": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Stable job key referenced by SDKs and the MCP server (`list_agent_monitors` returns it).\n\n" +
					"When set, this exact string is used as the job key. When omitted, the server auto-generates a slug " +
					"from the monitor name (visible after apply). Changing `key` is an in-place update. Must be unique within the account.",
			},

			// ── Outcome rules ──────────────────────────────────────────────────
			"report_required": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				MarkdownDescription: "When `true` (the default), a success ping carrying no parseable run report is recorded as " +
					"**unverified** rather than success, and can fire `on_unverified_run`. This is the point of the kind: a bare " +
					"exit-0 is not proof of work. Set `false` only for an agent whose runtime cannot POST a body.\n\n" +
					"**Forces replacement.** The API accepts this on create only — it is deliberately absent from the update surface.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"items_label": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "What this agent produces, in your own words — `\"tickets\"`, `\"rows\"`, `\"documents\"`. " +
					"Labels `itemsProduced` throughout the dashboard and in alert text. Max 40 characters.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(40),
				},
			},
			"rule_empty_result_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				MarkdownDescription: "When `true` (the default), a run reporting zero items produced is a **failure**. This is the " +
					"headline rule — the failure mode a plain cron monitor cannot catch. It does not pile onto an already-failed " +
					"run, and it does not fire when neither `itemsProduced` nor `outputChars` was reported.\n\n" +
					"**Forces replacement.** The API accepts this on create only.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"rule_max_cost_usd_per_run": schema.Float64Attribute{
				Optional: true,
				MarkdownDescription: "Alert (`on_cost_exceeded`) when a single run costs more than this, in **USD**. Overspending " +
					"alerts but does not fail the run. An unpriced run never trips a cost rule — a model with no price on file " +
					"must not read as free. Omit for no ceiling.",
			},
			"rule_max_cost_usd_per_period": schema.Float64Attribute{
				Optional:            true,
				MarkdownDescription: "Alert when this monitor's summed spend across `rule_cost_period` exceeds this, in **USD**. Omit for no ceiling.",
			},
			"rule_cost_period": schema.StringAttribute{
				Optional: true,
				// Computed because the server fills this in: setting a per-period ceiling without
				// a period stores `month`. Leaving it Optional-only made that a "Provider produced
				// inconsistent result after apply" — config said null, the response said "month".
				Computed:            true,
				MarkdownDescription: "The window `rule_max_cost_usd_per_period` sums over: `day` or `month`. Buckets are evaluated in the monitor's own `timezone`. Defaults to `month` when a per-period ceiling is set, and is cleared with it.",
				Validators: []validator.String{
					stringvalidator.OneOf("day", "month"),
				},
			},
			"rule_max_steps": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Alert (`on_no_progress`) when a run's reported step count exceeds this — loop detection. Omit for no ceiling.",
			},
			"rule_max_tool_calls": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Alert (`on_no_progress`) when a run's reported tool-call count exceeds this. Omit for no ceiling.",
			},
			"rule_max_duration_ms": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "Alert (`on_slow_run`) when a run's **measured** duration — the gap between its `/start` and its " +
					"completion, not the figure the agent reports — exceeds this, in milliseconds. An optional lower warning " +
					"threshold beneath `max_run_duration_seconds`. Omit for no ceiling.",
			},

			// ── Computed / sensitive ───────────────────────────────────────────
			"ping_url": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The completion URL to POST the run report to (`https://ping.steadycron.com/<token>/success`). **Sensitive** — treat like a secret.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"start_ping_url": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The URL to ping when a run begins. Required — a completion with no run open is rejected. **Sensitive** — treat like a secret.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"fail_ping_url": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The URL to POST a failure report to. Unlike a completion, this is never gated on an open `/start` — refusing a failure report would turn a known failure into silence. **Sensitive** — treat like a secret.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The raw ping token. **Sensitive** — treat like a secret. Unchanged by rename operations.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Derived current status. `unverified` means a run arrived on time but carried no required report — the schedule was met, the outcome is unproven.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "RFC3339 creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 last-updated timestamp.",
			},
		},
	}
}

func (r *AgentMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *AgentMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan agentMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateSchedule(plan.CronExpression, plan.IntervalSeconds); err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}
	if err := validateAgentCostRules(plan); err != nil {
		resp.Diagnostics.AddError("Invalid cost rules", err.Error())
		return
	}

	apiReq, diags := agentModelToRequest(ctx, plan, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.CreateJob(ctx, apiReq)
	if err != nil {
		appendAPIError(&resp.Diagnostics, "creating agent monitor", err)
		return
	}

	// The create body does not wire tags; use the dedicated endpoint.
	if err := r.client.SetJobTags(ctx, job.ID, apiReq.Tags); err != nil {
		appendAPIError(&resp.Diagnostics, "setting tags on agent monitor", err)
		return
	}

	// The Create response may omit ping_urls; fetch the full resource via GET.
	if job.PingUrls == nil || job.PingUrls.Success == "" {
		job, err = r.client.GetJob(ctx, job.ID)
		if err != nil {
			appendAPIError(&resp.Diagnostics, "reading agent monitor after create", err)
			return
		}
	}

	// Save planned tags before agentResponseToModel overwrites them with the create response
	// (which has no tags, since tags are set via a separate endpoint).
	plannedTags := plan.Tags

	resp.Diagnostics.Append(agentResponseToModel(ctx, job, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Tags = plannedTags
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state agentMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.GetJob(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		appendAPIError(&resp.Diagnostics, "reading agent monitor", err)
		return
	}

	// Preserve sensitive fields from state — the API redacts them on GET.
	savedToken := state.Token
	savedPingURL := state.PingURL
	savedStartURL := state.StartPingURL
	savedFailURL := state.FailPingURL

	resp.Diagnostics.Append(agentResponseToModel(ctx, job, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Token.IsNull() || state.Token.ValueString() == "" {
		state.Token = savedToken
	}
	if state.PingURL.IsNull() || state.PingURL.ValueString() == "" {
		state.PingURL = savedPingURL
		state.StartPingURL = savedStartURL
		state.FailPingURL = savedFailURL
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AgentMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan agentMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state agentMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateSchedule(plan.CronExpression, plan.IntervalSeconds); err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}
	if err := validateAgentCostRules(plan); err != nil {
		resp.Diagnostics.AddError("Invalid cost rules", err.Error())
		return
	}

	apiReq, diags := agentModelToRequest(ctx, plan, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.UpdateJob(ctx, state.ID.ValueString(), apiReq)
	if err != nil {
		appendAPIError(&resp.Diagnostics, "updating agent monitor", err)
		return
	}

	// Sync tags via the dedicated endpoint — the update body does not wire tags.
	if err := r.client.SetJobTags(ctx, state.ID.ValueString(), apiReq.Tags); err != nil {
		appendAPIError(&resp.Diagnostics, "setting tags on agent monitor", err)
		return
	}

	// Preserve sensitive fields from state — the token is stable across renames.
	savedToken := state.Token
	savedPingURL := state.PingURL
	savedStartURL := state.StartPingURL
	savedFailURL := state.FailPingURL
	plannedTags := plan.Tags

	resp.Diagnostics.Append(agentResponseToModel(ctx, job, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Token.IsNull() || plan.Token.ValueString() == "" {
		plan.Token = savedToken
	}
	if plan.PingURL.IsNull() || plan.PingURL.ValueString() == "" {
		plan.PingURL = savedPingURL
		plan.StartPingURL = savedStartURL
		plan.FailPingURL = savedFailURL
	}
	plan.Tags = plannedTags

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state agentMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteJob(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			appendAPIError(&resp.Diagnostics, "deleting agent monitor", err)
		}
	}
}

func (r *AgentMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	job, err := r.client.GetJob(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Monitor not found", fmt.Sprintf("No agent monitor with id %q was found.", req.ID))
			return
		}
		appendAPIError(&resp.Diagnostics, "importing agent monitor", err)
		return
	}
	if job.Kind != "agent" {
		resp.Diagnostics.AddError("Wrong resource type",
			fmt.Sprintf("Job %q has kind %q; import it with steadycron_%s instead.", req.ID, job.Kind, resourceSuffixForKind(job.Kind)))
		return
	}

	var state agentMonitorModel
	resp.Diagnostics.Append(agentResponseToModel(ctx, job, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// validateAgentCostRules rejects the one combination the server would otherwise reject for us,
// so the practitioner sees it at plan time with the attribute named.
func validateAgentCostRules(m agentMonitorModel) error {
	periodSet := !m.RuleCostPeriod.IsNull() && !m.RuleCostPeriod.IsUnknown()
	ceilingSet := !m.RuleMaxCostUsdPerPeriod.IsNull() && !m.RuleMaxCostUsdPerPeriod.IsUnknown() &&
		m.RuleMaxCostUsdPerPeriod.ValueFloat64() > 0

	if periodSet && !ceilingSet {
		return fmt.Errorf("rule_cost_period requires rule_max_cost_usd_per_period to be set")
	}
	return nil
}

// agentModelToRequest builds the create or update body.
//
// forUpdate matters because the two surfaces read an absent ceiling differently. On create,
// omitting a field leaves it null. On update, omitting means "unchanged" — so a practitioner
// deleting `rule_max_steps = 40` from their config would otherwise silently keep the ceiling.
// The API's documented sentinel for "clear this" is 0, so that is what an unset attribute sends.
func agentModelToRequest(ctx context.Context, m agentMonitorModel, forUpdate bool) (client.UpsertJobRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := client.UpsertJobRequest{
		Kind:                  "agent",
		Name:                  m.Name.ValueString(),
		Description:           m.Description.ValueString(),
		Timezone:              m.Timezone.ValueString(),
		GraceSeconds:          int64Ptr(m.GraceSeconds.ValueInt64()),
		MaxRunDurationSeconds: int64Ptr(m.MaxRunDurationSeconds.ValueInt64()),
	}

	if !m.RunbookNotes.IsNull() && !m.RunbookNotes.IsUnknown() {
		v := m.RunbookNotes.ValueString()
		req.RunbookNotes = &v
	}
	if !m.RunbookUrl.IsNull() && !m.RunbookUrl.IsUnknown() {
		v := m.RunbookUrl.ValueString()
		req.RunbookUrl = &v
	}
	if !m.CronExpression.IsNull() && !m.CronExpression.IsUnknown() {
		req.ScheduleKind = "cron"
		req.CronExpression = m.CronExpression.ValueString()
	} else {
		req.ScheduleKind = "interval"
		v := m.IntervalSeconds.ValueInt64()
		req.IntervalSeconds = &v
	}

	var tags []string
	diags.Append(m.Tags.ElementsAs(ctx, &tags, false)...)
	req.Tags = tags

	if !m.Key.IsNull() && !m.Key.IsUnknown() {
		v := m.Key.ValueString()
		req.JobKey = &v
	}
	if !m.MisfirePolicy.IsNull() && !m.MisfirePolicy.IsUnknown() {
		v := m.MisfirePolicy.ValueString()
		req.MisfirePolicy = &v
	}

	// items_label: "" clears it server-side, which is what an unset attribute means here.
	itemsLabel := ""
	if !m.ItemsLabel.IsNull() && !m.ItemsLabel.IsUnknown() {
		itemsLabel = m.ItemsLabel.ValueString()
	}
	if itemsLabel != "" || forUpdate {
		req.ItemsLabel = &itemsLabel
	}

	// Create-only flags. Both are RequiresReplace, so an update never needs to carry them —
	// and the API would ignore them if it did.
	if !forUpdate {
		if !m.ReportRequired.IsNull() && !m.ReportRequired.IsUnknown() {
			v := m.ReportRequired.ValueBool()
			req.ReportRequired = &v
		}
		if !m.RuleEmptyResultEnabled.IsNull() && !m.RuleEmptyResultEnabled.IsUnknown() {
			v := m.RuleEmptyResultEnabled.ValueBool()
			req.RuleEmptyResultEnabled = &v
		}
	}

	req.RuleMaxCostUsdPerRun = float64Ceiling(m.RuleMaxCostUsdPerRun, forUpdate)
	req.RuleMaxCostUsdPerPeriod = float64Ceiling(m.RuleMaxCostUsdPerPeriod, forUpdate)
	req.RuleMaxSteps = int64Ceiling(m.RuleMaxSteps, forUpdate)
	req.RuleMaxToolCalls = int64Ceiling(m.RuleMaxToolCalls, forUpdate)
	req.RuleMaxDurationMs = int64Ceiling(m.RuleMaxDurationMs, forUpdate)

	// The period travels only alongside a ceiling that is staying set; clearing the ceiling
	// clears the period server-side, so sending one here would be contradictory.
	if !m.RuleCostPeriod.IsNull() && !m.RuleCostPeriod.IsUnknown() && req.RuleMaxCostUsdPerPeriod != nil &&
		*req.RuleMaxCostUsdPerPeriod > 0 {
		v := m.RuleCostPeriod.ValueString()
		req.RuleCostPeriod = &v
	}

	return req, diags
}

// float64Ceiling maps an optional ceiling attribute onto the wire. Unset sends nothing on create
// (the server leaves it null) and an explicit 0 on update (the server's "clear this" sentinel).
func float64Ceiling(v types.Float64, forUpdate bool) *float64 {
	if !v.IsNull() && !v.IsUnknown() {
		f := v.ValueFloat64()
		return &f
	}
	if forUpdate {
		zero := 0.0
		return &zero
	}
	return nil
}

func int64Ceiling(v types.Int64, forUpdate bool) *int64 {
	if !v.IsNull() && !v.IsUnknown() {
		n := v.ValueInt64()
		return &n
	}
	if forUpdate {
		var zero int64
		return &zero
	}
	return nil
}

func agentResponseToModel(ctx context.Context, job *client.JobResponse, m *agentMonitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(job.ID)
	m.Name = types.StringValue(job.Name)
	m.Description = types.StringValue(stringPtrOrEmpty(job.Description))
	m.RunbookNotes = types.StringPointerValue(job.RunbookNotes)
	m.RunbookUrl = types.StringPointerValue(job.RunbookUrl)
	m.Timezone = types.StringValue(job.Timezone)
	m.GraceSeconds = types.Int64Value(job.GraceSeconds)
	m.MaxRunDurationSeconds = types.Int64Value(job.MaxRunDurationSeconds)

	if job.CronExpression != nil {
		m.CronExpression = types.StringValue(*job.CronExpression)
		m.IntervalSeconds = types.Int64Null()
	} else if job.IntervalSeconds != nil {
		m.IntervalSeconds = types.Int64Value(*job.IntervalSeconds)
		m.CronExpression = types.StringNull()
	}

	// Read/Update callers preserve prior state values when the API redacts them on GET.
	if job.PingUrls != nil && job.PingUrls.Success != "" {
		m.PingURL = types.StringValue(job.PingUrls.Success)
		m.StartPingURL = types.StringValue(job.PingUrls.Start)
		m.FailPingURL = types.StringValue(job.PingUrls.Fail)
		// Derive the raw token from the last path segment of the start URL, which is the one
		// suffix that is always present on an agent monitor.
		m.Token = types.StringValue(tokenFromPingURL(job.PingUrls.Success))
	} else {
		m.PingURL = types.StringValue("")
		m.StartPingURL = types.StringValue("")
		m.FailPingURL = types.StringValue("")
		m.Token = types.StringValue("")
	}

	// Agent settings come back null on any other kind; the import guard already refuses those,
	// so a null here means the server has no value rather than a wrong-kind read.
	m.ReportRequired = types.BoolValue(boolPtrOrDefault(job.ReportRequired, true))
	m.RuleEmptyResultEnabled = types.BoolValue(boolPtrOrDefault(job.RuleEmptyResultEnabled, true))
	m.ItemsLabel = types.StringPointerValue(job.ItemsLabel)
	m.RuleMaxCostUsdPerRun = types.Float64PointerValue(job.RuleMaxCostUsdPerRun)
	m.RuleMaxCostUsdPerPeriod = types.Float64PointerValue(job.RuleMaxCostUsdPerPeriod)
	m.RuleCostPeriod = types.StringPointerValue(job.RuleCostPeriod)
	m.RuleMaxSteps = types.Int64PointerValue(job.RuleMaxSteps)
	m.RuleMaxToolCalls = types.Int64PointerValue(job.RuleMaxToolCalls)
	m.RuleMaxDurationMs = types.Int64PointerValue(job.RuleMaxDurationMs)

	m.Status = types.StringPointerValue(job.Status)
	m.CreatedAt = types.StringValue(normalizeTimestamp(job.CreatedAt))
	m.UpdatedAt = types.StringValue(normalizeTimestamp(job.UpdatedAt))

	if job.JobKey != nil {
		m.Key = types.StringValue(*job.JobKey)
	} else {
		m.Key = types.StringNull()
	}

	if job.MisfirePolicy != "" {
		m.MisfirePolicy = types.StringValue(job.MisfirePolicy)
	} else {
		m.MisfirePolicy = types.StringValue("do_nothing")
	}

	tagElems := make([]attr.Value, len(job.Tags))
	for i, t := range job.Tags {
		tagElems[i] = types.StringValue(t.ID)
	}
	tagsVal, d := types.SetValue(types.StringType, tagElems)
	diags.Append(d...)
	m.Tags = tagsVal

	return diags
}

// tokenFromPingURL strips the trailing verb segment (/success, /start, /fail) when present and
// returns the raw token.
func tokenFromPingURL(pingURL string) string {
	u := strings.TrimRight(pingURL, "/")
	for _, suffix := range []string{"/success", "/start", "/fail"} {
		if strings.HasSuffix(u, suffix) {
			u = strings.TrimSuffix(u, suffix)
			break
		}
	}
	if i := strings.LastIndex(u, "/"); i >= 0 {
		return u[i+1:]
	}
	return u
}

func boolPtrOrDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

// resourceSuffixForKind names the resource that does own a job of the given kind, so a
// wrong-kind import error points at the right one instead of guessing.
func resourceSuffixForKind(kind string) string {
	switch kind {
	case "agent":
		return "agent_monitor"
	case "heartbeat":
		return "heartbeat_monitor"
	default:
		return "http_job"
	}
}
