output "public_ip" {
  description = "Public IPv4 of the k3s VM."
  value       = google_compute_instance.lab.network_interface[0].access_config[0].nat_ip
}

output "ssh_user" {
  description = "SSH username on the VM."
  value       = "ubuntu"
}

output "kubeconfig_path_on_host" {
  description = "Path to the kubeconfig file inside the VM. The control plane fetches it via SSH after boot."
  value       = "/home/ubuntu/kubeconfig"
}

output "kubernetes_api_endpoint" {
  description = "URL of the Kubernetes API server."
  value       = "https://${google_compute_instance.lab.network_interface[0].access_config[0].nat_ip}:6443"
}
