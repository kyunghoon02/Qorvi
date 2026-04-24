resource "google_project_service" "services" {
  for_each = toset(var.services)

  project                    = var.project
  service                    = each.value
  disable_dependent_services = false
  disable_on_destroy         = false
}
