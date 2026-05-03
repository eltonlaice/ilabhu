# Architecture

This document describes how ilabhu turns a `lab.yaml` manifest into a running, validated lab session in a user-owned cloud account, and the design decisions behind that flow.

## Components

```
┌─────────────────┐     HTTP      ┌──────────────────────┐
│  Web UI         │──────────────▶│   ilabhud            │
│  (Next.js, M1)  │               │   (control plane)    │
└─────────────────┘               └──────────┬───────────┘
                                             │
                              ┌──────────────┼──────────────┐
                              ▼              ▼              ▼
                       ┌────────────┐ ┌───────────┐  ┌──────────────┐
                       │  catalog   │ │  session  │  │  validator   │
                       │  (lab.yaml)│ │  manager  │  │  (kubectl)   │
                       └────────────┘ └─────┬─────┘  └──────────────┘
                                            │
                                            ▼
                                   ┌────────────────┐
                                   │  provisioner   │
                                   │  (terraform)   │
                                   └────────┬───────┘
                                            │
                              sts:AssumeRole + Terraform apply
                                            ▼
                                   ┌────────────────┐
                                   │  User's cloud  │
                                   │  account       │
                                   └────────────────┘
```

| Component | Package | Role |
|---|---|---|
| Catalog | `internal/catalog` | Walks `labs/` on startup, parses every `lab.yaml`, exposes a read-only index keyed by lab id. |
| Session manager | `internal/session` | Owns the lifecycle: create record → assume role → copy module → terraform apply → fetch kubeconfig → ready. Mirror flow on destroy. |
| Provisioner | `internal/provisioner` | Thin wrapper around the user's `terraform` binary. Per-session workdir under `~/.ilabhu/sessions/<id>/module/`. |
| Cloud adapter | `internal/cloud/aws` | `sts:AssumeRole` with external-id, returns short-lived credentials as Terraform-compatible env vars. |
| Validator | `internal/validator` | Runs the `validations` block of a task — currently `kind: kubectl`, with `shell` and `http` planned. |
| HTTP API | `internal/api` | Stdlib `net/http` with Go 1.22 mux patterns. JSON in, JSON out. |

## Lifecycle of a session

1. **Client posts to `/v1/sessions`** with `{lab_id, aws_role_arn, aws_external_id}`.
2. **Manager allocates a session record** in memory with `status: provisioning` and a fresh UUID.
3. **Manager prepares a workdir** under `~/.ilabhu/sessions/<session_id>/`. The lab's Terraform module is *copied* into `module/` so `terraform apply` writes its state inside the session's directory and never touches the source tree.
4. **Manager generates an ed25519 SSH keypair** for this session via `ssh-keygen`. The public key is passed as a Terraform input (`ssh_public_key`). The private key never leaves the control-plane host.
5. **Cloud adapter calls `sts:AssumeRole`** against the user's account using the role ARN and the external id. Returns short-lived credentials with a 1-hour TTL.
6. **Provisioner runs** `terraform init && terraform apply -auto-approve` with the assumed-role credentials in the environment and the lab manifest's `infrastructure.inputs` plus the session id and SSH public key as variables.
7. **Manager reads outputs** via `terraform output -json`. For `access.kind: kubeconfig` labs, three outputs are required: `public_ip`, `ssh_user`, `kubeconfig_path_on_host`.
8. **Manager fetches the kubeconfig** by polling `ssh user@host test -f <path>` until it appears, then `scp`-ing it down. The user's `user_data` script is responsible for writing the file once the cluster is ready.
9. **Session transitions to `ready`.** The HTTP API exposes the kubeconfig (base64-encoded) and the outputs map.
10. **Client posts to `/v1/sessions/{id}/tasks/{task_id}/validate`.** The validator writes the kubeconfig to a temp file, runs each validation in order, and returns the per-validation pass/fail breakdown.
11. **Client deletes the session.** The manager re-assumes the role (the original credentials may have expired) and runs `terraform destroy`.

## Why these choices

### One Terraform module per lab, copied per session

Each lab ships its own Terraform module under `labs/<exam>/<id>/terraform/`. The control plane never points Terraform at the source tree directly — it copies the module into the session workdir first.

- **Isolation.** State, lock files, and downloaded providers stay scoped to the session. Two concurrent sessions of the same lab don't fight over `terraform.tfstate`.
- **Versioning.** A future lab edit doesn't invalidate in-flight sessions; their module copy is frozen at the moment of `apply`.
- **Self-host portability.** The user installs `terraform` once and we drive their binary. We don't link the HCL library and we don't ship a custom runtime.

### BYO-cloud via `sts:AssumeRole` + external id

The control plane never holds long-lived credentials for any user.

- The user creates an IAM Role in *their* account whose trust policy names the ilabhu deployment's principal and an external id that ilabhu generates per-user.
- Each session call rotates short-lived credentials (1 h default) for that single operation.
- If the control-plane host is ever compromised, the blast radius is whatever fits in the role's IAM policy and the credentials' TTL — not full account access.

The external id requirement defeats the [confused deputy](https://docs.aws.amazon.com/IAM/latest/UserGuide/confused-deputy.html) attack: another customer cannot guess the deputy's role and trick the control plane into acting on their behalf.

### Validations run on the control plane, not on the lab VM

A lab declares validations like:

```yaml
validations:
  - kind: kubectl
    args: [get, pod, nginx-limited, -o, jsonpath={.status.phase}]
    expect_equals: Running
```

These run on the control-plane host with `KUBECONFIG` pointing at the session's kubeconfig. Running them server-side means:

- The validation result is authoritative — the user can't fake it from inside the VM.
- Validations don't add agent code to the lab VM. The VM is whatever the user is meant to be learning on, with no ilabhu instrumentation.

The trade-off is that validation kinds are limited to what the control plane can express remotely. `kind: kubectl` already covers most CKA-style assertions. `kind: shell` (over SSH) and `kind: http` are planned for cases where shelling into the VM or hitting an exposed endpoint is the most natural assertion.

### Stateless control plane (for now)

The session store is in-memory. Restarting `ilabhud` loses the session map, though the underlying cloud resources still exist (Terraform state lives on disk under the session workdir). This is deliberate for v0 — the API and lifecycle shape lock first; persistence comes when we know the data model is right. Postgres is on the roadmap.

### Stdlib HTTP, no framework

`net/http` with Go 1.22 mux patterns covers everything the API needs (path parameters, method routing). Adding `chi`, `gin`, or `echo` would buy little and add a maintenance surface. If we ever need middleware chains beyond what stdlib offers cleanly, we'll revisit.

## Relevant files

- [`docs/lab-schema.md`](lab-schema.md) — `lab.yaml` reference.
- [`docs/byo-cloud-setup.md`](byo-cloud-setup.md) — IAM policy + role trust setup.
- [`control-plane/README.md`](../control-plane/README.md) — running the daemon, HTTP API surface.
- [`control-plane/internal/session/manager.go`](../control-plane/internal/session/manager.go) — the lifecycle code described above.
