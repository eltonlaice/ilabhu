output "public_ip" {
  description = "Public IP of the k3s node."
  value       = aws_instance.lab.public_ip
}

output "ssh_user" {
  description = "SSH username for the lab VM."
  value       = "ubuntu"
}

output "kubeconfig_path_on_host" {
  description = "Path to the kubeconfig file inside the lab VM. The control plane fetches it via SSH after boot."
  value       = "/home/ubuntu/kubeconfig"
}

output "kubernetes_api_endpoint" {
  description = "URL of the Kubernetes API server."
  value       = "https://${aws_instance.lab.public_ip}:6443"
}
