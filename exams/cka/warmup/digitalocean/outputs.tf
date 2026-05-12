output "public_ip" {
  description = "Public IPv4 of the k3s droplet."
  value       = digitalocean_droplet.lab.ipv4_address
}

output "ssh_user" {
  description = "SSH username on the droplet."
  value       = "root"
}

output "kubeconfig_path_on_host" {
  description = "Path to the kubeconfig file inside the droplet. The control plane fetches it via SSH after boot."
  value       = "/root/kubeconfig"
}

output "kubernetes_api_endpoint" {
  description = "URL of the Kubernetes API server."
  value       = "https://${digitalocean_droplet.lab.ipv4_address}:6443"
}
