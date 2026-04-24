output "enabled_services" {
  description = "Enabled project services."
  value       = keys(google_project_service.services)
}
