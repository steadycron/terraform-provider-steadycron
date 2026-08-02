# Terraform Provider for [SteadyCron](https://steadycron.com)

[![CI](https://github.com/steadycron/terraform-provider-steadycron/actions/workflows/ci.yml/badge.svg)](https://github.com/steadycron/terraform-provider-steadycron/actions/workflows/ci.yml)
[![Registry](https://img.shields.io/badge/Terraform_Registry-steadycron%2Fsteadycron-blue)](https://registry.terraform.io/providers/steadycron/steadycron/latest)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-blue.svg)](LICENSE)

The official Terraform provider for [SteadyCron](https://steadycron.com) — manage
cron jobs, heartbeat monitors, alert channels, and alert rules as infrastructure as code
alongside the rest of your Terraform stack. Declare schedules, alert routing, and
monitoring configuration in HCL and keep them in sync with `plan` / `apply` / `import`.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (to build from source)
- A SteadyCron account and a **Full**-scope API key for write operations

## Install

Add the provider to your `required_providers` block:

```hcl
terraform {
  required_providers {
    steadycron = {
      source  = "steadycron/steadycron"
      version = "1.1.0"
    }
  }
}
```

Then run `terraform init`.

## Authentication

Create an API key in the SteadyCron dashboard under **Settings → API keys**.

- **Full**-scope key: required for `apply` (create/update/delete).
- **Read-only**-scope key: sufficient for data sources only.

Provide the key via an environment variable (recommended for CI):

```bash
export STEADYCRON_API_KEY=sc_...
```

Or in the provider block (avoid committing this to version control):

```hcl
provider "steadycron" {
  api_key = "sc_..."   # or use var.steadycron_api_key
}
```

## Provider configuration

```hcl
provider "steadycron" {
  # optional; defaults to https://api.steadycron.com
  # can also be set via STEADYCRON_ENDPOINT
  endpoint = "https://api.steadycron.com"

  # required; Full-scope key for writes
  # can also be set via STEADYCRON_API_KEY
  api_key = var.steadycron_api_key
}
```

## Rate limits

The SteadyCron API allows **120 requests per minute per key**. The provider automatically retries
`429 Too Many Requests` responses with exponential backoff + jitter (up to 5 retries, respecting
`Retry-After` headers). If you hit limits regularly, reduce provider parallelism with
`terraform apply -parallelism=5`.

## Resources and data sources

| Resource / Data Source | Description |
|---|---|
| `steadycron_http_job` | Scheduled HTTPS call |
| `steadycron_heartbeat_monitor` | Expected-ping monitor with a unique ping URL |
| `steadycron_agent_monitor` | AI agent monitor — judges each run's reported output, cost, and step count |
| `steadycron_alert_channel` | Delivery channel (email, Slack, Discord, webhook, Telegram) |
| `steadycron_alert_rule` | Links a job to a channel with a trigger condition |
| `steadycron_tag` | `key=value` label for grouping/filtering jobs |
| `steadycron_template_variable` | Named placeholder for server-side substitution in job fields |
| `data.steadycron_http_job` | Look up an HTTP job by ID |
| `data.steadycron_heartbeat_monitor` | Look up a heartbeat monitor by ID |
| `data.steadycron_agent_monitor` | Look up an agent monitor by ID |
| `data.steadycron_tag` | Look up a tag by ID |
| `data.steadycron_alert_channel` | Look up an alert channel by ID |

## Example

```hcl
terraform {
  required_providers {
    steadycron = {
      source  = "steadycron/steadycron"
      version = "1.1.0"
    }
  }
}

provider "steadycron" {
  # api_key from STEADYCRON_API_KEY env var
}

resource "steadycron_tag" "env_prod" {
  key   = "env"
  value = "prod"
  color = "green"
}

resource "steadycron_alert_channel" "ops_email" {
  name     = "ops-email"
  kind     = "email"
  email_to = "ops@example.com"
}

resource "steadycron_http_job" "weekly_digest" {
  name   = "weekly-digest"
  method = "POST"
  url    = "https://api.example.com/jobs/digest"

  cron_expression = "0 9 * * 1"   # Monday 09:00
  timezone        = "Europe/Berlin"

  timeout_seconds = 120
  max_retries     = 2

  headers = {
    Authorization = "Bearer {{digest_token}}"
  }

  tags = [steadycron_tag.env_prod.id]
}

resource "steadycron_alert_rule" "digest_failure" {
  job_id     = steadycron_http_job.weekly_digest.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_failure"
  severity   = "P1"
}
```

## Monitoring an AI agent

`steadycron_agent_monitor` is a heartbeat that also reads what the run produced. Each run posts a
small JSON report and SteadyCron judges it — so an agent that exits 0 having done nothing is a
failure, not a green check.

```hcl
resource "steadycron_agent_monitor" "nightly_triage" {
  name            = "nightly-ticket-triage"
  cron_expression = "0 3 * * *"
  timezone        = "Europe/Berlin"

  # Two clocks: when the run must start, and how long it may then take.
  grace_seconds            = 600
  max_run_duration_seconds = 1800

  items_label = "tickets"   # names itemsProduced everywhere; 0 produced = failure

  rule_max_cost_usd_per_run    = 0.50
  rule_max_cost_usd_per_period = 20
  rule_cost_period             = "month"
  rule_max_steps               = 40    # loop detection
}

resource "steadycron_alert_rule" "triage_empty" {
  job_id     = steadycron_agent_monitor.nightly_triage.id
  channel_id = steadycron_alert_channel.ops_email.id
  trigger    = "on_empty_result"
  severity   = "P1"
}
```

A run is an **ordered pair** of calls — `/start` is mandatory, since without it there is nothing
to time a hung run against:

```bash
curl "$START_PING_URL"                       # start_ping_url

# … the agent does its work …

curl -X POST "$PING_URL" \
  -H 'Content-Type: application/json' \
  -d '{"itemsProduced": 42, "model": "claude-opus-5", "costUsd": 0.84}'
```

Every report field is optional; `itemsProduced` is the one the empty-result rule reads. Agent
monitors add four alert triggers — `on_empty_result`, `on_cost_exceeded`, `on_no_progress`, and
`on_unverified_run` — and a missed run still fires `on_missed_heartbeat`.

Two attributes force replacement rather than an in-place update, because the API accepts them on
create only: `report_required` and `rule_empty_result_enabled`. `stuck_run_detection` is not
exposed at all — the server forces it on for this kind.

## Cron-as-Code interoperability

Resources created via Terraform have a null `manifest_namespace`, so they are **never pruned** by
`steadycron sync --prune`. Manage a resource via Terraform **or** via the CLI/manifest, not both —
mixing tools for the same resource is unsupported.

If your team already uses Terraform for infrastructure, the provider is the natural way to bring
cron jobs under the same code review, state management, and deployment pipeline as the rest of
your stack — no separate manifest files or CLI tooling required.

## Code-monitoring SDK integration

All three job resources expose a `key` attribute (the stable `job_key` identifier). Set it to the
stable string your application code references:

```hcl
resource "steadycron_heartbeat_monitor" "db_backup" {
  name = "Nightly DB backup"
  key  = "nightly-db-backup"   # ← paste this into @steadycron.job("…") or TrackAsync("…")
  cron_expression = "0 2 * * *"
  grace_seconds   = 1800
}

output "db_backup_key" {
  value = steadycron_heartbeat_monitor.db_backup.key
}
```

**Rules:**
- `key` must be unique within the account. A duplicate produces a clear plan/apply error naming the conflicting key.
- When omitted, the server generates a slug from `name` (visible after `terraform apply`).
- Renaming `key` is an in-place update — no replacement occurs, but any in-code references must be updated.
- The SDK resolves the ping token from the key at runtime using `STEADYCRON_API_KEY` (read-only scope is sufficient).

Declaring the `key` in Terraform means your cron schedule, alert rules, and SDK instrumentation
all live in the same configuration — the cron job itself and the code that monitors it are
managed as code together.

## Importing existing resources

```bash
terraform import steadycron_http_job.example <job_id>
terraform import steadycron_heartbeat_monitor.db_backup <job_id>
terraform import steadycron_agent_monitor.nightly_triage <job_id>
terraform import steadycron_alert_channel.ops_email <channel_id>
terraform import steadycron_alert_rule.digest_failure <rule_id>
terraform import steadycron_tag.env_prod <tag_id>
terraform import steadycron_template_variable.digest_token <variable_id>
```

After importing, run `terraform plan`. Fields that the API redacts on GET (alert channel secrets,
heartbeat token) will show as diffs — add their values to your configuration.

## Development

```bash
git clone https://github.com/steadycron/terraform-provider-steadycron.git
cd terraform-provider-steadycron
make build          # build
make test           # unit tests
make testacc        # acceptance tests (requires STEADYCRON_API_KEY)
make docs           # regenerate docs from schema
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full development and release process.

## License

[Mozilla Public License 2.0](LICENSE) — same as the HashiCorp Terraform provider ecosystem.
