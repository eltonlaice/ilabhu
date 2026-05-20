variable "region" {
  description = "DigitalOcean region slug (https://docs.digitalocean.com/products/platform/availability-matrix/)."
  type        = string
  default     = "ams3"
}

variable "droplet_size" {
  description = "Droplet size slug (https://slugs.do-api.dev/)."
  type        = string
  default     = "s-1vcpu-2gb"
}

variable "image" {
  description = "Droplet image slug."
  type        = string
  default     = "ubuntu-24-04-x64"
}

variable "session_id" {
  description = "Unique id for this exam session. Used to tag and name resources."
  type        = string
}

variable "ssh_public_key" {
  description = "SSH public key the control plane uses to fetch the kubeconfig from the droplet."
  type        = string
}
