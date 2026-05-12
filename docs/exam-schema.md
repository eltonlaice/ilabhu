# Exam manifest schema (v1)

An exam is a directory under `exams/<exam-code>/<slug>/` containing:

```
exams/cka/warmup/
├── exam.yaml          # this manifest
├── README.md          # human-readable copy of the exam
├── aws/               # Terraform module — AWS provider
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
├── digitalocean/      # Terraform module — DigitalOcean provider
├── gcp/               # Terraform module — Google Cloud provider
├── azure/             # Terraform module — Azure provider
└── byo-hosts/         # setup.sh + teardown.sh, run over SSH on user-provided hosts
```

Each exam declares one or more **providers**. The user picks one when starting a session; each provider has its own module (Terraform-driven cloud) or its own setup/teardown scripts (BYO-hosts).

> A machine-readable [JSON Schema](exam-schema.json) lives alongside this document. Add `# yaml-language-server: $schema=https://raw.githubusercontent.com/eltonlaice/ilabhu/main/docs/exam-schema.json` to the top of any `exam.yaml` to get autocomplete and validation in VS Code / Helix.

## `exam.yaml` fields

| Field | Type | Required | Description |
|---|---|---|---|
| `schema_version` | int | yes | Manifest schema version. Currently `1`. |
| `id` | string | yes | Globally unique exam id. Convention: `<exam-code>/<slug>` (e.g. `cka/warmup`). |
| `version` | int | yes | Bumped when the exam changes in a way that invalidates in-flight sessions. |
| `exam` | string | yes | Exam code: `CKA`, `CKAD`, `CKS`, `RHCSA`, ... |
| `title` | string | yes | Short human title. |
| `summary` | string | yes | One-line description. |
| `difficulty` | enum | yes | `easy` \| `medium` \| `hard`. |
| `estimated_minutes` | int | yes | Expected time to complete; informational. |
| `time_limit_minutes` | int | yes | Hard timer enforced by the UI. |
| `passing_score` | int | yes | Minimum weighted score (0-100) needed to pass. |
| `domains` | array | yes | Weighted curriculum domains; weights should sum to 100. |
| `infrastructure` | object | yes | Provisioning config and provider modules. |
| `access` | object | yes | How the validator and the user reach the environment. |
| `instructions` | string | yes | Markdown shown in the UI before tasks. |
| `tasks` | array | yes | Ordered tasks. Each task has its own validations and a domain attribution. |

### `domains[]`

```yaml
domains:
  - name: Workloads & Scheduling
    weight: 15
  - name: Cluster Architecture, Installation & Configuration
    weight: 25
  - name: Services & Networking
    weight: 20
  - name: Storage
    weight: 10
  - name: Troubleshooting
    weight: 30
```

The sum of weights should equal 100. Each task's `domain` field references one of these names, and each task's `weight` contributes proportionally inside that domain.

### `infrastructure`

```yaml
infrastructure:
  ttl_minutes: 120
  providers:
    aws:
      module: ./aws            # path relative to the exam dir
      inputs:
        region: eu-west-1
        instance_type: t3.small
    digitalocean:
      module: ./digitalocean
      inputs:
        region: ams3
        droplet_size: s-2vcpu-2gb
    byo-hosts:
      setup_script: ./byo-hosts/setup.sh
      teardown_script: ./byo-hosts/teardown.sh
      roles:
        - name: control-plane
          count: 1
          min_specs:
            cpu: 2
            ram_gb: 2
            distro: "ubuntu>=22.04|rocky>=9"
        - name: worker
          count: 2
          min_specs:
            cpu: 1
            ram_gb: 1
```

The user picks one provider at session-start time. The control plane validates that the credentials/host shape match the chosen provider's spec.

### `access`

```yaml
access:
  kind: kubeconfig         # kubeconfig | ssh
  output: kubeconfig       # name of the Terraform output to read
```

For BYO-hosts the access kind is the same; the setup script is responsible for emitting the kubeconfig (or SSH credentials) to a known location.

### `tasks[]`

```yaml
tasks:
  - id: create-pod
    title: Create the Pod
    domain: Workloads & Scheduling
    weight: 100        # weight inside the task's domain (sums to 100 per domain)
    instructions: |
      Create a Pod ...
    validations:
      - kind: kubectl
        args: ["get", "pod", "nginx-limited", "-o", "jsonpath={.status.phase}"]
        expect_equals: "Running"
```

#### Validation kinds (v1)

| `kind` | Inputs | Pass condition |
|---|---|---|
| `kubectl` | `args` (list), `expect_equals` \| `expect_contains` \| `expect_exit_code` | stdout / exit code matches |
| `shell` | `script` (piped to `/bin/sh` over SSH on the access host), `expect_exit_code` (default `0`), `expect_equals` \| `expect_contains` against the trimmed combined stdout+stderr | exit code matches expectation and any string assertion passes |
| `http` | `url` (with `{{public_ip}}` substitution), `expect_status` (default `200`), `expect_body_contains` | status matches expectation and (if set) body contains the substring |

Validations within a task are evaluated in order; all must pass for the task to be marked complete.
