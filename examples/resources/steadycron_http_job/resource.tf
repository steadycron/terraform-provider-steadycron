resource "steadycron_tag" "env_prod" {
  key   = "env"
  value = "prod"
  color = "green"
}

# HTTP job with a cron schedule
resource "steadycron_http_job" "weekly_digest" {
  name        = "weekly-digest"
  description = "Send the weekly digest email every Monday"

  method = "POST"
  url    = "https://api.example.com/jobs/digest"

  cron_expression = "0 9 * * 1" # Monday 09:00
  timezone        = "Europe/Berlin"

  timeout_seconds      = 120
  max_retries          = 2
  retry_backoff_seconds = 60

  headers = {
    Authorization = "Bearer {{digest_token}}" # server-side template substitution
    Content-Type  = "application/json"
  }

  body = jsonencode({ segment = "weekly" })

  skip_if_running = true

  tags = [steadycron_tag.env_prod.id]
}

# HTTP job with an interval schedule
resource "steadycron_http_job" "warm_cache" {
  name = "warm-cdn-cache"

  method           = "GET"
  url              = "https://api.example.com/warm"
  interval_seconds = 900 # every 15 minutes
}

output "weekly_digest_id" {
  value = steadycron_http_job.weekly_digest.id
}

output "weekly_digest_status" {
  value = steadycron_http_job.weekly_digest.status
}
