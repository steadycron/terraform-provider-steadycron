package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ datasource.DataSource = &HeartbeatMonitorDataSource{}

func NewHeartbeatMonitorDataSource() datasource.DataSource {
	return &HeartbeatMonitorDataSource{}
}

type HeartbeatMonitorDataSource struct {
	client *client.Client
}

func (d *HeartbeatMonitorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat_monitor"
}

func (d *HeartbeatMonitorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a SteadyCron heartbeat monitor by its server-assigned `id`.",
		Attributes: map[string]schema.Attribute{
			"id":                       schema.StringAttribute{Required: true},
			"name":                     schema.StringAttribute{Computed: true},
			"description":              schema.StringAttribute{Computed: true},
			"runbook_notes":            schema.StringAttribute{Computed: true},
			"runbook_url":              schema.StringAttribute{Computed: true},
			"cron_expression":          schema.StringAttribute{Computed: true},
			"interval_seconds":         schema.Int64Attribute{Computed: true},
			"timezone":                 schema.StringAttribute{Computed: true},
			"grace_seconds":            schema.Int64Attribute{Computed: true},
			"stuck_run_detection":      schema.BoolAttribute{Computed: true},
			"max_run_duration_seconds": schema.Int64Attribute{Computed: true},
			"misfire_policy":           schema.StringAttribute{Computed: true, MarkdownDescription: "Misfire policy: `do_nothing` or `fire_once_now`."},
			"tags": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"ping_url":   schema.StringAttribute{Computed: true, Sensitive: true},
			"token":      schema.StringAttribute{Computed: true, Sensitive: true},
			"key":        schema.StringAttribute{Computed: true, MarkdownDescription: "Stable job key used by code-monitoring SDKs."},
			"status":     schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *HeartbeatMonitorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HeartbeatMonitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
		StuckRunDetection     types.Bool   `tfsdk:"stuck_run_detection"`
		MaxRunDurationSeconds types.Int64  `tfsdk:"max_run_duration_seconds"`
		MisfirePolicy         types.String `tfsdk:"misfire_policy"`
		Tags                  types.Set    `tfsdk:"tags"`
		PingURL               types.String `tfsdk:"ping_url"`
		Token                 types.String `tfsdk:"token"`
		Key                   types.String `tfsdk:"key"`
		Status                types.String `tfsdk:"status"`
		CreatedAt             types.String `tfsdk:"created_at"`
		UpdatedAt             types.String `tfsdk:"updated_at"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := d.client.GetJob(ctx, config.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Monitor not found", fmt.Sprintf("No heartbeat monitor with id %q was found.", config.ID.ValueString()))
			return
		}
		appendAPIError(&resp.Diagnostics, "reading heartbeat monitor data source", err)
		return
	}
	if job.Kind != "heartbeat" {
		resp.Diagnostics.AddError("Wrong kind", fmt.Sprintf("Job %q has kind %q; use steadycron_http_job data source instead.", config.ID.ValueString(), job.Kind))
		return
	}

	config.Name = types.StringValue(job.Name)
	config.Description = types.StringPointerValue(job.Description)
	config.RunbookNotes = types.StringPointerValue(job.RunbookNotes)
	config.RunbookUrl = types.StringPointerValue(job.RunbookUrl)
	config.Timezone = types.StringValue(job.Timezone)
	config.GraceSeconds = types.Int64Value(job.GraceSeconds)
	config.StuckRunDetection = types.BoolValue(job.StuckRunDetection)
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
		u := strings.TrimRight(job.PingUrls.Success, "/")
		if i := strings.LastIndex(u, "/"); i >= 0 {
			config.Token = types.StringValue(u[i+1:])
		} else {
			config.Token = types.StringValue(u)
		}
	} else {
		config.PingURL = types.StringNull()
		config.Token = types.StringNull()
	}

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
