variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "GCP region. Singapore keeps latency low for Qorvi and avoids the US exchange API restrictions you hit in FlowLens."
  type        = string
  default     = "asia-southeast1"
}

variable "zone" {
  description = "GCP zone in the selected region."
  type        = string
  default     = "asia-southeast1-a"
}

variable "network_name" {
  description = "VPC network name."
  type        = string
  default     = "qorvi-network"
}

variable "subnet_name" {
  description = "Subnetwork name."
  type        = string
  default     = "qorvi-subnet"
}

variable "subnet_cidr" {
  description = "CIDR block for the Qorvi subnet."
  type        = string
  default     = "10.50.0.0/24"
}

variable "instance_name" {
  description = "Compute Engine VM name."
  type        = string
  default     = "qorvi-backend"
}

variable "machine_type" {
  description = "Recommended minimum machine type. e2-standard-2 is the smallest shape that is still comfortable for api + postgres + redis + neo4j on one VM."
  type        = string
  default     = "e2-standard-2"
}

variable "boot_disk_size_gb" {
  description = "Boot disk size in GB."
  type        = number
  default     = 50
}

variable "boot_disk_type" {
  description = "Boot disk type."
  type        = string
  default     = "pd-standard"
}

variable "image_family" {
  description = "Image family for the VM."
  type        = string
  default     = "ubuntu-2204-lts"
}

variable "image_project" {
  description = "Image project for the VM image."
  type        = string
  default     = "ubuntu-os-cloud"
}

variable "ssh_source_ranges" {
  description = "CIDR blocks allowed to SSH into the VM."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "additional_web_ports" {
  description = "Additional TCP ports to open publicly. Leave empty to keep the API behind nginx only."
  type        = list(number)
  default     = []
}

variable "ssh_user" {
  description = "SSH username for instance metadata."
  type        = string
  default     = "qorvi"
}

variable "ssh_public_key" {
  description = "SSH public key contents for VM access."
  type        = string
  default     = ""
  sensitive   = true
}

variable "service_account_id" {
  description = "Optional existing service account email. Leave empty to use the default compute service account."
  type        = string
  default     = ""
}

variable "enable_os_login" {
  description = "Whether to enable OS Login on the VM."
  type        = bool
  default     = true
}

variable "startup_script_extra" {
  description = "Optional additional shell commands appended to the startup script."
  type        = string
  default     = ""
}

variable "app_repo_url" {
  description = "Git repository URL for the Qorvi application bootstrap."
  type        = string
  default     = ""
}

variable "app_repo_token" {
  description = "Optional GitHub token used only for cloning/fetching private Qorvi app repositories."
  type        = string
  default     = ""
  sensitive   = true
}

variable "app_repo_ref" {
  description = "Git branch, tag, or commit to deploy."
  type        = string
  default     = "main"
}

variable "app_directory" {
  description = "Absolute path where Qorvi should be checked out."
  type        = string
  default     = "/opt/qorvi/app"
}

variable "app_service_name" {
  description = "systemd service name for the Qorvi backend bootstrap."
  type        = string
  default     = "qorvi-backend"
}

variable "app_bind_host" {
  description = "Bind host for the Qorvi API service."
  type        = string
  default     = "0.0.0.0"
}

variable "app_bind_port" {
  description = "Bind port for the Qorvi API service."
  type        = number
  default     = 4000
}

variable "app_user" {
  description = "Linux user that owns and runs the deployed app."
  type        = string
  default     = "qorvi"
}

variable "app_domain" {
  description = "Optional public domain for the API reverse proxy."
  type        = string
  default     = "api.qorvi.app"
}

variable "app_env_content" {
  description = "Full .env.backend contents written into the deployed app directory."
  type        = string
  default     = ""
  sensitive   = true
}

variable "labels" {
  description = "Additional labels for resources."
  type        = map(string)
  default     = {}
}
