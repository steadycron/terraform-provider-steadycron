# Template variables are used as {{name}} placeholders in HTTP job fields.
# This resource manages the variable's existence (the name).
# Set the VALUE via the SteadyCron dashboard or:
#   steadycron vars set digest_token "your-secret-value"

resource "steadycron_template_variable" "digest_token" {
  name = "digest_token"
}

resource "steadycron_template_variable" "api_base_url" {
  name = "api_base_url"
}

# Reference the variable in an HTTP job
resource "steadycron_http_job" "digest" {
  name   = "weekly-digest"
  method = "POST"
  url    = "{{api_base_url}}/jobs/digest"  # substituted at execution time

  interval_seconds = 604800

  headers = {
    Authorization = "Bearer {{digest_token}}"
  }

  depends_on = [
    steadycron_template_variable.digest_token,
    steadycron_template_variable.api_base_url,
  ]
}
