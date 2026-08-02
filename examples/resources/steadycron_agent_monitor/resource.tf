# AI agent monitor: like a heartbeat, but each run reports what it actually produced
# and SteadyCron judges it against the outcome rules below. A bare "exit 0" is no
# longer taken as proof that any work happened.
resource "steadycron_agent_monitor" "nightly_triage" {
  name        = "nightly-ticket-triage"
  description = "Claude agent that triages the overnight support queue"

  # key = stable identifier your code and the MCP server reference.
  # When omitted the server auto-generates a slug from the name (visible after apply).
  key = "nightly-ticket-triage"

  cron_expression = "0 3 * * *" # 03:00 daily
  timezone        = "Europe/Berlin"

  # Two clocks, not one:
  #   grace_seconds            — no /start by 03:10 → the schedule window is missed
  #   max_run_duration_seconds — a /start with no completion by then → abandoned
  # Sizing a single grace period to cover both is what made healthy agent runs
  # look missed every night.
  grace_seconds            = 600
  max_run_duration_seconds = 1800

  # What this agent produces. Labels itemsProduced across the dashboard and in
  # alert text — deciding it is the point, because it is what proves a run worked.
  items_label = "tickets"

  # Both default to true and are the reason the kind exists. Changing either forces
  # a new monitor: the API accepts them on create only.
  report_required           = true # a success ping with no report is "unverified"
  rule_empty_result_enabled = true # a run producing 0 tickets is a FAILURE

  # Spend ceilings in USD — the unit agents report and providers quote prices in.
  # Exceeding one alerts but does not fail the run: it is worth waking someone for,
  # it does not mean the work failed. An unpriced run never trips a cost rule.
  rule_max_cost_usd_per_run    = 0.50
  rule_max_cost_usd_per_period = 20
  rule_cost_period             = "month" # bucketed in this monitor's own timezone

  # Loop detection: alert when one run burns more steps or tool calls than it should.
  rule_max_steps      = 40
  rule_max_tool_calls = 100

  # Optional lower warning threshold beneath max_run_duration_seconds.
  rule_max_duration_ms = 900000 # 15 minutes
}

# ── The reporting contract ─────────────────────────────────────────────────────
# A run is an ordered PAIR of calls. /start is mandatory — a completion with no
# run open is rejected with 422, because without a start there is nothing to time.
#
#   curl "${steadycron_agent_monitor.nightly_triage.start_ping_url}"
#
#   ... the agent does its work ...
#
#   curl -X POST "${steadycron_agent_monitor.nightly_triage.ping_url}" \
#     -H 'Content-Type: application/json' \
#     -d '{"itemsProduced": 42, "model": "claude-opus-5", "costUsd": 0.84,
#          "summary": "42 tickets triaged, 3 escalated"}'
#
# Every report field is optional; itemsProduced is the one the empty-result rule
# reads. Report failures the same way against fail_ping_url.

output "start_ping_url" {
  value     = steadycron_agent_monitor.nightly_triage.start_ping_url
  sensitive = true
}

output "ping_url" {
  value       = steadycron_agent_monitor.nightly_triage.ping_url
  description = "POST the JSON run report here when the run completes."
  sensitive   = true
}

# ── Alerting ───────────────────────────────────────────────────────────────────
# Without these rules the evaluator still records the finding, but the alert it
# enqueues matches nothing — so the empty-run detection you came for stays silent.
resource "steadycron_alert_rule" "triage_empty_result" {
  job_id     = steadycron_agent_monitor.nightly_triage.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_empty_result"
  severity   = "P1"
}

resource "steadycron_alert_rule" "triage_cost_exceeded" {
  job_id     = steadycron_agent_monitor.nightly_triage.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_cost_exceeded"
}

# A missed agent run fires on_missed_heartbeat — one transition, two vocabularies.
resource "steadycron_alert_rule" "triage_missed" {
  job_id     = steadycron_agent_monitor.nightly_triage.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_missed_heartbeat"
  severity   = "P1"
}

resource "steadycron_alert_channel" "ops_email" {
  name = "ops-email"
  kind = "email"

  email_to = "ops@example.com"
}
