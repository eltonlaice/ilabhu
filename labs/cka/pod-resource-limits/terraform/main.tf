terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

locals {
  name = "ilabhu-${var.session_id}"
  tags = {
    "ilabhu.session_id" = var.session_id
    "ilabhu.lab"        = "cka/pod-resource-limits"
    "ilabhu.managed_by" = "ilabhu-control-plane"
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"]
  }
}

resource "aws_key_pair" "lab" {
  key_name   = local.name
  public_key = var.ssh_public_key
  tags       = local.tags
}

resource "aws_security_group" "lab" {
  name        = local.name
  description = "ilabhu lab session ${var.session_id}"
  tags        = local.tags

  ingress {
    description = "Kubernetes API"
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "lab" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_type
  key_name                    = aws_key_pair.lab.key_name
  vpc_security_group_ids      = [aws_security_group.lab.id]
  associate_public_ip_address = true
  tags                        = merge(local.tags, { Name = local.name })

  user_data = <<-EOT
    #!/bin/bash
    set -euxo pipefail
    curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode 644 --tls-san $(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)" sh -
    until [ -f /etc/rancher/k3s/k3s.yaml ]; do sleep 1; done
    cp /etc/rancher/k3s/k3s.yaml /home/ubuntu/kubeconfig
    sed -i "s|127.0.0.1|$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)|" /home/ubuntu/kubeconfig
    chown ubuntu:ubuntu /home/ubuntu/kubeconfig
  EOT

  root_block_device {
    volume_size = 20
    volume_type = "gp3"
  }
}
