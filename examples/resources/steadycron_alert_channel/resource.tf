# Email channel
resource "steadycron_alert_channel" "ops_email" {
  name     = "ops-email"
  kind     = "email"
  email_to = "ops@example.com"
}

# Slack channel — webhook_url is sensitive; use a variable or env var
resource "steadycron_alert_channel" "slack_oncall" {
  name             = "slack-oncall"
  kind             = "slack"
  slack_webhook_url = var.slack_webhook_url
}

# Discord
resource "steadycron_alert_channel" "discord_alerts" {
  name                = "discord-alerts"
  kind                = "discord"
  discord_webhook_url = var.discord_webhook_url
}

# Generic webhook with optional HMAC signing secret
resource "steadycron_alert_channel" "pagerduty" {
  name           = "pagerduty-webhook"
  kind           = "webhook"
  webhook_url    = "https://events.pagerduty.com/integration/abcdef/enqueue"
  webhook_secret = var.pagerduty_secret
}

# Telegram
resource "steadycron_alert_channel" "telegram_ops" {
  name               = "telegram-ops"
  kind               = "telegram"
  telegram_bot_token = var.telegram_bot_token
  telegram_chat_id   = "-1001234567890"
}

variable "slack_webhook_url" {
  type      = string
  sensitive = true
}

variable "discord_webhook_url" {
  type      = string
  sensitive = true
}

variable "pagerduty_secret" {
  type      = string
  sensitive = true
  default   = ""
}

variable "telegram_bot_token" {
  type      = string
  sensitive = true
}
