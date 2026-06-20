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

var _ datasource.DataSource = &HTTPJobDataSource{}

func NewHTTPJobDataSource() datasource.DataSource {
	return &HTTPJobDataSource{}
}

type HTTPJobDataSource struct {
	client *client.Client
}

func (d *HTTPJobDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_job"
}

func (d *HTTPJobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a SteadyCron HTTP job by its server-assigned `id`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Server-assigned UUID of the job.",
			},
			"name":                  schema.StringAttribute{Computed: true},
			"description":           schema.StringAttribute{Computed: true},
			"runbook_notes":         schema.StringAttribute{Computed: true},
			"runbook_url":           schema.StringAttribute{Computed: true},
			"method":                schema.StringAttribute{Computed: true},
			"url":                   schema.StringAttribute{Computed: true},
			"cron_expression":       schema.StringAttribute{Computed: true},
			"interval_seconds":      schema.Int64Attribute{Computed: true},
			"timezone":              schema.StringAttribute{Computed: true},
			"timeout_seconds":       schema.Int64Attribute{Computed: true},
			"max_retries":           schema.Int64Attribute{Computed: true},
			"retry_backoff_seconds": schema.Int64Attribute{Computed: true},
			"headers": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"body":            schema.StringAttribute{Computed: true},
			"skip_if_running": schema.BoolAttribute{Computed: true},
			"misfire_policy":  schema.StringAttribute{Computed: true, MarkdownDescription: "Misfire policy: `do_nothing` or `fire_once_now`."},
			"tags": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"key":          schema.StringAttribute{Computed: true, MarkdownDescription: "Stable job key used by code-monitoring SDKs."},
			"status":       schema.StringAttribute{Computed: true},
			"next_fire_at": schema.StringAttribute{Computed: true},
			"last_fire_at": schema.StringAttribute{Computed: true},
			"created_at":   schema.StringAttribute{Computed: true},
			"updated_at":   schema.StringAttribute{Computed: true},
		},
	}
}

func (d *HTTPJobDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HTTPJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config struct {
		ID types.String `tfsdk:"id"`
		// other fields will be populated
		Name                types.String `tfsdk:"name"`
		Description         types.String `tfsdk:"description"`
		RunbookNotes        types.String `tfsdk:"runbook_notes"`
		RunbookUrl          types.String `tfsdk:"runbook_url"`
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
		MisfirePolicy       types.String `tfsdk:"misfire_policy"`
		Tags                types.Set    `tfsdk:"tags"`
		Key                 types.String `tfsdk:"key"`
		Status              types.String `tfsdk:"status"`
		NextFireAt          types.String `tfsdk:"next_fire_at"`
		LastFireAt          types.String `tfsdk:"last_fire_at"`
		CreatedAt           types.String `tfsdk:"created_at"`
		UpdatedAt           types.String `tfsdk:"updated_at"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := d.client.GetJob(ctx, config.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Job not found", fmt.Sprintf("No job with id %q was found.", config.ID.ValueString()))
			return
		}
		appendAPIError(&resp.Diagnostics, "reading HTTP job data source", err)
		return
	}
	if job.Kind != "http" {
		resp.Diagnostics.AddError("Wrong kind", fmt.Sprintf("Job %q has kind %q; use steadycron_heartbeat_monitor data source instead.", config.ID.ValueString(), job.Kind))
		return
	}

	config.Name = types.StringValue(job.Name)
	config.Description = types.StringPointerValue(job.Description)
	config.RunbookNotes = types.StringPointerValue(job.RunbookNotes)
	config.RunbookUrl = types.StringPointerValue(job.RunbookUrl)
	config.Timezone = types.StringValue(job.Timezone)
	config.SkipIfRunning = types.BoolValue(job.SkipIfRunning)

	if job.CronExpression != nil {
		config.CronExpression = types.StringValue(*job.CronExpression)
		config.IntervalSeconds = types.Int64Null()
	} else if job.IntervalSeconds != nil {
		config.IntervalSeconds = types.Int64Value(*job.IntervalSeconds)
		config.CronExpression = types.StringNull()
	}
	if job.Method != nil {
		config.Method = types.StringValue(*job.Method)
	}
	if job.URL != nil {
		config.URL = types.StringValue(*job.URL)
	}
	if job.Body != nil {
		config.Body = types.StringValue(*job.Body)
	}
	if job.TimeoutSeconds != nil {
		config.TimeoutSeconds = types.Int64Value(*job.TimeoutSeconds)
	}
	if job.MaxRetries != nil {
		config.MaxRetries = types.Int64Value(*job.MaxRetries)
	}
	if job.RetryBackoffSeconds != nil {
		config.RetryBackoffSeconds = types.Int64Value(*job.RetryBackoffSeconds)
	}

	headerAttrs := make(map[string]attr.Value, len(job.Headers))
	for k, v := range job.Headers {
		headerAttrs[k] = types.StringValue(v)
	}
	hv, d2 := types.MapValue(types.StringType, headerAttrs)
	resp.Diagnostics.Append(d2...)
	config.Headers = hv

	tagElems := make([]attr.Value, len(job.Tags))
	for i, t := range job.Tags {
		tagElems[i] = types.StringValue(t.ID)
	}
	tv, d2 := types.SetValue(types.StringType, tagElems)
	resp.Diagnostics.Append(d2...)
	config.Tags = tv

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
	config.NextFireAt = types.StringPointerValue(job.NextFireAt)
	config.LastFireAt = types.StringPointerValue(job.LastFireAt)
	config.CreatedAt = types.StringValue(normalizeTimestamp(job.CreatedAt))
	config.UpdatedAt = types.StringValue(normalizeTimestamp(job.UpdatedAt))

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
