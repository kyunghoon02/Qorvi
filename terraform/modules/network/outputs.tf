output "network_name" {
  description = "Created VPC network name."
  value       = google_compute_network.this.name
}

output "network_id" {
  description = "Created VPC network ID."
  value       = google_compute_network.this.id
}

output "subnetwork_name" {
  description = "Created subnetwork name."
  value       = google_compute_subnetwork.this.name
}

output "subnetwork_id" {
  description = "Created subnetwork ID."
  value       = google_compute_subnetwork.this.id
}
