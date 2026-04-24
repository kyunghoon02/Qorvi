output "web_firewall_name" {
  description = "Web firewall rule name."
  value       = google_compute_firewall.web.name
}

output "ssh_firewall_name" {
  description = "SSH firewall rule name."
  value       = google_compute_firewall.ssh.name
}
