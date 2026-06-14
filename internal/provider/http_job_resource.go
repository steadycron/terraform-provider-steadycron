package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ resource.Resource = &HTTPJobResource{}
var _ resource.ResourceWithImportState = &HTTPJobResource{}

func NewHTTPJobResource() resource.Resource {
	return &HTTPJobResource{}
}

type HTTPJobResource struct {
	client *client.Client
}

// httpJobModel is the Terraform state model for steadycron_http_job.
type httpJobModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Method              types.String `tfsdk:"method"`
	URL                 types.String `tfsdk:"url"`
	CronExpression      types.String `tfsdk:"cron_expression"`
	IntervalSeconds     types.Int64  `tfsdk:"interval_seconds"`
	Timezone            types.String `tfsdk:"timezone"`
	TimeoutSeconds      types.Int64  `tfsdk:"timeout_seconds"`
	MaxRetries          types.Int64  `tfsdk:"max_retries"`
	RetryBackoffSeconds types.Int64  `tfsdk:"retry_backoff_seconds"`
	Headers             types.Map    `tfsdk:"headers"`
	Body                types.String `tfsdk:"body"`
	SkipIfRunning       types.Bool   `tfsdk:"skip_if_running"`
	Tags                types.Set    `tfsdk:"tags"`
	// Computed
	Status     types.String `tfsdk:"status"`
	NextFireAt types.String `tfsdk:"next_fire_at"`
	LastFireAt types.String `tfsdk:"last_fire_at"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (r *HTTPJobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_job"
}

func (r *HTTPJobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SteadyCron HTTP job — a scheduled HTTPS call SteadyCron makes on your behalf.\n\n" +
			"Exactly one of `cron_expression` or `interval_seconds` must be set.\n\n" +
			"**Ownership note:** resources created via Terraform have a null `manifest_namespace`, so they are never " +
			"pruned by `steadycron sync --prune`. Manage a resource via Terraform *or* the manifest/CLI, not both.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned UUID for this job.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name for the job. In-place rename is supported without recreating the job.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Optional free-text description.",
			},
			"method": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "HTTP method. One of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`.",
				Validators: []validator.String{
					stringvalidator.OneOf("GET", "POST", "PUT", "PATCH", "DELETE"),
				},
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target URL. Must be `http://` or `https://`. Private/RFC-1918 addresses are blocked by the SSRF guard.",
			},
			"cron_expression": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Cron expression (5-field). Mutually exclusive with `interval_seconds`.",
			},
			"interval_seconds": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Run interval in seconds. Mutually exclusive with `cron_expression`.",
			},
			"timezone": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
				MarkdownDescription: "IANA timezone name for cron scheduling (e.g. `Europe/Berlin`). Defaults to `UTC`.",
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Request timeout in seconds. The server enforces a plan-specific maximum. Defaults to the account plan's default when not set.",
			},
			"max_retries": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "Number of retry attempts on failure. Defaults to `0` (no retries).",
			},
			"retry_backoff_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
				MarkdownDescription: "Seconds to wait between retries. Defaults to `30`.",
			},
			"headers": schema.MapAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				MarkdownDescription: "HTTP headers sent with each execution. Values containing `{{template_var}}` placeholders are resolved server-side at execution time.",
			},
			"body": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Request body (for POST/PUT/PATCH).",
			},
			"skip_if_running": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Skip the scheduled fire if a previous execution is still running. Defaults to `false`.",
			},
			"tags": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				MarkdownDescription: "Set of tag IDs to attach to this job. Use `steadycron_tag` resources and reference their `id`.",
			},
			// Computed-only
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Derived current status: `success`, `failure`, `running`, `paused`, `missed`, `skipped`, or `null` (new).",
			},
			"next_fire_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp of the next scheduled fire.",
			},
			"last_fire_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp of the most recent fire.",
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

func (r *HTTPJobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *HTTPJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan httpJobModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateSchedule(plan.CronExpression, plan.IntervalSeconds); err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}

	apiReq, diags := httpJobModelToRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.CreateJob(ctx, apiReq)
	if err != nil {
		appendAPIError(&resp.Diagnostics, "creating HTTP job", err)
		return
	}

	// The create body does not wire tags; use the dedicated endpoint.
	if err := r.client.SetJobTags(ctx, job.ID, apiReq.Tags); err != nil {
		appendAPIError(&resp.Diagnostics, "setting tags on HTTP job", err)
		return
	}

	// Save planned tags before httpJobResponseToModel overwrites them with the
	// create response (which has no tags since tags are set via a separate endpoint).
	plannedTags := plan.Tags

	resp.Diagnostics.Append(httpJobResponseToModel(ctx, job, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Tags = plannedTags
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HTTPJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state httpJobModel
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
		appendAPIError(&resp.Diagnostics, "reading HTTP job", err)
		return
	}

	resp.Diagnostics.Append(httpJobResponseToModel(ctx, job, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HTTPJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan httpJobModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state httpJobModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateSchedule(plan.CronExpression, plan.IntervalSeconds); err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}

	apiReq, diags := httpJobModelToRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.UpdateJob(ctx, state.ID.ValueString(), apiReq)
	if err != nil {
		appendAPIError(&resp.Diagnostics, "updating HTTP job", err)
		return
	}

	// Sync tags via the dedicated endpoint — update body does not wire tags.
	if err := r.client.SetJobTags(ctx, state.ID.ValueString(), apiReq.Tags); err != nil {
		appendAPIError(&resp.Diagnostics, "setting tags on HTTP job", err)
		return
	}

	// Save planned tags before httpJobResponseToModel overwrites them with the
	// update response (which has no tags).
	plannedTags := plan.Tags

	resp.Diagnostics.Append(httpJobResponseToModel(ctx, job, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Tags = plannedTags
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HTTPJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state httpJobModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteJob(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			appendAPIError(&resp.Diagnostics, "deleting HTTP job", err)
		}
	}
}

func (r *HTTPJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	job, err := r.client.GetJob(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Job not found", fmt.Sprintf("No HTTP job with id %q was found.", req.ID))
			return
		}
		appendAPIError(&resp.Diagnostics, "importing HTTP job", err)
		return
	}
	if job.Kind != "http" {
		resp.Diagnostics.AddError("Wrong resource type",
			fmt.Sprintf("Job %q has kind %q; import it with steadycron_heartbeat_monitor instead.", req.ID, job.Kind))
		return
	}

	var state httpJobModel
	resp.Diagnostics.Append(httpJobResponseToModel(ctx, job, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func httpJobModelToRequest(ctx context.Context, m httpJobModel) (client.UpsertJobRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := client.UpsertJobRequest{
		Kind:          "http",
		Name:          m.Name.ValueString(),
		Description:   m.Description.ValueString(),
		Method:        m.Method.ValueString(),
		URL:           m.URL.ValueString(),
		Timezone:      m.Timezone.ValueString(),
		SkipIfRunning: boolPtr(m.SkipIfRunning.ValueBool()),
	}
	if !m.CronExpression.IsNull() && !m.CronExpression.IsUnknown() {
		req.ScheduleKind = "cron"
		req.CronExpression = m.CronExpression.ValueString()
	} else {
		req.ScheduleKind = "interval"
		v := m.IntervalSeconds.ValueInt64()
		req.IntervalSeconds = &v
	}
	if !m.TimeoutSeconds.IsNull() && !m.TimeoutSeconds.IsUnknown() {
		v := m.TimeoutSeconds.ValueInt64()
		req.TimeoutSeconds = &v
	}
	if !m.MaxRetries.IsNull() {
		v := m.MaxRetries.ValueInt64()
		req.MaxRetries = &v
	}
	if !m.RetryBackoffSeconds.IsNull() {
		v := m.RetryBackoffSeconds.ValueInt64()
		req.RetryBackoffSeconds = &v
	}
	if !m.Body.IsNull() {
		req.Body = m.Body.ValueString()
	}

	var headers map[string]string
	diags.Append(m.Headers.ElementsAs(ctx, &headers, false)...)
	req.Headers = headers

	var tags []string
	diags.Append(m.Tags.ElementsAs(ctx, &tags, false)...)
	req.Tags = tags

	return req, diags
}

func httpJobResponseToModel(ctx context.Context, job *client.JobResponse, m *httpJobModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(job.ID)
	m.Name = types.StringValue(job.Name)
	m.Description = types.StringValue(stringPtrOrEmpty(job.Description))
	m.Timezone = types.StringValue(job.Timezone)
	m.SkipIfRunning = types.BoolValue(job.SkipIfRunning)

	if job.CronExpression != nil {
		m.CronExpression = types.StringValue(*job.CronExpression)
		m.IntervalSeconds = types.Int64Null()
	} else if job.IntervalSeconds != nil {
		m.IntervalSeconds = types.Int64Value(*job.IntervalSeconds)
		m.CronExpression = types.StringNull()
	}

	if job.Method != nil {
		m.Method = types.StringValue(*job.Method)
	}
	if job.URL != nil {
		m.URL = types.StringValue(*job.URL)
	}
	if job.Body != nil {
		m.Body = types.StringValue(*job.Body)
	} else {
		m.Body = types.StringValue("")
	}
	if job.TimeoutSeconds != nil {
		m.TimeoutSeconds = types.Int64Value(*job.TimeoutSeconds)
	} else {
		m.TimeoutSeconds = types.Int64Null()
	}
	if job.MaxRetries != nil {
		m.MaxRetries = types.Int64Value(*job.MaxRetries)
	} else {
		m.MaxRetries = types.Int64Value(0)
	}
	if job.RetryBackoffSeconds != nil {
		m.RetryBackoffSeconds = types.Int64Value(*job.RetryBackoffSeconds)
	} else {
		m.RetryBackoffSeconds = types.Int64Value(30)
	}

	// Headers
	headerAttrs := make(map[string]attr.Value, len(job.Headers))
	for k, v := range job.Headers {
		headerAttrs[k] = types.StringValue(v)
	}
	headersVal, d := types.MapValue(types.StringType, headerAttrs)
	diags.Append(d...)
	m.Headers = headersVal

	// Tags
	tagElems := make([]attr.Value, len(job.Tags))
	for i, t := range job.Tags {
		tagElems[i] = types.StringValue(t.ID)
	}
	tagsVal, d := types.SetValue(types.StringType, tagElems)
	diags.Append(d...)
	m.Tags = tagsVal

	// Computed status fields
	m.Status = types.StringPointerValue(job.Status)
	m.NextFireAt = types.StringPointerValue(job.NextFireAt)
	m.LastFireAt = types.StringPointerValue(job.LastFireAt)
	m.CreatedAt = types.StringValue(normalizeTimestamp(job.CreatedAt))
	m.UpdatedAt = types.StringValue(normalizeTimestamp(job.UpdatedAt))

	return diags
}
