output "instance_name" {
  description = "Qorvi VM instance name."
  value       = module.compute_vm.instance_name
}

output "instance_external_ip" {
  description = "Public IP for the Qorvi VM."
  value       = module.static_ip.address
}

output "app_urls" {
  description = "Convenience URLs after deployment."
  value = {
    api_root   = "http://${module.static_ip.address}"
    healthz    = "http://${module.static_ip.address}/healthz"
    direct_api = "http://${module.static_ip.address}:${var.app_bind_port}"
    domain     = var.app_domain != "" ? "https://${var.app_domain}" : ""
  }
}
