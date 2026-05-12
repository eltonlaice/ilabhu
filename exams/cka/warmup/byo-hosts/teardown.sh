#!/bin/bash
# ilabhu CKA warmup — byo-hosts teardown script.
#
# Best-effort cleanup. ilabhud invokes this when the user destroys the
# session; failures from any individual step are logged but do not abort
# the rest of the teardown.
set -uxo pipefail

if [ -x /usr/local/bin/k3s-uninstall.sh ]; then
  /usr/local/bin/k3s-uninstall.sh || true
fi

TARGET_USER="${SUDO_USER:-$USER}"
TARGET_HOME=$(getent passwd "$TARGET_USER" | cut -d: -f6)
rm -f "$TARGET_HOME/kubeconfig" || true

echo "ilabhu warmup teardown complete"
