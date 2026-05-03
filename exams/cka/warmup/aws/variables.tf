variable "region" {
  description = "AWS region to deploy the lab into."
  type        = string
  default     = "eu-west-1"
}

variable "instance_type" {
  description = "EC2 instance type for the k3s node."
  type        = string
  default     = "t3.small"
}

variable "session_id" {
  description = "Unique id for this lab session. Used to tag and name resources."
  type        = string
}

variable "allowed_cidr" {
  description = "CIDR allowed to reach the Kubernetes API and SSH. Defaults to anywhere; the control plane should narrow this to its egress IP."
  type        = string
  default     = "0.0.0.0/0"
}

variable "ssh_public_key" {
  description = "SSH public key the control plane will use to fetch the kubeconfig and (optionally) expose a terminal to the user."
  type        = string
}
