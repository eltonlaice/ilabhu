# ilabhud — ilabhu control plane

`ilabhud` is the daemon that turns an `exam.yaml` manifest into a running, BYO-cloud (or BYO-host) exam session. It owns the session lifecycle: assume the user's cloud role (or accept the user's host list), copy the matching provider's Terraform module — or run the BYO-hosts setup script — into a per-session workdir, apply, fetch access details, and tear down on demand.

## Prerequisites

- **Go** — version pinned in [`go.mod`](go.mod)
- **Terraform** 1.5+ on `PATH` (for cloud providers)
- **kubectl** on `PATH` (used by the validator for `kind: kubectl` checks)
- **ssh** and **ssh-keygen** on `PATH` (per-session keypair, kubeconfig fetch, BYO-hosts setup)
- **AWS credentials** for the control plane itself, resolved via the standard AWS chain. Used only to call `sts:AssumeRole` against the user's account.

## Build

```sh
cd control-plane
go build -o bin/ilabhud ./cmd/ilabhud
```

## Run

```sh
./bin/ilabhud \
  -addr :8080 \
  -exams-dir ../exams \
  -state-dir ~/.ilabhu
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-addr` | `:8080` | HTTP listen address. |
| `-exams-dir` | `../exams` | Directory containing exam manifests. The daemon walks this on startup. |
| `-state-dir` | `~/.ilabhu` | Where per-session Terraform state and SSH keys live. Created if missing. |

The daemon emits structured JSON logs on stdout (`slog`).

## HTTP API (v0)

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/healthz` | Liveness probe. |
| `GET` | `/v1/exams` | List exams found in `-exams-dir`. |
| `GET` | `/v1/exams/{id...}` | Full exam manifest (tasks, domains, providers). |
| `POST` | `/v1/sessions` | Start a session. Body: `{"exam_id": "...", "provider": "aws", "aws": {"role_arn": "...", "external_id": "..."}}`. Returns `202` with the session record. |
| `GET` | `/v1/sessions/{id}` | Read session status, outputs, and (once `ready`) base64-encoded kubeconfig. |
| `DELETE` | `/v1/sessions/{id}` | Tear down. Accepts the same provider credentials body as create. |
| `POST` | `/v1/sessions/{id}/tasks/{task_id}/validate` | Run a task's validations against the session. |

### Smoke test

```sh
go run ./cmd/ilabhud -addr :8080 -exams-dir ../exams &
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/v1/exams
```

## Layout

```
control-plane/
├── cmd/ilabhud/        # entrypoint
└── internal/
    ├── api/            # HTTP handlers (stdlib net/http, Go 1.22 mux patterns)
    ├── catalog/        # parses exams/<...>/exam.yaml on startup
    ├── cloud/aws/      # sts:AssumeRole helper
    ├── provisioner/    # shells out to terraform
    ├── session/        # in-memory session store + lifecycle manager
    └── validator/      # runs exam.yaml validations
```

## Limitations (current)

- Sessions live in memory; restarting the daemon loses them. Postgres is on the roadmap.
- Sessions are not auto-destroyed on TTL expiry yet — call `DELETE` manually.
- Only the AWS provider is implemented. GCP, Azure, DigitalOcean and BYO-hosts adapters live alongside `cloud/aws/` in follow-up PRs.
- No authentication on the HTTP API. Run it on `localhost` for now.
