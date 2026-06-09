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
			"cron_expression":          schema.StringAttribute{Computed: true},
			"interval_seconds":         schema.Int64Attribute{Computed: true},
			"timezone":                 schema.StringAttribute{Computed: true},
			"grace_seconds":            schema.Int64Attribute{Computed: true},
			"stuck_run_detection":      schema.BoolAttribute{Computed: true},
			"max_run_duration_seconds": schema.Int64Attribute{Computed: true},
			"tags": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"ping_url": schema.StringAttribute{Computed: true, Sensitive: true},
			"token":    schema.StringAttribute{Computed: true, Sensitive: true},
			"status":          schema.StringAttribute{Computed: true},
			"last_success_at": schema.StringAttribute{Computed: true},
			"last_start_at":   schema.StringAttribute{Computed: true},
			"last_fail_at":    schema.StringAttribute{Computed: true},
			"created_at":      schema.StringAttribute{Computed: true},
			"updated_at":      schema.StringAttribute{Computed: true},
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
		CronExpression        types.String `tfsdk:"cron_expression"`
		IntervalSeconds       types.Int64  `tfsdk:"interval_seconds"`
		Timezone              types.String `tfsdk:"timezone"`
		GraceSeconds          types.Int64  `tfsdk:"grace_seconds"`
		StuckRunDetection     types.Bool   `tfsdk:"stuck_run_detection"`
		MaxRunDurationSeconds types.Int64  `tfsdk:"max_run_duration_seconds"`
		Tags                  types.Set    `tfsdk:"tags"`
		PingURL               types.String `tfsdk:"ping_url"`
		Token                 types.String `tfsdk:"token"`
		Status                types.String `tfsdk:"status"`
		LastSuccessAt         types.String `tfsdk:"last_success_at"`
		LastStartAt           types.String `tfsdk:"last_start_at"`
		LastFailAt            types.String `tfsdk:"last_fail_at"`
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
		appendAPIError(resp.Diagnostics, "reading heartbeat monitor data source", err)
		return
	}
	if job.Kind != "heartbeat" {
		resp.Diagnostics.AddError("Wrong kind", fmt.Sprintf("Job %q has kind %q; use steadycron_http_job data source instead.", config.ID.ValueString(), job.Kind))
		return
	}

	config.Name = types.StringValue(job.Name)
	config.Description = types.StringPointerValue(job.Description)
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

	config.PingURL = nullableString(job.PingURL)
	config.Token = nullableString(job.Token)
	config.Status = types.StringPointerValue(job.Status)
	config.LastSuccessAt = nullableString(job.LastSuccessAt)
	config.LastStartAt = nullableString(job.LastStartAt)
	config.LastFailAt = nullableString(job.LastFailAt)
	config.CreatedAt = types.StringValue(job.CreatedAt)
	config.UpdatedAt = types.StringValue(job.UpdatedAt)

	tagElems := make([]attr.Value, len(job.Tags))
	for i, t := range job.Tags {
		tagElems[i] = types.StringValue(t)
	}
	tv, d2 := types.SetValue(types.StringType, tagElems)
	resp.Diagnostics.Append(d2...)
	config.Tags = tv

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
