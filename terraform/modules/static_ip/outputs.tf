output "address" {
  description = "Reserved external IP address."
  value       = google_compute_address.this.address
}

output "name" {
  description = "Static IP resource name."
  value       = google_compute_address.this.name
}
