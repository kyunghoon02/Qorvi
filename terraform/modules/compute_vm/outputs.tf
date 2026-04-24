output "instance_name" {
  description = "VM instance name."
  value       = google_compute_instance.this.name
}

output "self_link" {
  description = "VM self link."
  value       = google_compute_instance.this.self_link
}
