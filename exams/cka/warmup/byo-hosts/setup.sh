#!/bin/bash
# ilabhu CKA warmup — byo-hosts setup script.
#
# Runs as the SSH user (with sudo -E) that ilabhud connects with. Installs
# k3s and writes the kubeconfig at $HOME/kubeconfig with the public IP
# substituted in. Idempotent: re-running on an already-set-up host is fine.
set -euxo pipefail

if ! command -v k3s >/dev/null 2>&1; then
  curl -sfL https://get.k3s.io | sh -
fi

until [ -f /etc/rancher/k3s/k3s.yaml ]; do sleep 1; done

# Prefer the address ilabhud reached us on (set by the smoke harness or
# manually for testing); fall back to ifconfig.me; final fallback to the
# first non-loopback IPv4.
PUBLIC_IP="${ILABHU_PUBLIC_IP:-}"
if [ -z "$PUBLIC_IP" ]; then
  PUBLIC_IP=$(curl -fsS --max-time 5 https://ifconfig.me 2>/dev/null || true)
fi
if [ -z "$PUBLIC_IP" ]; then
  PUBLIC_IP=$(hostname -I | awk '{print $1}')
fi

# Home of the SSH user (the caller). $SUDO_USER is set when invoked via sudo.
TARGET_USER="${SUDO_USER:-$USER}"
TARGET_HOME=$(getent passwd "$TARGET_USER" | cut -d: -f6)

cp /etc/rancher/k3s/k3s.yaml "$TARGET_HOME/kubeconfig"
sed -i "s|127.0.0.1|$PUBLIC_IP|" "$TARGET_HOME/kubeconfig"
chown "$TARGET_USER:$TARGET_USER" "$TARGET_HOME/kubeconfig"
chmod 600 "$TARGET_HOME/kubeconfig"

echo "ilabhu warmup ready on $PUBLIC_IP for user $TARGET_USER"
