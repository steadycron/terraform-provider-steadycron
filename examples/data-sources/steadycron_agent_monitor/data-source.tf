data "steadycron_agent_monitor" "triage" {
  id = "01234567-89ab-cdef-0123-456789abcdef"
}

# A run is an ordered pair: /start when it begins, then the JSON report on completion.
output "start_ping_url" {
  value     = data.steadycron_agent_monitor.triage.start_ping_url
  sensitive = true
}

output "ping_url" {
  value     = data.steadycron_agent_monitor.triage.ping_url
  sensitive = true
}

# What this monitor treats as a unit of work — the noun itemsProduced is counted in.
output "items_label" {
  value = data.steadycron_agent_monitor.triage.items_label
}

output "monthly_budget_usd" {
  value = data.steadycron_agent_monitor.triage.rule_max_cost_usd_per_period
}
