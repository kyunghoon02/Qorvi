variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "network_name" {
  description = "VPC network name."
  type        = string
}

variable "ssh_source_ranges" {
  description = "CIDRs allowed to SSH."
  type        = list(string)
}

variable "additional_web_ports" {
  description = "Additional public TCP ports to allow."
  type        = list(number)
  default     = []
}

variable "target_tag" {
  description = "Primary network tag attached to the application VM."
  type        = string
  default     = "qorvi"
}
