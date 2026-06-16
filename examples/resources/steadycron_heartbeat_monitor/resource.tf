# Heartbeat monitor: SteadyCron watches for expected pings.
# Configure your cron job to call ping_url on every successful run.
resource "steadycron_heartbeat_monitor" "nightly_backup" {
  name        = "nightly-db-backup"
  description = "Postgres nightly backup — expected at 02:00 UTC"

  # key = stable identifier your code references — paste it into @steadycron.job("…") or TrackAsync("…").
  # When omitted the server auto-generates a slug from the name (visible after apply).
  key = "nightly-db-backup"

  cron_expression = "0 2 * * *" # 02:00 UTC daily
  timezone        = "UTC"
  grace_seconds   = 1800 # alert after 30 min overdue

  stuck_run_detection      = true
  max_run_duration_seconds = 3600 # alert if backup takes > 1 hour
}

# ── Shell / direct-ping usage ──────────────────────────────────────────────────
# Use the ping_url in your backup script:
#   curl "${steadycron_heartbeat_monitor.nightly_backup.ping_url}/start"
#   ... run backup ...
#   curl "${steadycron_heartbeat_monitor.nightly_backup.ping_url}/success"

output "ping_url" {
  value     = steadycron_heartbeat_monitor.nightly_backup.ping_url
  sensitive = true
}

# ── Code-monitoring SDK usage ──────────────────────────────────────────────────
# Paste the key into your application code. The SDK resolves the ping token at
# runtime via GET /api/monitors/resolve using your STEADYCRON_API_KEY (read-only).
#
# Python:
#   @steadycron.job("nightly-db-backup")
#   async def nightly_backup(): ...
#
# .NET:
#   await monitor.TrackAsync("nightly-db-backup", async ct => { ... }, ct);

output "job_key" {
  value       = steadycron_heartbeat_monitor.nightly_backup.key
  description = "Paste this value into @steadycron.job(\"…\") or TrackAsync(\"…\")."
}
