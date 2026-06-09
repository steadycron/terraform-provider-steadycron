data "steadycron_http_job" "example" {
  id = "01234567-89ab-cdef-0123-456789abcdef"
}

output "job_name" {
  value = data.steadycron_http_job.example.name
}

output "job_status" {
  value = data.steadycron_http_job.example.status
}
