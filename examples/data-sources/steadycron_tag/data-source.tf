data "steadycron_tag" "env_prod" {
  id = "01234567-89ab-cdef-0123-456789abcdef"
}

output "tag_key" {
  value = data.steadycron_tag.env_prod.key
}
