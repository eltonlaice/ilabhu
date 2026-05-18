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

CI exercises both paths: `Smoke (catalog load + healthz)` covers `make run`, and `Compose smoke` covers `docker compose up --build` end-to-end against the same in-tree exams catalog. Locally, you can reproduce the compose path with `make compose-smoke`.

## Use a published image instead of building locally

Every release tag is published to two registries — pick whichever you prefer:

| Registry | `ilabhud` | `ilabhu-web` |
|---|---|---|
| GHCR | <https://github.com/eltonlaice/ilabhu/pkgs/container/ilabhud> | <https://github.com/eltonlaice/ilabhu/pkgs/container/ilabhu-web> |
| Docker Hub | <https://hub.docker.com/r/eltonlaicedev/ilabhud> | <https://hub.docker.com/r/eltonlaicedev/ilabhu-web> |

The `deploy/docker-compose.yml` defaults to GHCR. To use Docker Hub, override the image lines or set `ILABHU_REGISTRY=docker.io/eltonlaicedev` in your shell before running compose.

```sh
ILABHU_IMAGE_TAG=v0.1.1 docker compose -f deploy/docker-compose.yml up \
  --no-build --pull always
```

Available platforms: `linux/amd64`, `linux/arm64`. Both registries carry the same digests, so `gh attestation verify` against either coordinates works. Builds are produced with [build provenance](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds) and an SBOM attached to each image:

```sh
# Verify from GHCR
gh attestation verify oci://ghcr.io/eltonlaice/ilabhud:v0.1.1 --owner eltonlaice

# Or from Docker Hub
gh attestation verify oci://docker.io/eltonlaicedev/ilabhud:v0.1.1 --owner eltonlaice
```

## Bumping pinned versions (local build)

```sh
docker compose -f deploy/docker-compose.yml build \
  --build-arg TERRAFORM_VERSION=1.10.4 \
  --build-arg KUBECTL_VERSION=v1.31.4 \
  ilabhud
```
