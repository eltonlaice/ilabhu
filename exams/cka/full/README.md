# CKA — Full simulation

The full Certified Kubernetes Administrator simulation. 120-minute timer, 66% passing score, single-node k3s on the user's chosen infrastructure.

This is the **v1 content drop**. It covers four of five CKA domains. The fifth (Troubleshooting) and multi-node-specific scenarios (drain, taints/tolerations, affinity, DaemonSet) land in follow-up releases alongside the multi-node Terraform modules.

## Tasks shipped (v1)

| # | Domain | Task | Single-node OK |
|---|---|---|---|
| 1 | Workloads & Scheduling | Create and scale a Deployment | ✅ |
| 2 | Services & Networking | Expose a Deployment via NodePort + HTTP check | ✅ |
| 3 | Storage | Two containers sharing an `emptyDir` volume | ✅ |
| 4 | Cluster Architecture | RBAC: ServiceAccount + Role + RoleBinding + `can-i` check | ✅ |

Each task has between 3 and 7 deterministic validations (kubectl jsonpath + regex). Pass = every validation in every task passes.

## What's not in v1

- **Troubleshooting domain (5–6 tasks)** — needs a pre-staged broken cluster state. The `setup.manifests` schema feature lands in a follow-up so exam authors can declare `kubectl apply` files that the control plane runs at session start.
- **Multi-node scenarios (3–4 tasks)** — `kubectl drain`, taints/tolerations, node affinity, DaemonSet topology. These need the multi-node Terraform AWS module currently under design.
- **Real-cluster bootstrap quirks** — the warmup's `t3.small` is too tight for the 4 v1 tasks (you'll hit OOM with 5 replicas of nginx + an extra Deployment). \`cka/full\` bumps the default to `t3.medium` (~$0.04/hour) / `s-2vcpu-2gb` / `e2-medium` / `Standard_B2s`.

## Pre-conditions

The cluster starts clean. Every task creates the resources it needs. No pre-existing Deployments, Services, ConfigMaps, etc.

## Cost (approximate)

For a full 2-hour session:

| Provider | Default size | Hourly | **2 hours** |
|---|---|---|---:|
| AWS | `t3.medium` in `eu-west-1` | $0.046 | **$0.09** |
| DigitalOcean | `s-2vcpu-2gb` in `ams3` | $0.024 | **$0.05** |
| GCP | `e2-medium` in `europe-west1` | $0.036 | **$0.07** |
| Azure | `Standard_B2s` in `westeurope` | $0.041 | **$0.08** |
| BYO-hosts | a VM you already own | — | **$0** |

See [`docs/cost-comparison.md`](../../../docs/cost-comparison.md).
