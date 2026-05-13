# Cost comparison

The warmup exam runs a single-node k3s cluster for the duration of the session (default TTL: 2 hours). The same exam ships modules for AWS, DigitalOcean, GCP and Azure, plus a setup script for the BYO-hosts adapter. Their costs differ by an order of magnitude.

All numbers are list price for **a single 2-hour warmup session**, in USD, mid-2026, and exclude egress and managed-service fees that you would not normally hit running one VM for two hours.

## Per-session cost

| Provider | Default size | Region | Hourly | **2-hour session** | Notes |
|---|---|---|---:|---:|---|
| **BYO-hosts** | hardware you already own | — | — | **$0.00** | The differentiator. No cloud account required. |
| **DigitalOcean** | `s-1vcpu-2gb` | `ams3` | $0.018 | **$0.04** | Token auth, no IAM setup. Cheapest cloud. |
| **GCP** | `e2-small` | `europe-west1-b` | $0.018 | **$0.04** | Free-tier credit covers many sessions for new accounts. |
| **AWS** | `t3.small` | `eu-west-1` | $0.023 | **$0.05** | Standard cert-prep choice; you probably already have an account. |
| **Azure** | `Standard_B2s` | `westeurope` | $0.041 | **$0.08** | Most expensive of the four clouds; pick GCP / DO if cost is the only criterion. |

Add ~$0.005 for EBS / persistent disk on the cloud providers if the session runs the full 2 hours. Egress is negligible for a kubeconfig fetch + a few `kubectl` calls.

## Per-month, casual practice (10 sessions/week)

10 sessions × ~2h × ~4 weeks = ~80 hours of VM time per month.

| Provider | Monthly compute | Caveats |
|---|---:|---|
| BYO-hosts | **$0** (€4–€8 if you rent a small VPS for it) | The VPS sits idle most of the time. |
| DigitalOcean | **~$1.50** | Pure pay-as-you-go. |
| GCP | **~$1.50** | First three months on free credits effectively make this $0. |
| AWS | **~$1.85** | Same shape; t3.small fits comfortably in the free tier for the first 12 months. |
| Azure | **~$3.30** | Worst on this axis. |

## Per-month, intensive cram (3 sessions/day, 30 days)

90 sessions × 2h = ~180 hours.

| Provider | Monthly compute |
|---|---:|
| BYO-hosts | **$0** (the VPS does not care) |
| DigitalOcean | **~$3.25** |
| GCP | **~$3.25** |
| AWS | **~$4.15** |
| Azure | **~$7.40** |

## Which provider to pick

```
Do you already have a Linux server (homelab, VPS, work box)?
├── YES → BYO-hosts. $0 / session.
└── NO
    │
    Is "no cloud account" / "no credit card" a hard constraint?
    ├── YES → BYO-hosts is your only path. Grab a €4/month VPS (Hetzner,
    │         Contabo, Scaleway, OVH start-1-s). You break even on the
    │         cloud alternatives after ~80 sessions.
    └── NO
        │
        Are you already invested in a specific cloud?
        ├── AWS    → use the AWS adapter; you know the console.
        ├── GCP    → use GCP; free-tier credit absorbs the cost.
        ├── Azure  → use Azure; SP setup is fastest if you already use az ad.
        ├── None   → DigitalOcean. Token-based auth, no IAM gymnastics,
                     cheapest cloud, and the console is the friendliest.
```

## What the cost does NOT include

- **Time** you spend learning the exam objectives. ilabhu hands you the environment; it does not study for you.
- **Re-running failed sessions.** If `terraform apply` fails halfway through, the resources that did come up still bill until you `terraform destroy`. The control plane attempts this automatically on session failure, but a network glitch can leave residues. Set [AWS Budgets](https://aws.amazon.com/aws-cost-management/aws-budgets/), [GCP Billing alerts](https://cloud.google.com/billing/docs/how-to/budgets), [Azure cost alerts](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/cost-mgt-alerts-monitor-usage-spending), or the [DigitalOcean monthly budget](https://docs.digitalocean.com/products/billing/billing-alerts/) at $10 to be safe.
- **Multi-node setups.** The warmup is single-node. The full CKA exam (planned) will need multi-node — expect costs to roughly 3× on cloud and stay $0 on BYO-hosts with three boxes.
- **The control plane itself.** \`ilabhud\` is a Go binary you run wherever — a laptop, a $5 VPS, a Fly.io machine. Choose what you have.

## How the costs were measured

Numbers come from each provider's published pricing page as of May 2026:

- AWS EC2: https://aws.amazon.com/ec2/pricing/on-demand/
- DigitalOcean Droplets: https://www.digitalocean.com/pricing/droplets
- Google Compute Engine: https://cloud.google.com/compute/all-pricing
- Azure Linux VMs: https://azure.microsoft.com/en-us/pricing/details/virtual-machines/linux/

Your bill may differ — check before you cram.

See also: [BYO-cloud setup](byo-cloud-setup.md), [Architecture](architecture.md), [Authoring an exam](authoring-an-exam.md).
