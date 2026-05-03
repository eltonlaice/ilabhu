# ilabhu

[![CI](https://github.com/eltonlaice/ilabhu/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/eltonlaice/ilabhu/actions/workflows/go.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/eltonlaice/ilabhu/control-plane.svg)](https://pkg.go.dev/github.com/eltonlaice/ilabhu/control-plane)
[![Go Report Card](https://goreportcard.com/badge/github.com/eltonlaice/ilabhu/control-plane)](https://goreportcard.com/report/github.com/eltonlaice/ilabhu/control-plane)

**Hands-on certification labs you run in your own cloud.**

Open source. Exam-focused (CKA, CKAD, CKS, RHCSA, ...). Provisioned in your own AWS, GCP or Azure account — no shared sandbox, no subscription.

---

## Why

Preparing for certifications like the CKA or RHCSA requires real environments, not slideware. Existing platforms either:

- charge a recurring subscription (KodeKloud, A Cloud Guru),
- run on shared infrastructure you can't inspect or extend (Killercoda, iximiuz Labs),
- or are tied to a specific vendor's cloud.

`ilabhu` is different:

- **Open source** (Apache 2.0). Fork it, self-host it, contribute labs.
- **Bring your own cloud.** Connect your AWS / GCP / Azure account via short-lived assumed roles. No long-lived keys stored. Labs run on your infra, billed to you, destroyed automatically when the TTL expires.
- **Exam-mapped content.** Each lab cites the exact exam objective it covers (e.g. *CKA 1.31 — Workloads & Scheduling — configure resource limits*).
- **Self-hostable.** A single `docker compose up` for individuals; a Kubernetes deploy for teams and training providers.

## Status

Pre-alpha. Not usable yet. Roadmap below.

## How it works

```
┌────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Web UI    │───▶│  Control plane   │───▶│  Terraform      │
│ (Next.js)  │    │      (Go)        │    │  (your cloud)   │
└────────────┘    └──────────────────┘    └────────┬────────┘
                          │                        │
                          │   AssumeRole / WIF     │
                          │   (no static keys)     │
                          ▼                        ▼
                  ┌──────────────────┐    ┌─────────────────┐
                  │  Validator       │◀───│  Lab VM(s)      │
                  │  (kubectl/SSH)   │    │  + ilabhu-agent │
                  └──────────────────┘    └─────────────────┘
```

A lab is a versioned directory under `labs/` containing:

- `lab.yaml` — metadata, instructions, tasks, validation rules
- `terraform/` — a Terraform module that provisions the lab's infrastructure

When a user starts a lab, the control plane assumes a role in the user's cloud account, runs `terraform apply` against the lab module, opens a browser terminal, and runs the validation rules on demand.

## Roadmap

- [ ] Lab manifest schema (`lab.yaml`) v1
- [ ] First lab: CKA — pod with resource limits (AWS, single-node k3s)
- [ ] Control plane: assume-role + terraform apply/destroy
- [ ] Web UI: lab catalog, terminal (xterm.js), task checklist
- [ ] Validator: kubectl + SSH check kinds
- [ ] `docker compose` self-host
- [ ] GCP and Azure adapters
- [ ] CKAD, CKS, RHCSA lab packs
- [ ] Lab authoring docs

## License

Apache License 2.0. See [LICENSE](./LICENSE).
