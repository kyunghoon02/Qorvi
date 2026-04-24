resource "google_compute_address" "this" {
  project = var.project_id
  name    = var.address_name
  region  = var.region
}
