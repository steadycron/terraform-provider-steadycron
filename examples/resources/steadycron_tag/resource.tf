resource "steadycron_tag" "env_prod" {
  key   = "env"
  value = "prod"
  color = "green"
}

resource "steadycron_tag" "env_staging" {
  key   = "env"
  value = "staging"
  color = "yellow"
}

resource "steadycron_tag" "team_platform" {
  key   = "team"
  value = "platform"
}

# Attach tags to a job via the job's `tags` attribute
resource "steadycron_http_job" "example" {
  name   = "example-job"
  method = "GET"
  url    = "https://example.com"

  interval_seconds = 3600

  tags = [
    steadycron_tag.env_prod.id,
    steadycron_tag.team_platform.id,
  ]
}
