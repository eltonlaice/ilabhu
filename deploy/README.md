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

## Use a published image instead of building locally

Starting with v0.1.0, every release tag is published to GHCR:

- <https://github.com/eltonlaice/ilabhu/pkgs/container/ilabhud>
- <https://github.com/eltonlaice/ilabhu/pkgs/container/ilabhu-web>

To run the bundle without building anything, pin the tag and skip the build step:

```sh
ILABHU_IMAGE_TAG=v0.1.0 docker compose -f deploy/docker-compose.yml up \
  --no-build --pull always
```

Available platforms: `linux/amd64`, `linux/arm64`. Builds are produced with [build provenance](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds) and an SBOM attached to each image; verify with:

```sh
gh attestation verify oci://ghcr.io/eltonlaice/ilabhud:v0.1.0 \
  --owner eltonlaice
```

## Bumping pinned versions (local build)

```sh
docker compose -f deploy/docker-compose.yml build \
  --build-arg TERRAFORM_VERSION=1.10.4 \
  --build-arg KUBECTL_VERSION=v1.31.4 \
  ilabhud
```
