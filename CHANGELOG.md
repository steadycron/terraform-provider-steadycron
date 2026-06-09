# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `steadycron_http_job` resource with full CRUD and `terraform import` support
- `steadycron_heartbeat_monitor` resource with full CRUD and `terraform import` support
- `steadycron_alert_channel` resource supporting email, Slack, Discord, webhook, and Telegram
- `steadycron_alert_rule` resource
- `steadycron_tag` resource
- `steadycron_template_variable` resource (name-only; values are write-only server-side)
- `data.steadycron_http_job` data source
- `data.steadycron_heartbeat_monitor` data source
- `data.steadycron_tag` data source
- `data.steadycron_alert_channel` data source
- Provider configuration: `endpoint` and `api_key` attributes; env var fallback
- 429/5xx retry with exponential backoff + jitter; `Retry-After` header respected
- `terraform import` for all resources
- GoReleaser cross-platform release pipeline
- Terraform Registry publishing workflow
- Generated provider documentation via `tfplugindocs`
