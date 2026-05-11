# ilabhu

[![CI](https://github.com/eltonlaice/ilabhu/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/eltonlaice/ilabhu/actions/workflows/go.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/eltonlaice/ilabhu/control-plane.svg)](https://pkg.go.dev/github.com/eltonlaice/ilabhu/control-plane)
[![Go Report Card](https://goreportcard.com/badge/github.com/eltonlaice/ilabhu/control-plane)](https://goreportcard.com/report/github.com/eltonlaice/ilabhu/control-plane)

**Practice the full certification exam, on infrastructure you control.**

Open source. Exam-focused (CKA, CKAD, CKS, RHCSA, ...). Provisioned in your own AWS, GCP, Azure or DigitalOcean account — or on Linux servers you already own. No shared sandbox, no subscription.

🔗 **Landing page:** [eltonlaice.github.io/ilabhu](https://eltonlaice.github.io/ilabhu/)

---

## Why

Preparing for certifications like the CKA or RHCSA needs realistic environments where you can drive the whole exam end-to-end, not micro-exercises that only test one objective. Existing platforms either:

- charge a recurring subscription (KodeKloud, A Cloud Guru),
- run on shared infrastructure you can't inspect or extend (Killercoda, iximiuz Labs),
- or are tied to a specific vendor's cloud.

`ilabhu` is different:

- **Open source** (Apache 2.0). Fork it, self-host it, contribute exams.
- **Bring your own cloud or your own hosts.** Connect AWS / GCP / Azure / DigitalOcean via short-lived assumed credentials, or hand ilabhu a list of Linux servers you already operate. No long-lived keys stored. Resources you own, billing you control, lifecycle you decide.
- **Exam-shaped content.** Each exam mirrors the real certification — multiple weighted domains, ~15-20 tasks, time-bounded, scored. Not a stack of disconnected micro-labs.
- **Self-hostable.** A single `docker compose up` for individuals; a Kubernetes deploy for teams and training providers.

## Status

Pre-alpha. Pipeline is wired (catalog, provisioning, validation) but only the AWS adapter and a single warm-up exam are shipped. See the roadmap below.

## How it works

```
┌────────────┐    ┌──────────────────┐    ┌──────────────────────┐
│  Web UI    │───▶│  Control plane   │───▶│  Provider adapter    │
│ (Next.js)  │    │      (Go)        │    │  AWS · GCP · Azure   │
└────────────┘    └──────────────────┘    │  DO · BYO-hosts      │
                          │                └──────────┬───────────┘
                          │   AssumeRole / WIF /      │
                          │   SP / token / SSH key    │
                          ▼                           ▼
                  ┌──────────────────┐    ┌────────────────────────┐
                  │  Validator       │◀───│  Exam environment      │
                  │  (kubectl/SSH)   │    │  (your infra)          │
                  └──────────────────┘    └────────────────────────┘
```

An exam is a versioned directory under `exams/` containing:

- `exam.yaml` — metadata, domains, tasks, validations, declared providers
- `<provider>/` — one Terraform module (cloud) or one setup script (BYO-hosts) per declared provider

When a user starts a session, they pick a provider; the control plane uses the matching adapter to provision the environment in the user's account or hosts, opens a browser terminal, and runs the validation rules on demand.

For the IAM role + external-id setup, see [docs/byo-cloud-setup.md](docs/byo-cloud-setup.md). For the full architecture, see [docs/architecture.md](docs/architecture.md). For the manifest schema, see [docs/exam-schema.md](docs/exam-schema.md). To add your own exam, see [docs/authoring-an-exam.md](docs/authoring-an-exam.md). When something breaks, see [docs/troubleshooting.md](docs/troubleshooting.md).

## Roadmap

- [x] Exam manifest schema (`exam.yaml`) v1, multi-provider
- [x] First exam: CKA — warmup (single task, AWS)
- [x] Control plane: AWS assume-role + terraform apply/destroy
- [x] Web UI: exam catalog, exam detail, session monitor with kubeconfig download, per-task Validate
- [x] Validator: kubectl checks
- [ ] BYO-hosts adapter (SSH-driven setup/teardown)
- [ ] DigitalOcean adapter
- [ ] GCP and Azure adapters
- [ ] Multi-node Terraform modules
- [ ] Full CKA content (~17 tasks across all domains)
- [ ] CKAD, CKS, RHCSA exam packs
- [ ] Time-limit timer + weighted scoring UI
- [ ] `docker compose` self-host
- [ ] Authoring docs

## License

Apache License 2.0. See [LICENSE](./LICENSE).
