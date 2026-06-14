terraform {
  required_providers {
    steadycron = {
      source  = "steadycron/steadycron"
      version = "1.0.3"
    }
  }
}

# API key from the STEADYCRON_API_KEY environment variable (recommended).
provider "steadycron" {}

# Or explicitly — use a variable, never hardcode.
# provider "steadycron" {
#   api_key  = var.steadycron_api_key
#   endpoint = "https://api.steadycron.com"  # optional
# }
