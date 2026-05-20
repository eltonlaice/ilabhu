terraform {
  required_version = ">= 1.5.0"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# The provider reads the DIGITALOCEAN_TOKEN env var the control plane injects.
provider "digitalocean" {}

locals {
  name = "ilabhu-${var.session_id}"
  tags = [
    "ilabhu",
    "ilabhu-session:${var.session_id}",
    "ilabhu-exam:cka-warmup",
  ]
}

resource "digitalocean_ssh_key" "lab" {
  name       = local.name
  public_key = var.ssh_public_key
}

resource "digitalocean_droplet" "lab" {
  image    = var.image
  name     = local.name
  region   = var.region
  size     = var.droplet_size
  ssh_keys = [digitalocean_ssh_key.lab.id]
  tags     = local.tags

  user_data = <<-EOT
    #!/bin/bash
    set -euxo pipefail
    PUBLIC_IP=$(curl -s http://169.254.169.254/metadata/v1/interfaces/public/0/ipv4/address)
    curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode 644 --tls-san $PUBLIC_IP" sh -
    until [ -f /etc/rancher/k3s/k3s.yaml ]; do sleep 1; done
    cp /etc/rancher/k3s/k3s.yaml /root/kubeconfig
    sed -i "s|127.0.0.1|$PUBLIC_IP|" /root/kubeconfig
  EOT
}
