variable "project" {
  description = "GCP project id. Injected by the control plane via TF_VAR_project from the service account key's project_id field — do not set in inputs."
  type        = string
}

variable "region" {
  description = "GCP region."
  type        = string
  default     = "europe-west1"
}

variable "zone" {
  description = "GCP zone within the region."
  type        = string
  default     = "europe-west1-b"
}

variable "machine_type" {
  description = "Compute Engine machine type."
  type        = string
  default     = "e2-small"
}

variable "image" {
  description = "Boot disk image (family or self-link)."
  type        = string
  default     = "ubuntu-os-cloud/ubuntu-2404-lts-amd64"
}

variable "session_id" {
  description = "Unique id for this exam session. Used to tag and name resources."
  type        = string
}

variable "ssh_public_key" {
  description = "SSH public key the control plane uses to fetch the kubeconfig from the VM."
  type        = string
}
