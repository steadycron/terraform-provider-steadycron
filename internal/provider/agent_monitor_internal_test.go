package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// baseAgentModel is a minimal valid plan: cron-scheduled, no outcome rules set.
func baseAgentModel() agentMonitorModel {
	return agentMonitorModel{
		Name:                  types.StringValue("nightly-triage"),
		Description:           types.StringValue(""),
		CronExpression:        types.StringValue("0 3 * * *"),
		IntervalSeconds:       types.Int64Null(),
		Timezone:              types.StringValue("Europe/Berlin"),
		GraceSeconds:          types.Int64Value(600),
		MaxRunDurationSeconds: types.Int64Value(1800),
		MisfirePolicy:         types.StringValue("do_nothing"),
		Tags:                  types.SetValueMust(types.StringType, nil),
		Key:                   types.StringNull(),

		ReportRequired:          types.BoolValue(true),
		ItemsLabel:              types.StringNull(),
		RuleEmptyResultEnabled:  types.BoolValue(true),
		RuleMaxCostUsdPerRun:    types.Float64Null(),
		RuleMaxCostUsdPerPeriod: types.Float64Null(),
		RuleCostPeriod:          types.StringNull(),
		RuleMaxSteps:            types.Int64Null(),
		RuleMaxToolCalls:        types.Int64Null(),
		RuleMaxDurationMs:       types.Int64Null(),
	}
}

func TestAgentModelToRequest_createOmitsUnsetCeilings(t *testing.T) {
	// On create, an omitted ceiling must stay omitted — sending 0 would be indistinguishable
	// from "clear", and the server would have nothing to clear anyway.
	req, diags := agentModelToRequest(context.Background(), baseAgentModel(), false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if req.Kind != "agent" {
		t.Errorf("Kind = %q, want %q", req.Kind, "agent")
	}
	if req.RuleMaxCostUsdPerRun != nil {
		t.Errorf("RuleMaxCostUsdPerRun = %v, want nil on create", *req.RuleMaxCostUsdPerRun)
	}
	if req.RuleMaxSteps != nil {
		t.Errorf("RuleMaxSteps = %v, want nil on create", *req.RuleMaxSteps)
	}
	if req.ItemsLabel != nil {
		t.Errorf("ItemsLabel = %q, want nil on create", *req.ItemsLabel)
	}
	// stuck_run_detection is never sent: the server forces it true for this kind.
	if req.StuckRunDetection != nil {
		t.Errorf("StuckRunDetection = %v, want nil (server-forced)", *req.StuckRunDetection)
	}
}

func TestAgentModelToRequest_updateSendsZeroToClearCeilings(t *testing.T) {
	// The whole reason this is a distinct code path: Terraform always sends full desired state,
	// so removing `rule_max_steps = 40` from a config has to reach the server as an explicit
	// clear. The API's documented sentinel for that is 0 — an omitted field means "unchanged".
	req, diags := agentModelToRequest(context.Background(), baseAgentModel(), true)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	for name, got := range map[string]*int64{
		"RuleMaxSteps":      req.RuleMaxSteps,
		"RuleMaxToolCalls":  req.RuleMaxToolCalls,
		"RuleMaxDurationMs": req.RuleMaxDurationMs,
	} {
		if got == nil {
			t.Errorf("%s = nil, want an explicit 0 on update", name)
		} else if *got != 0 {
			t.Errorf("%s = %d, want 0", name, *got)
		}
	}

	if req.RuleMaxCostUsdPerRun == nil || *req.RuleMaxCostUsdPerRun != 0 {
		t.Errorf("RuleMaxCostUsdPerRun = %v, want an explicit 0 on update", req.RuleMaxCostUsdPerRun)
	}
	if req.ItemsLabel == nil || *req.ItemsLabel != "" {
		t.Errorf("ItemsLabel = %v, want an explicit empty string on update", req.ItemsLabel)
	}
}

func TestAgentModelToRequest_updateOmitsCreateOnlyFlags(t *testing.T) {
	// Both are RequiresReplace precisely because PATCH does not accept them.
	req, _ := agentModelToRequest(context.Background(), baseAgentModel(), true)

	if req.ReportRequired != nil {
		t.Errorf("ReportRequired = %v, want nil on update", *req.ReportRequired)
	}
	if req.RuleEmptyResultEnabled != nil {
		t.Errorf("RuleEmptyResultEnabled = %v, want nil on update", *req.RuleEmptyResultEnabled)
	}
}

func TestAgentModelToRequest_createCarriesEveryOutcomeRule(t *testing.T) {
	m := baseAgentModel()
	m.ItemsLabel = types.StringValue("tickets")
	m.ReportRequired = types.BoolValue(false)
	m.RuleEmptyResultEnabled = types.BoolValue(false)
	m.RuleMaxCostUsdPerRun = types.Float64Value(0.5)
	m.RuleMaxCostUsdPerPeriod = types.Float64Value(20)
	m.RuleCostPeriod = types.StringValue("day")
	m.RuleMaxSteps = types.Int64Value(40)
	m.RuleMaxToolCalls = types.Int64Value(100)
	m.RuleMaxDurationMs = types.Int64Value(900000)

	req, _ := agentModelToRequest(context.Background(), m, false)

	if req.ItemsLabel == nil || *req.ItemsLabel != "tickets" {
		t.Errorf("ItemsLabel = %v, want \"tickets\"", req.ItemsLabel)
	}
	if req.ReportRequired == nil || *req.ReportRequired {
		t.Errorf("ReportRequired = %v, want false", req.ReportRequired)
	}
	if req.RuleEmptyResultEnabled == nil || *req.RuleEmptyResultEnabled {
		t.Errorf("RuleEmptyResultEnabled = %v, want false", req.RuleEmptyResultEnabled)
	}
	if req.RuleMaxCostUsdPerRun == nil || *req.RuleMaxCostUsdPerRun != 0.5 {
		t.Errorf("RuleMaxCostUsdPerRun = %v, want 0.5", req.RuleMaxCostUsdPerRun)
	}
	if req.RuleCostPeriod == nil || *req.RuleCostPeriod != "day" {
		t.Errorf("RuleCostPeriod = %v, want \"day\"", req.RuleCostPeriod)
	}
	if req.RuleMaxDurationMs == nil || *req.RuleMaxDurationMs != 900000 {
		t.Errorf("RuleMaxDurationMs = %v, want 900000", req.RuleMaxDurationMs)
	}
}

func TestAgentModelToRequest_costPeriodDroppedWhenTheCeilingIsCleared(t *testing.T) {
	// Clearing the ceiling clears the period server-side, so sending a period alongside a
	// zeroed ceiling would be a contradiction.
	m := baseAgentModel()
	m.RuleCostPeriod = types.StringValue("month")
	m.RuleMaxCostUsdPerPeriod = types.Float64Null()

	req, _ := agentModelToRequest(context.Background(), m, true)

	if req.RuleCostPeriod != nil {
		t.Errorf("RuleCostPeriod = %q, want nil when the period ceiling is cleared", *req.RuleCostPeriod)
	}
	if req.RuleMaxCostUsdPerPeriod == nil || *req.RuleMaxCostUsdPerPeriod != 0 {
		t.Errorf("RuleMaxCostUsdPerPeriod = %v, want an explicit 0", req.RuleMaxCostUsdPerPeriod)
	}
}

func TestAgentModelToRequest_intervalSchedule(t *testing.T) {
	m := baseAgentModel()
	m.CronExpression = types.StringNull()
	m.IntervalSeconds = types.Int64Value(3600)

	req, _ := agentModelToRequest(context.Background(), m, false)

	if req.ScheduleKind != "interval" {
		t.Errorf("ScheduleKind = %q, want %q", req.ScheduleKind, "interval")
	}
	if req.IntervalSeconds == nil || *req.IntervalSeconds != 3600 {
		t.Errorf("IntervalSeconds = %v, want 3600", req.IntervalSeconds)
	}
}

func TestValidateAgentCostRules(t *testing.T) {
	withPeriodOnly := baseAgentModel()
	withPeriodOnly.RuleCostPeriod = types.StringValue("month")
	if err := validateAgentCostRules(withPeriodOnly); err == nil {
		t.Error("expected an error for a cost period with no per-period ceiling")
	}

	withBoth := baseAgentModel()
	withBoth.RuleCostPeriod = types.StringValue("month")
	withBoth.RuleMaxCostUsdPerPeriod = types.Float64Value(20)
	if err := validateAgentCostRules(withBoth); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := validateAgentCostRules(baseAgentModel()); err != nil {
		t.Errorf("unexpected error for a monitor with no cost rules: %v", err)
	}
}

func TestTokenFromPingURL(t *testing.T) {
	// Agent monitors are shown the /success URL rather than the bare token URL, so the token
	// is one segment further back than it is for a heartbeat.
	cases := map[string]string{
		"https://ping.steadycron.com/abc123/success": "abc123",
		"https://ping.steadycron.com/abc123/start":   "abc123",
		"https://ping.steadycron.com/abc123/fail":    "abc123",
		"https://ping.steadycron.com/abc123":         "abc123",
		"https://ping.steadycron.com/abc123/":        "abc123",
	}

	for input, want := range cases {
		if got := tokenFromPingURL(input); got != want {
			t.Errorf("tokenFromPingURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResourceSuffixForKind(t *testing.T) {
	cases := map[string]string{
		"agent":     "agent_monitor",
		"heartbeat": "heartbeat_monitor",
		"http":      "http_job",
	}

	for kind, want := range cases {
		if got := resourceSuffixForKind(kind); got != want {
			t.Errorf("resourceSuffixForKind(%q) = %q, want %q", kind, got, want)
		}
	}
}
