package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ datasource.DataSource = &AgentMonitorDataSource{}

func NewAgentMonitorDataSource() datasource.DataSource {
	return &AgentMonitorDataSource{}
}

type AgentMonitorDataSource struct {
	client *client.Client
}

func (d *AgentMonitorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_monitor"
}

func (d *AgentMonitorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a SteadyCron AI agent monitor by its server-assigned `id`, including its outcome rules.",
		Attributes: map[string]schema.Attribute{
			"id":                       schema.StringAttribute{Required: true},
			"name":                     schema.StringAttribute{Computed: true},
			"description":              schema.StringAttribute{Computed: true},
			"runbook_notes":            schema.StringAttribute{Computed: true},
			"runbook_url":              schema.StringAttribute{Computed: true},
			"cron_expression":          schema.StringAttribute{Computed: true},
			"interval_seconds":         schema.Int64Attribute{Computed: true},
			"timezone":                 schema.StringAttribute{Computed: true},
			"grace_seconds":            schema.Int64Attribute{Computed: true, MarkdownDescription: "How long after the scheduled time the run may still `/start`."},
			"max_run_duration_seconds": schema.Int64Attribute{Computed: true, MarkdownDescription: "How long a started run may take before it counts as abandoned."},
			"misfire_policy":           schema.StringAttribute{Computed: true, MarkdownDescription: "Misfire policy: `do_nothing` or `fire_once_now`."},
			"tags": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"report_required":              schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether a success ping with no run report is recorded as unverified."},
			"items_label":                  schema.StringAttribute{Computed: true, MarkdownDescription: "What this agent produces, e.g. `tickets`."},
			"rule_empty_result_enabled":    schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether a run reporting zero items produced is a failure."},
			"rule_max_cost_usd_per_run":    schema.Float64Attribute{Computed: true, MarkdownDescription: "Per-run spend ceiling in USD; null when no ceiling is set."},
			"rule_max_cost_usd_per_period": schema.Float64Attribute{Computed: true, MarkdownDescription: "Per-period spend ceiling in USD; null when no ceiling is set."},
			"rule_cost_period":             schema.StringAttribute{Computed: true, MarkdownDescription: "`day` or `month` — the window the per-period ceiling sums over."},
			"rule_max_steps":               schema.Int64Attribute{Computed: true},
			"rule_max_tool_calls":          schema.Int64Attribute{Computed: true},
			"rule_max_duration_ms":         schema.Int64Attribute{Computed: true},

			"ping_url":       schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Completion URL to POST the run report to."},
			"start_ping_url": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "URL to ping when a run begins."},
			"fail_ping_url":  schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "URL to POST a failure report to."},
			"token":          schema.StringAttribute{Computed: true, Sensitive: true},
			"key":            schema.StringAttribute{Computed: true, MarkdownDescription: "Stable job key used by SDKs and the MCP server."},
			"status":         schema.StringAttribute{Computed: true},
			"created_at":     schema.StringAttribute{Computed: true},
			"updated_at":     schema.StringAttribute{Computed: true},
		},
	}
}

func (d *AgentMonitorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *AgentMonitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config struct {
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

		ReportRequired          types.Bool    `tfsdk:"report_required"`
		ItemsLabel              types.String  `tfsdk:"items_label"`
		RuleEmptyResultEnabled  types.Bool    `tfsdk:"rule_empty_result_enabled"`
		RuleMaxCostUsdPerRun    types.Float64 `tfsdk:"rule_max_cost_usd_per_run"`
		RuleMaxCostUsdPerPeriod types.Float64 `tfsdk:"rule_max_cost_usd_per_period"`
		RuleCostPeriod          types.String  `tfsdk:"rule_cost_period"`
		RuleMaxSteps            types.Int64   `tfsdk:"rule_max_steps"`
		RuleMaxToolCalls        types.Int64   `tfsdk:"rule_max_tool_calls"`
		RuleMaxDurationMs       types.Int64   `tfsdk:"rule_max_duration_ms"`

		PingURL      types.String `tfsdk:"ping_url"`
		StartPingURL types.String `tfsdk:"start_ping_url"`
		FailPingURL  types.String `tfsdk:"fail_ping_url"`
		Token        types.String `tfsdk:"token"`
		Key          types.String `tfsdk:"key"`
		Status       types.String `tfsdk:"status"`
		CreatedAt    types.String `tfsdk:"created_at"`
		UpdatedAt    types.String `tfsdk:"updated_at"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := d.client.GetJob(ctx, config.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Monitor not found", fmt.Sprintf("No agent monitor with id %q was found.", config.ID.ValueString()))
			return
		}
		appendAPIError(&resp.Diagnostics, "reading agent monitor data source", err)
		return
	}
	if job.Kind != "agent" {
		resp.Diagnostics.AddError("Wrong kind",
			fmt.Sprintf("Job %q has kind %q; use the steadycron_%s data source instead.",
				config.ID.ValueString(), job.Kind, resourceSuffixForKind(job.Kind)))
		return
	}

	config.Name = types.StringValue(job.Name)
	config.Description = types.StringPointerValue(job.Description)
	config.RunbookNotes = types.StringPointerValue(job.RunbookNotes)
	config.RunbookUrl = types.StringPointerValue(job.RunbookUrl)
	config.Timezone = types.StringValue(job.Timezone)
	config.GraceSeconds = types.Int64Value(job.GraceSeconds)
	config.MaxRunDurationSeconds = types.Int64Value(job.MaxRunDurationSeconds)

	if job.CronExpression != nil {
		config.CronExpression = types.StringValue(*job.CronExpression)
		config.IntervalSeconds = types.Int64Null()
	} else if job.IntervalSeconds != nil {
		config.IntervalSeconds = types.Int64Value(*job.IntervalSeconds)
		config.CronExpression = types.StringNull()
	}

	if job.PingUrls != nil && job.PingUrls.Success != "" {
		config.PingURL = types.StringValue(job.PingUrls.Success)
		config.StartPingURL = types.StringValue(job.PingUrls.Start)
		config.FailPingURL = types.StringValue(job.PingUrls.Fail)
		config.Token = types.StringValue(tokenFromPingURL(job.PingUrls.Success))
	} else {
		config.PingURL = types.StringNull()
		config.StartPingURL = types.StringNull()
		config.FailPingURL = types.StringNull()
		config.Token = types.StringNull()
	}

	config.ReportRequired = types.BoolValue(boolPtrOrDefault(job.ReportRequired, true))
	config.RuleEmptyResultEnabled = types.BoolValue(boolPtrOrDefault(job.RuleEmptyResultEnabled, true))
	config.ItemsLabel = types.StringPointerValue(job.ItemsLabel)
	config.RuleMaxCostUsdPerRun = types.Float64PointerValue(job.RuleMaxCostUsdPerRun)
	config.RuleMaxCostUsdPerPeriod = types.Float64PointerValue(job.RuleMaxCostUsdPerPeriod)
	config.RuleCostPeriod = types.StringPointerValue(job.RuleCostPeriod)
	config.RuleMaxSteps = types.Int64PointerValue(job.RuleMaxSteps)
	config.RuleMaxToolCalls = types.Int64PointerValue(job.RuleMaxToolCalls)
	config.RuleMaxDurationMs = types.Int64PointerValue(job.RuleMaxDurationMs)

	if job.JobKey != nil {
		config.Key = types.StringValue(*job.JobKey)
	} else {
		config.Key = types.StringNull()
	}

	if job.MisfirePolicy != "" {
		config.MisfirePolicy = types.StringValue(job.MisfirePolicy)
	} else {
		config.MisfirePolicy = types.StringValue("do_nothing")
	}

	config.Status = types.StringPointerValue(job.Status)
	config.CreatedAt = types.StringValue(normalizeTimestamp(job.CreatedAt))
	config.UpdatedAt = types.StringValue(normalizeTimestamp(job.UpdatedAt))

	tagElems := make([]attr.Value, len(job.Tags))
	for i, t := range job.Tags {
		tagElems[i] = types.StringValue(t.ID)
	}
	tv, d2 := types.SetValue(types.StringType, tagElems)
	resp.Diagnostics.Append(d2...)
	config.Tags = tv

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
