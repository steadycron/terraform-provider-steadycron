package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ resource.Resource = &HeartbeatMonitorResource{}
var _ resource.ResourceWithImportState = &HeartbeatMonitorResource{}

func NewHeartbeatMonitorResource() resource.Resource {
	return &HeartbeatMonitorResource{}
}

type HeartbeatMonitorResource struct {
	client *client.Client
}

// heartbeatMonitorModel is the Terraform state model for steadycron_heartbeat_monitor.
type heartbeatMonitorModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	CronExpression        types.String `tfsdk:"cron_expression"`
	IntervalSeconds       types.Int64  `tfsdk:"interval_seconds"`
	Timezone              types.String `tfsdk:"timezone"`
	GraceSeconds          types.Int64  `tfsdk:"grace_seconds"`
	StuckRunDetection     types.Bool   `tfsdk:"stuck_run_detection"`
	MaxRunDurationSeconds types.Int64  `tfsdk:"max_run_duration_seconds"`
	Tags                  types.Set    `tfsdk:"tags"`
	Key                   types.String `tfsdk:"key"`
	// Computed
	PingURL   types.String `tfsdk:"ping_url"`
	Token     types.String `tfsdk:"token"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *HeartbeatMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat_monitor"
}

func (r *HeartbeatMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SteadyCron heartbeat monitor — SteadyCron watches for expected pings and alerts when they stop.\n\n" +
			"After creation, send periodic pings to `ping_url` (e.g. `curl {{ping_url}}`) from your cron job.\n\n" +
			"Exactly one of `cron_expression` or `interval_seconds` must be set.\n\n" +
			"**Token stability:** renaming a monitor is an in-place update; `ping_url` and `token` are preserved.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned UUID for this monitor.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name. Renaming is an in-place update — the ping URL and token are unchanged.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Optional free-text description.",
			},
			"cron_expression": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Expected schedule as a cron expression. Mutually exclusive with `interval_seconds`.",
			},
			"interval_seconds": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Expected ping interval in seconds. Mutually exclusive with `cron_expression`.",
			},
			"timezone": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
				MarkdownDescription: "IANA timezone name for cron scheduling. Defaults to `UTC`.",
			},
			"grace_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
				MarkdownDescription: "Grace period in seconds after the expected ping time before the monitor is considered missed. Defaults to `60`.",
			},
			"stuck_run_detection": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Alert if a `/start` ping is not followed by `/success` or `/fail` within `max_run_duration_seconds`. Defaults to `true`.",
			},
			"max_run_duration_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(120),
				MarkdownDescription: "Maximum expected run duration in seconds (used by stuck-run detection). Defaults to `120`.",
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
				MarkdownDescription: "Stable job key referenced by code-monitoring SDKs (e.g. `@steadycron.job(\"my-key\")`).\n\n" +
					"When set, this exact string is used as the job key. When omitted, the server " +
					"auto-generates a slug from the job name (visible after apply).\n\n" +
					"Changing `key` is an in-place update — no replacement occurs, but any in-code references using " +
					"the old key must be updated. Must be unique within the account.",
			},
			// Computed / sensitive
			"ping_url": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The URL to ping (e.g. `https://ping.steadycron.com/<token>`). **Sensitive** — treat like a secret.",
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
				MarkdownDescription: "Derived current status.",
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

func (r *HeartbeatMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HeartbeatMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan heartbeatMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateSchedule(plan.CronExpression, plan.IntervalSeconds); err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}

	apiReq, diags := heartbeatModelToRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.CreateJob(ctx, apiReq)
	if err != nil {
		appendAPIError(&resp.Diagnostics, "creating heartbeat monitor", err)
		return
	}

	// The create body does not wire tags; use the dedicated endpoint.
	if err := r.client.SetJobTags(ctx, job.ID, apiReq.Tags); err != nil {
		appendAPIError(&resp.Diagnostics, "setting tags on heartbeat monitor", err)
		return
	}

	// The Create response may omit ping_url; fetch the full resource via GET.
	if job.PingUrls == nil || job.PingUrls.Success == "" {
		job, err = r.client.GetJob(ctx, job.ID)
		if err != nil {
			appendAPIError(&resp.Diagnostics, "reading heartbeat monitor after create", err)
			return
		}
	}

	// Save planned tags before heartbeatResponseToModel overwrites them with the
	// create response (which has no tags since tags are set via a separate endpoint).
	plannedTags := plan.Tags

	resp.Diagnostics.Append(heartbeatResponseToModel(ctx, job, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Tags = plannedTags
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HeartbeatMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state heartbeatMonitorModel
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
		appendAPIError(&resp.Diagnostics, "reading heartbeat monitor", err)
		return
	}

	// Preserve sensitive fields from state — the API redacts them on GET.
	savedToken := state.Token
	savedPingURL := state.PingURL

	resp.Diagnostics.Append(heartbeatResponseToModel(ctx, job, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Token.IsNull() || state.Token.ValueString() == "" {
		state.Token = savedToken
	}
	if state.PingURL.IsNull() || state.PingURL.ValueString() == "" {
		state.PingURL = savedPingURL
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HeartbeatMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan heartbeatMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state heartbeatMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateSchedule(plan.CronExpression, plan.IntervalSeconds); err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}

	apiReq, diags := heartbeatModelToRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.UpdateJob(ctx, state.ID.ValueString(), apiReq)
	if err != nil {
		appendAPIError(&resp.Diagnostics, "updating heartbeat monitor", err)
		return
	}

	// Sync tags via the dedicated endpoint — update body does not wire tags.
	if err := r.client.SetJobTags(ctx, state.ID.ValueString(), apiReq.Tags); err != nil {
		appendAPIError(&resp.Diagnostics, "setting tags on heartbeat monitor", err)
		return
	}

	// Preserve sensitive fields from state — token is stable across renames.
	savedToken := state.Token
	savedPingURL := state.PingURL

	// Save planned tags before heartbeatResponseToModel overwrites them with the
	// update response (which has no tags).
	plannedTags := plan.Tags

	resp.Diagnostics.Append(heartbeatResponseToModel(ctx, job, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Token.IsNull() || plan.Token.ValueString() == "" {
		plan.Token = savedToken
	}
	if plan.PingURL.IsNull() || plan.PingURL.ValueString() == "" {
		plan.PingURL = savedPingURL
	}
	plan.Tags = plannedTags

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HeartbeatMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state heartbeatMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteJob(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			appendAPIError(&resp.Diagnostics, "deleting heartbeat monitor", err)
		}
	}
}

func (r *HeartbeatMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	job, err := r.client.GetJob(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Monitor not found", fmt.Sprintf("No heartbeat monitor with id %q was found.", req.ID))
			return
		}
		appendAPIError(&resp.Diagnostics, "importing heartbeat monitor", err)
		return
	}
	if job.Kind != "heartbeat" {
		resp.Diagnostics.AddError("Wrong resource type",
			fmt.Sprintf("Job %q has kind %q; import it with steadycron_http_job instead.", req.ID, job.Kind))
		return
	}

	var state heartbeatMonitorModel
	resp.Diagnostics.Append(heartbeatResponseToModel(ctx, job, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func heartbeatModelToRequest(ctx context.Context, m heartbeatMonitorModel) (client.UpsertJobRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := client.UpsertJobRequest{
		Kind:                  "heartbeat",
		Name:                  m.Name.ValueString(),
		Description:           m.Description.ValueString(),
		Timezone:              m.Timezone.ValueString(),
		GraceSeconds:          int64Ptr(m.GraceSeconds.ValueInt64()),
		StuckRunDetection:     boolPtr(m.StuckRunDetection.ValueBool()),
		MaxRunDurationSeconds: int64Ptr(m.MaxRunDurationSeconds.ValueInt64()),
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

	return req, diags
}

func heartbeatResponseToModel(ctx context.Context, job *client.JobResponse, m *heartbeatMonitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(job.ID)
	m.Name = types.StringValue(job.Name)
	m.Description = types.StringValue(stringPtrOrEmpty(job.Description))
	m.Timezone = types.StringValue(job.Timezone)
	m.GraceSeconds = types.Int64Value(job.GraceSeconds)
	m.StuckRunDetection = types.BoolValue(job.StuckRunDetection)
	m.MaxRunDurationSeconds = types.Int64Value(job.MaxRunDurationSeconds)

	if job.CronExpression != nil {
		m.CronExpression = types.StringValue(*job.CronExpression)
		m.IntervalSeconds = types.Int64Null()
	} else if job.IntervalSeconds != nil {
		m.IntervalSeconds = types.Int64Value(*job.IntervalSeconds)
		m.CronExpression = types.StringNull()
	}

	// Extract ping_url and token from the nested PingUrls object.
	// Read/Update callers preserve prior state values when the API redacts them on GET.
	if job.PingUrls != nil && job.PingUrls.Success != "" {
		m.PingURL = types.StringValue(job.PingUrls.Success)
		// Derive the raw token from the last path segment of the success URL.
		u := strings.TrimRight(job.PingUrls.Success, "/")
		if i := strings.LastIndex(u, "/"); i >= 0 {
			m.Token = types.StringValue(u[i+1:])
		} else {
			m.Token = types.StringValue(u)
		}
	} else {
		m.PingURL = types.StringValue("")
		m.Token = types.StringValue("")
	}

	m.Status = types.StringPointerValue(job.Status)
	m.CreatedAt = types.StringValue(normalizeTimestamp(job.CreatedAt))
	m.UpdatedAt = types.StringValue(normalizeTimestamp(job.UpdatedAt))

	if job.JobKey != nil {
		m.Key = types.StringValue(*job.JobKey)
	} else {
		m.Key = types.StringNull()
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
