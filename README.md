# Terraform Provider for SteadyCron

[![CI](https://github.com/steadycron/terraform-provider-steadycron/actions/workflows/ci.yml/badge.svg)](https://github.com/steadycron/terraform-provider-steadycron/actions/workflows/ci.yml)
[![Registry](https://img.shields.io/badge/Terraform_Registry-steadycron%2Fsteadycron-blue)](https://registry.terraform.io/providers/steadycron/steadycron/latest)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-blue.svg)](LICENSE)

The official Terraform provider for [SteadyCron](https://steadycron.com) — manage HTTP jobs,
heartbeat monitors, alert channels, alert rules, tags, and template-variable names
declaratively with `plan` / `apply` / `import`.

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
      version = "1.0.4"
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
| `steadycron_alert_channel` | Delivery channel (email, Slack, Discord, webhook, Telegram) |
| `steadycron_alert_rule` | Links a job to a channel with a trigger condition |
| `steadycron_tag` | `key=value` label for grouping/filtering jobs |
| `steadycron_template_variable` | Named placeholder for server-side substitution in job fields |
| `data.steadycron_http_job` | Look up an HTTP job by ID |
| `data.steadycron_heartbeat_monitor` | Look up a heartbeat monitor by ID |
| `data.steadycron_tag` | Look up a tag by ID |
| `data.steadycron_alert_channel` | Look up an alert channel by ID |

## Example

```hcl
terraform {
  required_providers {
    steadycron = {
      source  = "steadycron/steadycron"
      version = "1.0.4"
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

## Cron-as-Code interoperability

Resources created via Terraform have a null `manifest_namespace`, so they are **never pruned** by
`steadycron sync --prune`. Manage a resource via Terraform **or** via the CLI/manifest, not both —
mixing tools for the same resource is unsupported.

## Code-monitoring SDK integration

Both job resources expose a `key` attribute (the stable `job_key` identifier). Set it to the
stable string your code references:

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

## Importing existing resources

```bash
terraform import steadycron_http_job.example <job_id>
terraform import steadycron_heartbeat_monitor.db_backup <job_id>
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
