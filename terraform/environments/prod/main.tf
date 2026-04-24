terraform {
  required_version = ">= 1.6.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}

locals {
  common_labels = merge(
    {
      app         = "qorvi"
      environment = "prod"
      managed_by  = "terraform"
    },
    var.labels
  )

  required_services = [
    "compute.googleapis.com",
    "iam.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
  ]
}

module "project_services" {
  source   = "../../modules/project_services"
  project  = var.project_id
  services = local.required_services
}

module "network" {
  source       = "../../modules/network"
  project_id   = var.project_id
  region       = var.region
  network_name = var.network_name
  subnet_name  = var.subnet_name
  subnet_cidr  = var.subnet_cidr
}

module "firewall" {
  source               = "../../modules/firewall"
  project_id           = var.project_id
  network_name         = module.network.network_name
  ssh_source_ranges    = var.ssh_source_ranges
  additional_web_ports = var.additional_web_ports
  target_tag           = "qorvi"
}

module "static_ip" {
  source       = "../../modules/static_ip"
  project_id   = var.project_id
  region       = var.region
  address_name = "${var.instance_name}-ip"
}

module "compute_vm" {
  source               = "../../modules/compute_vm"
  project_id           = var.project_id
  region               = var.region
  zone                 = var.zone
  instance_name        = var.instance_name
  machine_type         = var.machine_type
  network_name         = module.network.network_name
  subnetwork_name      = module.network.subnetwork_name
  static_ip_address    = module.static_ip.address
  service_account_id   = var.service_account_id
  boot_disk_size_gb    = var.boot_disk_size_gb
  boot_disk_type       = var.boot_disk_type
  image_family         = var.image_family
  image_project        = var.image_project
  enable_os_login      = var.enable_os_login
  ssh_user             = var.ssh_user
  ssh_public_key       = var.ssh_public_key
  labels               = local.common_labels
  tags                 = ["qorvi", "http-server", "https-server"]
  app_repo_url         = var.app_repo_url
  app_repo_token       = var.app_repo_token
  app_repo_ref         = var.app_repo_ref
  app_directory        = var.app_directory
  app_service_name     = var.app_service_name
  app_bind_host        = var.app_bind_host
  app_bind_port        = var.app_bind_port
  app_user             = var.app_user
  app_env_content      = var.app_env_content
  app_domain           = var.app_domain
  startup_script_extra = var.startup_script_extra

  depends_on = [
    module.project_services,
    module.network,
    module.static_ip,
  ]
}
