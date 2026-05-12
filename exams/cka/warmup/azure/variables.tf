variable "location" {
  description = "Azure region (https://learn.microsoft.com/en-us/azure/availability-zones/az-overview)."
  type        = string
  default     = "westeurope"
}

variable "vm_size" {
  description = "Azure VM size (https://learn.microsoft.com/en-us/azure/virtual-machines/sizes)."
  type        = string
  default     = "Standard_B2s"
}

variable "session_id" {
  description = "Unique id for this exam session. Used to name resources and as a resource-group / tag value."
  type        = string
}

variable "ssh_public_key" {
  description = "SSH public key the control plane uses to fetch the kubeconfig from the VM."
  type        = string
}
