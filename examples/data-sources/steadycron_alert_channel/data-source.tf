data "steadycron_alert_channel" "ops_email" {
  id = "01234567-89ab-cdef-0123-456789abcdef"
}

output "channel_name" {
  value = data.steadycron_alert_channel.ops_email.name
}

output "channel_kind" {
  value = data.steadycron_alert_channel.ops_email.kind
}
