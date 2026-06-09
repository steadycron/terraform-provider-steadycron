resource "steadycron_alert_channel" "ops_email" {
  name     = "ops-email"
  kind     = "email"
  email_to = "ops@example.com"
}

resource "steadycron_http_job" "api_health" {
  name   = "api-health-check"
  method = "GET"
  url    = "https://api.example.com/health"

  interval_seconds = 300
}

# Alert on any failure
resource "steadycron_alert_rule" "on_failure" {
  job_id     = steadycron_http_job.api_health.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_failure"
  severity   = "P1"
}

# Alert on recovery (after a failure)
resource "steadycron_alert_rule" "on_recovery" {
  job_id     = steadycron_http_job.api_health.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_recovery"
  severity   = "P2"
}

# Alert after 3 consecutive failures
resource "steadycron_alert_rule" "on_3_consecutive" {
  job_id     = steadycron_http_job.api_health.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_n_consecutive"
  threshold  = 3
  severity   = "P1"
}

# Slow-run anomaly detection
resource "steadycron_alert_rule" "slow_run" {
  job_id     = steadycron_http_job.api_health.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_slow_run"
  severity   = "P2"

  param_factor                = 2.0
  param_min_baseline_samples  = 10
}
