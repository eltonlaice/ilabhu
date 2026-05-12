# Authoring an exam

This guide walks you through adding a new certification exam to ilabhu. It assumes you've cloned the repo and read [`README.md`](../README.md) and [`docs/architecture.md`](architecture.md).

The shortest path: copy [`exams/cka/warmup/`](../exams/cka/warmup/) to a new directory, edit the manifest, edit the Terraform module, run it locally. Everything below expands on those four steps.

## 1. Pick an id and a directory

Convention: `exams/<exam-code>/<slug>/`.

| Exam | Directory |
|---|---|
| CKA — full simulation | `exams/cka/full/` |
| CKA — quick warmup | `exams/cka/warmup/` |
| CKAD — full simulation | `exams/ckad/full/` |
| RHCSA — full simulation | `exams/rhcsa/full/` |

The id stored in `exam.yaml` is the path under `exams/` joined by `/` — e.g. `cka/full`. It must be globally unique.

## 2. Write `exam.yaml`

The full schema lives in [`docs/exam-schema.md`](exam-schema.md). The minimum a manifest needs:

```yaml
schema_version: 1
id: ckad/full
version: 1
exam: CKAD
title: CKAD — Certified Kubernetes Application Developer
summary: One-line pitch shown on the catalog tile.
difficulty: medium                  # easy | medium | hard
estimated_minutes: 120              # informational
time_limit_minutes: 120             # the UI's hard timer
passing_score: 66                   # weighted threshold (0-100) to pass

domains:
  - name: Application Design and Build
    weight: 20
  - name: Application Deployment
    weight: 20
  - name: Application observability and maintenance
    weight: 15
  - name: Application Environment, Configuration and Security
    weight: 25
  - name: Services and Networking
    weight: 20

infrastructure:
  ttl_minutes: 180
  providers:
    aws:
      module: ./aws
      inputs:
        region: eu-west-1
        instance_type: t3.medium

access:
  kind: kubeconfig
  output: kubeconfig

instructions: |
  Multi-node Kubernetes cluster ready. Read each task carefully; the
  validations check exact resource names and namespaces.

tasks:
  - id: create-deployment
    title: Create a Deployment
    domain: Application Deployment
    weight: 100         # weight inside this task's domain (sums to 100 per domain)
    instructions: |
      Create a Deployment named `web` in the `apps` namespace ...
    validations:
      - kind: kubectl
        args: [get, deployment, web, -n, apps, -o, jsonpath={.spec.replicas}]
        expect_equals: "3"
```

### Tips

- **Domain weights sum to 100.** The catalog tile and (eventually) the scoring page rely on this.
- **Per-domain task weights also sum to 100.** Two tasks under the same domain with weights `60` and `40` contribute proportionally to that domain's bucket.
- **Passing score is overall.** A typical cert (CKA) passes at 66%. Pick what matches the real exam; a warmup that requires every task can use `passing_score: 100`.
- **Time-limit is enforced by the UI**, not yet by the control plane. The session runs as long as the TTL allows, but the timer is what learners practise against.

## 3. Add a Terraform module per provider

Each provider you declare in `infrastructure.providers` needs a real module or script that the control plane can drive. All five adapters ship today: `aws`, `digitalocean`, `gcp`, `azure` (Terraform modules), and `byo-hosts` (SSH-driven setup / teardown scripts). The control plane will reject a `provider` value not listed in your manifest's `providers:` map with `exam %q does not declare provider %q`.

### AWS (the reference implementation)

Drop your module under `<exam-dir>/aws/` with three files:

- `variables.tf` — must accept at least `session_id` (string) and `ssh_public_key` (string). The control plane passes these on every apply.
- `main.tf` — your provisioning. The Ubuntu AMI lookup, security group, key pair, and a `user_data` script that lays down `/home/ubuntu/kubeconfig` are already implemented in the warmup module — copy and adapt.
- `outputs.tf` — for `access.kind: kubeconfig`, you must export three outputs the control plane reads:
  - `public_ip` — the SSH-able IP of the cluster node that hosts the kubeconfig
  - `ssh_user` — the username the control plane uses for `scp` (typically `ubuntu`)
  - `kubeconfig_path_on_host` — the absolute path to the kubeconfig file on that node

The control plane SSHes in, polls until the kubeconfig file exists (max 5 minutes), `scp`s it down, and exposes it to the user via the API.

### Multi-node clusters

The current warmup is single-node. For full exams you'll typically want one control plane and 2–3 workers. The recommended path: have the control-plane node's `user_data` install k3s as `server`, surface its node-token, and have the workers' `user_data` join with `K3S_URL` + `K3S_TOKEN` from a known location (e.g. an EC2 user-data dependency or a small bootstrap S3 bucket — keep it self-contained).

Outputs come from the control-plane node only.

### Terraform conventions

- **Tag every resource** with `ilabhu.session_id`, `ilabhu.exam`, `ilabhu.managed_by` so leaked resources are easy to find and clean up.
- **Use the session id in resource names** so concurrent sessions don't collide.
- **Cap costs explicitly.** Pick the smallest instance shape that holds the workload. A learner running the full CKA on `t3.medium`s for 2 hours should cost well under \$1.
- **Open the security group only to the control plane's egress IP** if possible. Defaulting to `0.0.0.0/0` is acceptable for the warmup but not for production training.

### Other providers

Until adapters land, skip them. If you want to hand-write a `digitalocean/` or `gcp/` module ahead of the adapter, it'll be ready to plug in — but users won't be able to pick that provider in the UI.

## 4. Write tasks and validations

A task has `id`, `title`, `domain`, `weight`, `instructions`, and a list of `validations`. Validations run server-side (on the control plane host) — the user can't fake them by editing local files.

### Validation kinds

| `kind` | What it does | Inputs |
|---|---|---|
| `kubectl` | Runs `kubectl <args>` with the session's kubeconfig | `args`, `expect_equals` \| `expect_contains` \| `expect_exit_code` |
| `shell` | *(planned)* Runs a script over SSH on a chosen host | `script`, `expect_exit_code` |
| `http` | *(planned)* Hits an exposed URL | `url`, `expect_status`, `expect_body_contains` |

### Patterns for `kubectl`

Single-value assertion using `jsonpath`:

```yaml
- kind: kubectl
  args: [get, pod, nginx, -n, default, -o, "jsonpath={.status.phase}"]
  expect_equals: "Running"
```

Existence check (non-zero exit if missing):

```yaml
- kind: kubectl
  args: [get, secret, db-creds, -n, apps]
  expect_exit_code: 0
```

Contains check (multiple values, parsing-tolerant):

```yaml
- kind: kubectl
  args: [get, networkpolicy, -n, ingress, -o, name]
  expect_contains: "networkpolicy.networking.k8s.io/deny-all"
```

### Writing tasks well

- **One observable outcome per task.** "Create a Deployment AND a Service AND configure a NetworkPolicy" is three tasks, not one.
- **Be unambiguous on names and namespaces.** `web` / `apps` is unambiguous; "your deployment" is not.
- **Validations are AND, not OR.** If any validation fails, the task fails. Don't write three validations where one would do.
- **Match the real exam's wording style.** Imperative, short, exact resource names in backticks.

## 5. Test locally

```sh
# from repo root
make smoke               # confirms ilabhud can load all manifests
make run                 # starts ilabhud on :8080
# in another terminal
curl http://127.0.0.1:8080/v1/exams | jq
curl http://127.0.0.1:8080/v1/exams/<your-id> | jq
```

If the manifest is invalid, `make smoke` fails with a clear error and the file path. Common mistakes:

- Missing required field → `parseManifest: missing required field: <name>`
- Wrong schema version → `unsupported schema_version <n> (expected 1)`
- Two manifests with the same id → `duplicate exam id "<id>"`

For an actual end-to-end test against AWS, follow [`docs/byo-cloud-setup.md`](byo-cloud-setup.md) and start a session against the new exam.

## 6. Submit a PR

- Branch name: `feat/exam-<exam-code>-<slug>` — e.g. `feat/exam-ckad-full`.
- One exam per PR. If you're touching the schema or shared infrastructure, separate that into its own PR first.
- Include in the PR body: the certification this targets, which domains and tasks are covered (or explicitly out of scope), what providers ship Terraform modules, and the cost estimate per session.
- Run `make check` locally before pushing.

Add yourself to the contributors section of `README.md` if it's your first contribution.

## Anti-patterns

- **Do not commit Terraform state.** `*.tfstate` is in `.gitignore`; per-session state lives under `~/.ilabhu/sessions/<id>/`.
- **Do not embed credentials in `inputs:`.** Every secret travels through the session-start request, never through the manifest.
- **Do not re-use a published `id`** for an incompatible rewrite. Bump the major part of the slug instead — e.g. `cka/full` and later `cka/full-v2` — so in-flight sessions on the old definition don't get scored against new validations.
- **Do not hand-roll `terraform apply` outside `<exam-dir>/<provider>/`.** The control plane copies the module into a per-session workdir; if you script around it, two concurrent sessions will collide on shared state.

## Questions?

Open an issue with the `question` label, or read the manifest schema reference in [`docs/exam-schema.md`](exam-schema.md).
