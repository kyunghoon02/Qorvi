variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "GCP region."
  type        = string
}

variable "zone" {
  description = "GCP zone."
  type        = string
}

variable "instance_name" {
  description = "VM instance name."
  type        = string
}

variable "machine_type" {
  description = "VM machine type."
  type        = string
}

variable "network_name" {
  description = "VPC network name."
  type        = string
}

variable "subnetwork_name" {
  description = "Subnetwork name."
  type        = string
}

variable "static_ip_address" {
  description = "Reserved external IP address."
  type        = string
}

variable "service_account_id" {
  description = "Optional service account email."
  type        = string
  default     = ""
}

variable "boot_disk_size_gb" {
  description = "Boot disk size in GB."
  type        = number
}

variable "boot_disk_type" {
  description = "Boot disk type."
  type        = string
}

variable "image_family" {
  description = "Image family."
  type        = string
}

variable "image_project" {
  description = "Image project."
  type        = string
}

variable "enable_os_login" {
  description = "Enable OS Login metadata."
  type        = bool
}

variable "ssh_user" {
  description = "SSH username."
  type        = string
}

variable "ssh_public_key" {
  description = "Optional SSH public key."
  type        = string
  default     = ""
  sensitive   = true
}

variable "labels" {
  description = "Resource labels."
  type        = map(string)
  default     = {}
}

variable "tags" {
  description = "Network tags."
  type        = list(string)
  default     = []
}

variable "startup_script_extra" {
  description = "Extra shell commands appended to startup."
  type        = string
  default     = ""
}

variable "app_repo_url" {
  description = "Git repository URL for Qorvi app bootstrap. Leave empty to skip app bootstrap."
  type        = string
  default     = ""
}

variable "app_repo_token" {
  description = "Optional GitHub token used only for cloning/fetching private app repositories."
  type        = string
  default     = ""
  sensitive   = true
}

variable "app_repo_ref" {
  description = "Git branch, tag, or commit to checkout during bootstrap."
  type        = string
  default     = "main"
}

variable "app_directory" {
  description = "Absolute path where the app repository should live on the VM."
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
  description = "Linux user that should own and run the Qorvi app."
  type        = string
  default     = "qorvi"
}

variable "app_env_content" {
  description = "Full .env file contents written to the deployed app directory."
  type        = string
  default     = ""
  sensitive   = true
}

variable "app_domain" {
  description = "Optional public domain for the Qorvi API reverse proxy."
  type        = string
  default     = ""
}
