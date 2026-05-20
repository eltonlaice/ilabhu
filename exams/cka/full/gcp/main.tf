terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

# The provider reads GOOGLE_CREDENTIALS (the SA key JSON) injected by the
# control plane.
provider "google" {
  project = var.project
  region  = var.region
  zone    = var.zone
}

locals {
  name = "ilabhu-${var.session_id}"
  labels = {
    "ilabhu"            = "true"
    "ilabhu-session"    = var.session_id
    "ilabhu-exam"       = "cka-warmup"
    "ilabhu-managed_by" = "ilabhu-control-plane"
  }
}

resource "google_compute_network" "lab" {
  name                    = local.name
  auto_create_subnetworks = true
}

resource "google_compute_firewall" "lab" {
  name    = "${local.name}-allow"
  network = google_compute_network.lab.name

  allow {
    protocol = "tcp"
    ports    = ["22", "6443"]
  }
  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["ilabhu-${var.session_id}"]
}

resource "google_compute_instance" "lab" {
  name         = local.name
  machine_type = var.machine_type
  zone         = var.zone
  tags         = ["ilabhu-${var.session_id}"]
  labels       = local.labels

  boot_disk {
    initialize_params {
      image = var.image
      size  = 20
    }
  }

  network_interface {
    network = google_compute_network.lab.name
    access_config {}
  }

  metadata = {
    "ssh-keys"       = "ubuntu:${var.ssh_public_key}"
    "startup-script" = <<-EOT
      #!/bin/bash
      set -euxo pipefail
      PUBLIC_IP=$(curl -s -H "Metadata-Flavor: Google" \
        http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip)
      curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode 644 --tls-san $PUBLIC_IP" sh -
      until [ -f /etc/rancher/k3s/k3s.yaml ]; do sleep 1; done
      cp /etc/rancher/k3s/k3s.yaml /home/ubuntu/kubeconfig
      sed -i "s|127.0.0.1|$PUBLIC_IP|" /home/ubuntu/kubeconfig
      chown ubuntu:ubuntu /home/ubuntu/kubeconfig
    EOT
  }
}
