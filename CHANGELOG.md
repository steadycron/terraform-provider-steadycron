# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.2] - 2026-06-14

### Added
- `steadycron_template_variable` now supports a `value` attribute (optional, computed, sensitive) — the server-side value is set at create/update time and stored encrypted in Terraform state

### Fixed
- Tags on `steadycron_http_job` and `steadycron_heartbeat_monitor` were silently ignored on create and update: the API requires tags to be set via a dedicated `PUT /jobs/{id}/tags` endpoint; both resources now call it after each create and update, eliminating the "Provider produced inconsistent result after apply" error for `.tags`
- `max_run_duration_seconds` values below the server minimum of 60 are now clamped before export, preventing a "Provider produced inconsistent result after apply" error when applying an account export that contained legacy sub-60-second values
- Updated required provider version in all exported `.tf` blocks and documentation from `~> 0.1` to `1.0.1`

## [1.0.1] - 2026-06-01

Re-release of 1.0.0 to fix Terraform Registry publishing (no code changes).

## [1.0.0] - 2026-06-01

### Added
- `steadycron_http_job` resource with full CRUD and `terraform import` support
- `steadycron_heartbeat_monitor` resource with full CRUD and `terraform import` support
- `steadycron_alert_channel` resource supporting email, Slack, Discord, webhook, and Telegram
- `steadycron_alert_rule` resource
- `steadycron_tag` resource
- `steadycron_template_variable` resource
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
