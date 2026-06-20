# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.7] - 2026-06-20

### Added
- `runbook_notes` and `runbook_url` attributes on `steadycron_http_job` and
  `steadycron_heartbeat_monitor` resources and data sources — optional markdown
  remediation notes and an external runbook link, embedded inline in failure alert
  notifications (Slack, Telegram, email) when the job fails or a heartbeat is missed.

## [1.0.6] - 2026-06-16

### Added
- `misfire_policy` attribute on `steadycron_http_job` and `steadycron_heartbeat_monitor` resources and data sources — controls what happens when a scheduled fire is missed (`do_nothing` skips it; `fire_once_now` fires once immediately). Defaults to `do_nothing`.

### Fixed
- Alert rule updates no longer silently 404: the backend has no PATCH alert-rule endpoint, so all mutable attributes (`job_id`, `channel_id`, `trigger`, `severity`, `dedup_window_seconds`, `threshold`, `param_factor`, `param_min_baseline_samples`) now carry `RequiresReplace`. Any change destroys and recreates the rule instead of calling a missing endpoint.
- `ListJobs` now fetches all pages instead of silently capping at 100 jobs. This also fixes `terraform import` for alert rules on accounts with more than 100 jobs.

## [1.0.5] - 2026-06-16

### Added
- `key` attribute on `steadycron_http_job` and `steadycron_heartbeat_monitor` resources — optional, computed stable job key referenced by code-monitoring SDKs (e.g. `@steadycron.job("my-key")`). When omitted, the server auto-generates a slug from the job name.
- `key` attribute on `data.steadycron_http_job` and `data.steadycron_heartbeat_monitor` data sources (read-only).

## [1.0.4] - 2026-06-14

### Fixed
- `on_n_consecutive` alert rules no longer show "Provider produced inconsistent result after apply: .threshold was N, but now null" — the `ruleResponseToModel` else-branch incorrectly overwrote the already-set threshold when `params` is null (which is always the case for `on_n_consecutive`)

## [1.0.3] - 2026-06-14

### Fixed
- Tags were not applied to jobs after `terraform apply`: `SetJobTags` was sending `{"tagIds": [...]}` but the API's snake_case naming policy requires `{"tag_ids": [...]}`, so the body always deserialized as null and the tag set was silently cleared instead of being written
- `on_n_consecutive` alert rules now correctly send `threshold` as a top-level field in the request (previously it was incorrectly nested inside `params`, which the API rejects)
- Creating a heartbeat monitor via API key no longer auto-creates a default email alert rule (`on_missed_heartbeat` + `on_failure`); those UI-convenience defaults are now skipped for programmatic callers so the declared rule set is the only one that exists

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
