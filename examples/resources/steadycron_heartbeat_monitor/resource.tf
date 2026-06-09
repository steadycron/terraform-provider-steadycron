# Heartbeat monitor: SteadyCron watches for expected pings.
# Configure your cron job to call ping_url on every successful run.
resource "steadycron_heartbeat_monitor" "nightly_backup" {
  name        = "nightly-db-backup"
  description = "Postgres nightly backup — expected at 02:00 UTC"

  cron_expression = "0 2 * * *" # 02:00 UTC daily
  timezone        = "UTC"
  grace_seconds   = 1800 # alert after 30 min overdue

  stuck_run_detection    = true
  max_run_duration_seconds = 3600 # alert if backup takes > 1 hour
}

# Use the ping_url in your backup script:
#   curl "${steadycron_heartbeat_monitor.nightly_backup.ping_url}/start"
#   ... run backup ...
#   curl "${steadycron_heartbeat_monitor.nightly_backup.ping_url}/success"

output "ping_url" {
  value     = steadycron_heartbeat_monitor.nightly_backup.ping_url
  sensitive = true
}
