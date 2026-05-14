# Changelog

All notable changes to ilabhu are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Pre-1.0 means breaking changes can land in any minor release; we'll call them out explicitly here.

## [Unreleased]

_Nothing yet._

## [0.1.0] — 2026-05-13

First tagged release. Everything below shipped between project bootstrap and this tag.

### Added — manifest schema (v1)

- Versioned `exam.yaml` manifests under `exams/<exam-code>/<slug>/` with weighted `domains`, `time_limit_minutes`, `passing_score`, and per-task `domain` + `weight`.
- One `Infrastructure.Providers` map per exam — the user picks the active provider at session-start time.
- JSON Schema at `docs/exam-schema.json` for editor autocomplete and inline validation. Reference is wired into `exam.yaml` via the `yaml-language-server: $schema=...` comment.

### Added — control plane (`ilabhud`)

- HTTP API on stdlib `net/http` with Go 1.22 mux patterns:
  - `GET /healthz`
  - `GET /v1/exams` and `GET /v1/exams/{id...}` (catch-all id supports `<exam>/<slug>`)
  - `POST /v1/sessions`, `GET /v1/sessions/{id}`, `DELETE /v1/sessions/{id}`
  - `POST /v1/sessions/{id}/tasks/{task_id}/validate`
- In-memory session store + lifecycle manager. `Manager.Start` dispatches to `provisionTerraform` or `provisionBYOHosts` based on the requested provider.
- Per-session workdir under `~/.ilabhu/sessions/<id>/` for terraform state and SSH keys; the source `exams/` tree is never written to.
- `provisioner/terraform.go` shells out to the user's `terraform` binary.
- `provisioner/workdir.go` copies the exam's provider module into the per-session workdir before apply.
- `cmd/ilabhud` entrypoint with `-addr`, `-exams-dir`, `-state-dir` flags.

### Added — provider adapters (5)

| Provider | Auth model |
|---|---|
| `aws` | `sts:AssumeRole` + external id |
| `digitalocean` | Personal Access Token |
| `gcp` | Service Account key JSON; `project_id` extracted server-side |
| `azure` | Service Principal with client secret (`ARM_*` env vars) |
| `byo-hosts` | One SSH private key + list of `{role, address, ssh_user}`; setup / teardown scripts run via SSH |

Each adapter has its own Go package under `internal/cloud/<provider>/` (cloud) or its own branch in `internal/session/byohosts.go` (BYO-hosts). Every cloud adapter ships a per-exam Terraform module under `exams/cka/warmup/<provider>/`. The BYO-hosts adapter validates declared roles against the supplied host list and sanitises host fields against shell-metacharacter injection (CodeQL-recognised regex sanitiser).

### Added — validator

Three validation kinds, each evaluated per task:

- **`kubectl`** — runs `kubectl` against the session kubeconfig. Supports `expect_equals`, `expect_contains`, `expect_regex`, `expect_exit_code`.
- **`shell`** — pipes a script over SSH to `/bin/sh` on the access host. Supports `expect_exit_code` (default 0) and the same string assertions as `kubectl`.
- **`http`** — `GET` against `v.URL`. Supports `{{public_ip}}` substitution, `expect_status` (default 200), `expect_body_contains`, `expect_regex`.

`expect_regex` uses Go's RE2 syntax. Invalid patterns fail the validation at run time with a clear message.

### Added — exam content

- `exams/cka/warmup/` — first warm-up exam (single task: a Pod with CPU and memory limits). Terraform modules for AWS, DigitalOcean, GCP and Azure plus an SSH `setup.sh` / `teardown.sh` pair for BYO-hosts.

### Added — web frontend

- Next.js 16 (App Router, TypeScript, Tailwind 4) at `web/`.
- Catalog page (`/`) — server-renders `GET /v1/exams` as a card grid.
- Exam detail page (`/exams/[...slug]`) — full manifest with domain weights and task instructions.
- Start-session form — provider selector with five chips; the active chip swaps in the correct credentials block.
- Session monitor (`/sessions/[id]`) — polls status every 5 s while non-terminal, exposes a kubeconfig download (base64-decoded client-side), a Destroy button, and per-task Validate buttons that surface a result list.
- Branded 404 page.
- API client in `src/lib/api.ts` with `apiURL` resolving to either the same-origin `/api/*` (browser, via Next rewrite) or the absolute `ILABHU_API_BASE` (server-side fetch).

### Added — landing site and SEO

- `site/index.html` at <https://ilabhu.com/> — branded landing page deployed via GitHub Pages.
- Structured data (`application/ld+json`) carrying five linked schemas: `WebSite`, `Organization`, `SoftwareApplication`, `SoftwareSourceCode`, `FAQPage`.
- FAQ section paired with the `FAQPage` schema (five Q&As).
- `site/robots.txt`, `site/sitemap.xml`, `site/og-image.svg`, `site/favicon.svg`.
- `site/llms.txt` — markdown summary for LLM crawlers (Anthropic, Perplexity, etc.).
- `site/.well-known/security.txt` (RFC 9116) pointing at the GitHub Security Advisories private flow.
- `site/humans.txt`.
- `site/.linter.py` — static validator run by the `Site lint` CI workflow on every PR that touches `site/`.

### Added — docs

- `README.md` with badges, positioning, architecture, roadmap.
- `docs/architecture.md` — components, lifecycle, design decisions.
- `docs/exam-schema.md` + `docs/exam-schema.json` — manifest reference.
- `docs/authoring-an-exam.md` — step-by-step contributor walkthrough.
- `docs/adding-a-provider.md` — contributor walkthrough for a sixth provider adapter.
- `docs/byo-cloud-setup.md` — AWS, DigitalOcean, GCP, Azure and BYO-hosts setup sections.
- `docs/cost-comparison.md` — per-session cost table and decision tree.
- `docs/troubleshooting.md` — common failure modes across control plane / cloud provisioning / web / CI.
- `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE` (Apache 2.0), issue templates, PR template.

### Added — deploy

- `deploy/docker-compose.yml` brings up the control plane + web on `localhost:3000` with a single `docker compose up`.
- `control-plane/Dockerfile` — multi-stage; runtime image bundles `terraform` and `kubectl` so a fresh container can drive any provider.
- `web/Dockerfile` — Next.js standalone output for a slim production image.

### Added — CI and governance

- Required status checks on `main`: `Build, vet, test` (with the 45% Go coverage gate), `golangci-lint`, `Analyze (go)` (CodeQL), `Build, lint` (web), `govulncheck`, `Smoke (catalog load + healthz)`, `Site lint`.
- Branch protection: linear history, strict status checks, no force-push, conversation resolution required, auto-merge enabled.
- Dependabot for Go modules and GitHub Actions.
- `Makefile` with `build / test / cover / lint / vet / fmt / run / web-* / smoke / check / clean / compose-*`.
- `.editorconfig`, `.golangci.yml`, repo metadata (description, topics).

[Unreleased]: https://github.com/eltonlaice/ilabhu/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/eltonlaice/ilabhu/releases/tag/v0.1.0
