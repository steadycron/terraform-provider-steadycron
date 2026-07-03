# Investigation Findings — PR 1

This document answers the pre-coding investigation questions from the scaffold spec.

---

## Q1 — Exact routes and DTOs for CRUD operations

Routes inferred from the REST API design documented in `ARCHITECTURE.md` and `DATA_MODEL.md`.
All routes are under `https://api.steadycron.com`.

### Jobs (`/api/jobs`)

| Method | Route | Notes |
|---|---|---|
| `POST` | `/api/jobs` | Create HTTP or heartbeat job; `kind` in body determines type |
| `GET` | `/api/jobs` | List jobs (paginated) |
| `GET` | `/api/jobs/{id}` | Get single job; returns full state suitable for import |
| `PATCH` | `/api/jobs/{id}` | In-place update; does **not** recreate |
| `DELETE` | `/api/jobs/{id}` | Soft-delete (sets `deleted_at`) |

**Create/update body fields (both kinds):**
- `kind` (string, required): `"http"` or `"heartbeat"`
- `name` (string, required)
- `description` (string, optional)
- `schedule_kind` (string, required): `"cron"` or `"interval"`
- `cron_expression` (string): required when `schedule_kind = "cron"`
- `interval_seconds` (int): required when `schedule_kind = "interval"`
- `timezone` (string): IANA name, defaults to `"UTC"`
- `tags` (array of UUIDs)

**HTTP-kind extra fields:**
- `method` (string): `GET | POST | PUT | PATCH | DELETE`
- `url` (string)
- `headers` (object): `{"key": "value"}`
- `body` (string)
- `timeout_seconds` (int)
- `max_retries` (int)
- `retry_backoff_seconds` (int)
- `skip_if_running` (bool)

**Heartbeat-kind extra fields:**
- `grace_seconds` (int)
- `stuck_run_detection` (bool)
- `max_run_duration_seconds` (int)

**Response extra fields (computed):**
- `id`, `account_id`, `created_at`, `updated_at`
- `status` (string, derived from three-axis model — see `DATA_MODEL.md § 4.4`)
- `next_fire_at`, `last_fire_at`, `badge_url`
- Heartbeat: `ping_url`, `token`, `last_success_at`, `last_start_at`, `last_fail_at`

### Alert channels (`/api/alert-channels`)

| Method | Route |
|---|---|
| `POST` | `/api/alert-channels` |
| `GET` | `/api/alert-channels` |
| `GET` | `/api/alert-channels/{id}` |
| `PATCH` | `/api/alert-channels/{id}` |
| `DELETE` | `/api/alert-channels/{id}` |

**Body:** `{ "name": "...", "kind": "email|slack|discord|webhook|telegram", "config": {...} }`

### Alert rules (`/api/alert-rules`)

| Method | Route |
|---|---|
| `POST` | `/api/alert-rules` |
| `GET` | `/api/alert-rules/{id}` |
| `PATCH` | `/api/alert-rules/{id}` |
| `DELETE` | `/api/alert-rules/{id}` |

### Tags (`/api/tags`)

| Method | Route |
|---|---|
| `POST` | `/api/tags` |
| `GET` | `/api/tags` |
| `GET` | `/api/tags/{id}` |
| `PATCH` | `/api/tags/{id}` |
| `DELETE` | `/api/tags/{id}` |

### Template variables (`/api/template-variables`)

| Method | Route |
|---|---|
| `POST` | `/api/template-variables` |
| `GET` | `/api/template-variables` |
| `GET` | `/api/template-variables/{id}` |
| `PATCH` | `/api/template-variables/{id}` |
| `DELETE` | `/api/template-variables/{id}` |

**No missing single-resource verbs found** based on the documented REST patterns. All resources expose full CRUD including individual `GET /{id}` for import support. Confirmation against the live OpenAPI spec (`GET /openapi/v1.json`) is recommended before shipping PR 2–3.

---

## Q2 — Alert rule creation endpoint

Alert rules are created at `POST /api/alert-rules` with `job_id` in the request body:

```json
{
  "job_id": "uuid",
  "channel_id": "uuid",
  "trigger": "on_failure",
  "severity": "P2",
  "dedup_window_seconds": 300
}
```

Rules are **not** nested under `/api/jobs/{id}/...`. The logical identity is `(job_id, channel_id, trigger)` — the server enforces uniqueness on this triple (consistent with the reconcile endpoint's upsert key documented in `DATA_MODEL.md § 8`).

---

## Q3 — Does `GET /{id}` return enough to reconstruct Terraform state?

**Jobs:** Yes. `GET /api/jobs/{id}` returns all schedule, config, and computed fields. The only fields not round-trippable are:
- HTTP job `headers` values — may be returned redacted if they contain template variables or service credentials. **Workaround:** after import, run `terraform plan` and add any diffing header values to your `.tf` file.
- Heartbeat `token` / `ping_url` — returned on creation; may be redacted on subsequent GETs. **Post-import diff:** the provider will show these as unknown after import. Add the token from the dashboard to avoid perpetual diff.

**Alert channels:** Non-secret config fields (email `to`, webhook `url`, telegram `chat_id`) are fully returned. Secret fields are redacted (see Q4). **Post-import action required:** add secret values to the `.tf` configuration.

**Tags, template variables, alert rules:** Fully reconstructable.

---

## Q4 — Alert channel config schemas and secret fields

| kind | Field | Type | Secret? | Notes |
|---|---|---|---|---|
| `email` | `to` | string | No | Recipient email; always returned |
| `slack` | `webhook_url` | string | **Yes** | Never returned on GET; set Sensitive + preserve from state |
| `discord` | `webhook_url` | string | **Yes** | Never returned on GET; set Sensitive + preserve from state |
| `webhook` | `url` | string | No | Target URL; returned on GET |
| `webhook` | `secret` | string | **Yes** | HMAC signing secret; never returned on GET |
| `telegram` | `bot_token` | string | **Yes** | Never returned on GET; set Sensitive + preserve from state |
| `telegram` | `chat_id` | string | No | Chat ID or @username; returned on GET |

**Implementation:** Secret fields are marked `Sensitive: true` in the Terraform schema. On `Read`, if the API returns a null/empty value for a secret field, the provider preserves the existing state value to prevent perpetual diffs.

---

## Q5 — Template variable GET response

`GET /api/template-variables/{id}` returns **only the variable name** (and metadata). Values are write-only server-side and are never included in GET responses. This is consistent with:
- `CRON_AS_CODE.md`: "NAME ONLY — values are write-only server-side, never in a manifest"
- Decision 5 in the scaffold spec

The `steadycron_template_variable` resource therefore manages existence only. No perpetual diff occurs because there is nothing to diff beyond the name.

---

## Q6 — OpenAPI spec completeness; client generation decision

The OpenAPI document is available at `GET /openapi/v1.json` (anonymous). Based on the documentation:

**Decision: hand-roll the Go client.**

Rationale:
- The API surface for the Terraform provider is narrow (< 20 endpoints).
- Generator tools like `oapi-codegen` add dependencies and generated boilerplate that obscures the logic.
- Hand-rolling gives precise control over retry behaviour, error mapping, and nullable field handling.
- The client is thin (see `internal/client/`) and straightforward to update when the API evolves.

If the API surface expands significantly in future PRs, revisit generation at that point.

**Recommendation:** Validate the hand-rolled client against the live spec by running `curl https://api.steadycron.com/openapi/v1.json | jq '.paths | keys'` and checking each route exists before shipping.

---

## Q7 — Acceptance test provisioning

**Recommendation:** Use a real SteadyCron account with a Full-scope API key.

Setup:
1. Sign up for a free SteadyCron account at `app.steadycron.com`.
2. Create a Full-scope API key at **Settings → API keys**.
3. Export: `export STEADYCRON_API_KEY=sc_...`
4. Run: `TF_ACC=1 go test ./internal/provider/... -v`

The acceptance tests skip automatically when `STEADYCRON_API_KEY` is absent (see `testAccPreCheck`).

**For CI:** store the key as a GitHub Actions secret `STEADYCRON_API_KEY` and enable the `testacc` workflow step only on trusted branch pushes (not pull requests from forks, as this would expose the key).

**Cleanup:** Each acceptance test creates and destroys its own resources via `terraform destroy`. No manual cleanup is needed unless a test run is interrupted.

---

## Blocking cross-repo dependencies

None identified based on documentation review. All resources appear to have full single-resource CRUD endpoints.

**Action required before merging PR 2–3:** Confirm against the live OpenAPI spec that:
- `PATCH /api/jobs/{id}` exists and supports partial/full updates.
- `GET /api/jobs/{id}` returns `ping_url` and `token` for heartbeat jobs.
- `PATCH /api/alert-rules/{id}` exists (some APIs make rules immutable; create+delete pattern may be needed instead).
- `PATCH /api/template-variables/{id}` exists for rename support.

If any endpoint is missing, open a companion API PR in `steadycron/steadycron` and block the affected resource PR on it.
