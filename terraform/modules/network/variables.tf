variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "Subnetwork region."
  type        = string
}

variable "network_name" {
  description = "VPC network name."
  type        = string
}

variable "subnet_name" {
  description = "Subnetwork name."
  type        = string
}

variable "subnet_cidr" {
  description = "Subnetwork CIDR."
  type        = string
}
