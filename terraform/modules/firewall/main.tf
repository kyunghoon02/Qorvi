resource "google_compute_firewall" "web" {
  project = var.project_id
  name    = "${var.network_name}-web"
  network = var.network_name

  allow {
    protocol = "tcp"
    ports    = concat(["80", "443"], [for port in var.additional_web_ports : tostring(port)])
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = [var.target_tag, "http-server", "https-server"]
}

resource "google_compute_firewall" "ssh" {
  project = var.project_id
  name    = "${var.network_name}-ssh"
  network = var.network_name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = var.ssh_source_ranges
  target_tags   = [var.target_tag]
}
