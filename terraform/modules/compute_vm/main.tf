locals {
  instance_metadata = merge(
    {
      enable-oslogin = var.enable_os_login ? "TRUE" : "FALSE"
    },
    var.ssh_public_key != "" ? {
      ssh-keys = "${var.ssh_user}:${var.ssh_public_key}"
    } : {}
  )
}

resource "google_compute_instance" "this" {
  project      = var.project_id
  zone         = var.zone
  name         = var.instance_name
  machine_type = var.machine_type
  tags         = var.tags
  labels       = var.labels

  boot_disk {
    auto_delete = true

    initialize_params {
      image = "projects/${var.image_project}/global/images/family/${var.image_family}"
      size  = var.boot_disk_size_gb
      type  = var.boot_disk_type
    }
  }

  network_interface {
    network    = var.network_name
    subnetwork = var.subnetwork_name

    access_config {
      nat_ip = var.static_ip_address
    }
  }

  metadata = local.instance_metadata

  metadata_startup_script = templatefile("${path.module}/startup.sh.tftpl", {
    app_bind_host        = var.app_bind_host
    app_bind_port        = var.app_bind_port
    app_directory        = var.app_directory
    app_domain           = var.app_domain
    app_env_content      = var.app_env_content
    app_repo_ref         = var.app_repo_ref
    app_repo_url         = var.app_repo_url
    app_repo_token       = var.app_repo_token
    app_service_name     = var.app_service_name
    app_user             = var.app_user
    startup_script_extra = var.startup_script_extra
  })

  service_account {
    email = var.service_account_id != "" ? var.service_account_id : null
    scopes = [
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
    ]
  }

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
    preemptible         = false
  }

  shielded_instance_config {
    enable_secure_boot          = true
    enable_vtpm                 = true
    enable_integrity_monitoring = true
  }

  lifecycle {
    ignore_changes = [
      metadata["ssh-keys"],
    ]
  }
}
