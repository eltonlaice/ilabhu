# Self-host with Docker Compose

The shortest path from "I just cloned this repo" to "ilabhu is running on `localhost:3000`".

## Prerequisites

- Docker 24+ with Compose v2 (bundled with current Docker Desktop / Docker Engine).

## Bring it up

From the repo root:

```sh
docker compose -f deploy/docker-compose.yml up --build
```

First build takes 3–5 minutes (Go toolchain + Next.js bundle). Subsequent builds reuse the layer cache and finish in seconds.

Then open <http://localhost:3000>. The catalog should list the warmup exam.

## Bring it down

```sh
docker compose -f deploy/docker-compose.yml down
```

Add `-v` to also remove the named volume `ilabhu-state` (per-session terraform state and SSH keys).

## What's in the bundle

| Service | Image | Port | Notes |
|---|---|---|---|
| `ilabhud` | built from `../control-plane/Dockerfile` | `8080:8080` | Mounts `../exams` read-only at `/exams`; persists `/var/lib/ilabhu` in a named volume |
| `web` | built from `../web/Dockerfile` (Next.js standalone) | `3000:3000` | `ILABHU_API_BASE=http://ilabhud:8080` so the rewrite resolves inside the compose network |

The control-plane image bundles `terraform` (pinned via the `TERRAFORM_VERSION` build arg) and `kubectl` (pinned via `KUBECTL_VERSION`) so a fresh container can drive any of the five providers without extra installation.

## When to use this vs `make run`

- **`make run`** — local dev: `ilabhud` runs against the source tree directly, no rebuild between edits.
- **`docker compose up`** — what users self-host with: production-shaped image, isolated, restartable.

CI's smoke job exercises the `make run` path; we have not yet wired a compose-up smoke. It is on the roadmap.

## Bumping pinned versions

```sh
docker compose -f deploy/docker-compose.yml build \
  --build-arg TERRAFORM_VERSION=1.10.4 \
  --build-arg KUBECTL_VERSION=v1.31.4 \
  ilabhud
```
