data "steadycron_heartbeat_monitor" "backup" {
  id = "01234567-89ab-cdef-0123-456789abcdef"
}

output "ping_url" {
  value     = data.steadycron_heartbeat_monitor.backup.ping_url
  sensitive = true
}
