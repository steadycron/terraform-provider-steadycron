package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ resource.Resource = &AlertRuleResource{}
var _ resource.ResourceWithImportState = &AlertRuleResource{}

func NewAlertRuleResource() resource.Resource {
	return &AlertRuleResource{}
}

type AlertRuleResource struct {
	client *client.Client
}

type alertRuleModel struct {
	ID                 types.String  `tfsdk:"id"`
	JobID              types.String  `tfsdk:"job_id"`
	ChannelID          types.String  `tfsdk:"channel_id"`
	Trigger            types.String  `tfsdk:"trigger"`
	Threshold          types.Int64   `tfsdk:"threshold"`
	Severity           types.String  `tfsdk:"severity"`
	DedupWindowSeconds types.Int64   `tfsdk:"dedup_window_seconds"`
	ParamFactor        types.Float64 `tfsdk:"param_factor"`
	ParamMinSamples    types.Int64   `tfsdk:"param_min_baseline_samples"`
	CreatedAt          types.String  `tfsdk:"created_at"`
}

func (r *AlertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *AlertRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SteadyCron alert rule — links a job to an alert channel with a trigger condition.\n\n" +
			"The logical identity of a rule is `(job_id, channel_id, trigger)`; the server enforces uniqueness on this triple.\n\n" +
			"`threshold` is required when `trigger = \"on_n_consecutive\"`.\n\n" +
			"`param_factor` and `param_min_baseline_samples` are used for `on_slow_run` and `on_size_anomaly` triggers.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"job_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the job this rule applies to.",
			},
			"channel_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the alert channel to deliver to.",
			},
			"trigger": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Alert trigger. One of: `on_failure`, `on_n_consecutive`, `on_missed_heartbeat`, `on_recovery`, `on_slow_run`, `on_size_anomaly`.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"on_failure",
						"on_n_consecutive",
						"on_missed_heartbeat",
						"on_recovery",
						"on_slow_run",
						"on_size_anomaly",
					),
				},
			},
			"threshold": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Consecutive failure count. Required when `trigger = \"on_n_consecutive\"`; ignored otherwise.",
			},
			"severity": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("P2"),
				MarkdownDescription: "Alert severity. One of: `P1`, `P2`, `P3`. `P1` bypasses quiet hours. Defaults to `P2`.",
				Validators: []validator.String{
					stringvalidator.OneOf("P1", "P2", "P3"),
				},
			},
			"dedup_window_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(300),
				MarkdownDescription: "Deduplication window in seconds. Defaults to `300` (5 minutes).",
			},
			"param_factor": schema.Float64Attribute{
				Optional:            true,
				MarkdownDescription: "Anomaly factor (for `on_slow_run` / `on_size_anomaly`). Multiplier above the baseline that triggers the alert.",
			},
			"param_min_baseline_samples": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum baseline samples required before anomaly detection fires (for `on_slow_run` / `on_size_anomaly`).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *AlertRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AlertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateAlertRule(plan); err != nil {
		resp.Diagnostics.AddError("Invalid alert rule", err.Error())
		return
	}

	rule, err := r.client.CreateAlertRule(ctx, plan.JobID.ValueString(), ruleModelToRequest(plan))
	if err != nil {
		appendAPIError(&resp.Diagnostics, "creating alert rule", err)
		return
	}

	ruleResponseToModel(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetAlertRule(ctx, state.JobID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		appendAPIError(&resp.Diagnostics, "reading alert rule", err)
		return
	}

	ruleResponseToModel(rule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state alertRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateAlertRule(plan); err != nil {
		resp.Diagnostics.AddError("Invalid alert rule", err.Error())
		return
	}

	rule, err := r.client.UpdateAlertRule(ctx, state.JobID.ValueString(), state.ID.ValueString(), ruleModelToRequest(plan))
	if err != nil {
		appendAPIError(&resp.Diagnostics, "updating alert rule", err)
		return
	}

	ruleResponseToModel(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAlertRule(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			appendAPIError(&resp.Diagnostics, "deleting alert rule", err)
		}
	}
}

func (r *AlertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Scan all jobs to locate the rule by ID (no flat GET-by-ID endpoint exists).
	rule, err := r.client.FindAlertRuleByID(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Alert rule not found", fmt.Sprintf("No alert rule with id %q was found.", req.ID))
			return
		}
		appendAPIError(&resp.Diagnostics, "importing alert rule", err)
		return
	}

	var state alertRuleModel
	ruleResponseToModel(rule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func validateAlertRule(m alertRuleModel) error {
	if m.Trigger.ValueString() == "on_n_consecutive" {
		if m.Threshold.IsNull() || m.Threshold.IsUnknown() {
			return fmt.Errorf("threshold is required when trigger = \"on_n_consecutive\"")
		}
		if m.Threshold.ValueInt64() < 1 {
			return fmt.Errorf("threshold must be >= 1")
		}
	}
	return nil
}

func ruleModelToRequest(m alertRuleModel) client.UpsertAlertRuleRequest {
	req := client.UpsertAlertRuleRequest{
		ChannelID: m.ChannelID.ValueString(),
		Trigger:   m.Trigger.ValueString(),
		Severity:  m.Severity.ValueString(),
	}
	if !m.DedupWindowSeconds.IsNull() {
		v := m.DedupWindowSeconds.ValueInt64()
		req.DedupWindowSeconds = &v
	}

	// Threshold, factor, and min_baseline_samples all go in the Params object.
	if !m.Threshold.IsNull() || !m.ParamFactor.IsNull() || !m.ParamMinSamples.IsNull() {
		params := &client.Params{}
		if !m.Threshold.IsNull() && !m.Threshold.IsUnknown() {
			v := m.Threshold.ValueInt64()
			params.Threshold = &v
		}
		if !m.ParamFactor.IsNull() {
			v := m.ParamFactor.ValueFloat64()
			params.Factor = &v
		}
		if !m.ParamMinSamples.IsNull() {
			v := m.ParamMinSamples.ValueInt64()
			params.MinBaselineSamples = &v
		}
		req.Params = params
	}
	return req
}

func ruleResponseToModel(rule *client.AlertRuleResponse, m *alertRuleModel) {
	m.ID = types.StringValue(rule.ID)
	m.JobID = types.StringValue(rule.JobID)
	m.ChannelID = types.StringValue(rule.ChannelID)
	m.Trigger = types.StringValue(rule.Trigger)
	m.Severity = types.StringValue(rule.Severity)
	m.DedupWindowSeconds = types.Int64Value(rule.DedupWindowSeconds)
	m.CreatedAt = types.StringValue(normalizeTimestamp(rule.CreatedAt))

	if rule.Params != nil {
		if rule.Params.Threshold != nil {
			m.Threshold = types.Int64Value(*rule.Params.Threshold)
		} else {
			m.Threshold = types.Int64Null()
		}
		if rule.Params.Factor != nil {
			m.ParamFactor = types.Float64Value(*rule.Params.Factor)
		} else {
			m.ParamFactor = types.Float64Null()
		}
		if rule.Params.MinBaselineSamples != nil {
			m.ParamMinSamples = types.Int64Value(*rule.Params.MinBaselineSamples)
		} else {
			m.ParamMinSamples = types.Int64Null()
		}
	} else {
		m.Threshold = types.Int64Null()
		m.ParamFactor = types.Float64Null()
		m.ParamMinSamples = types.Int64Null()
	}
}
