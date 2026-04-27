# Lab manifest schema (v1)

A lab is a directory under `labs/<exam>/<lab-id>/` containing:

```
labs/cka/pod-resource-limits/
├── lab.yaml           # this manifest
├── terraform/         # Terraform module that provisions the lab
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
└── README.md          # human-readable copy of instructions
```

## `lab.yaml` fields

| Field | Type | Required | Description |
|---|---|---|---|
| `schema_version` | int | yes | Manifest schema version. Currently `1`. |
| `id` | string | yes | Globally unique lab id. Convention: `<exam>/<slug>`. |
| `version` | int | yes | Bumped when the lab changes in a way that invalidates in-flight sessions. |
| `exam` | string | yes | Exam code: `CKA`, `CKAD`, `CKS`, `RHCSA`, ... |
| `exam_objective` | string | yes | Exact citation from the official exam curriculum. |
| `title` | string | yes | Short human title. |
| `summary` | string | yes | One-line description. |
| `difficulty` | enum | yes | `easy` \| `medium` \| `hard`. |
| `estimated_minutes` | int | yes | Expected time to complete. |
| `infrastructure` | object | yes | Provisioning config. |
| `access` | object | yes | How the validator and the user reach the lab. |
| `instructions` | string | yes | Markdown shown in the UI. |
| `tasks` | array | yes | Ordered list of tasks with validations. |

### `infrastructure`

```yaml
infrastructure:
  provider: aws            # aws | gcp | azure
  module: ./terraform      # path relative to lab dir
  ttl_minutes: 120         # auto-destroy after this many minutes
  inputs:                  # variables passed to the Terraform module
    instance_type: t3.small
    region: eu-west-1
```

### `access`

Describes the connection details the control plane reads from Terraform outputs and uses to (a) drop the user into a terminal and (b) run validations.

```yaml
access:
  kind: kubeconfig         # kubeconfig | ssh
  output: kubeconfig       # name of the Terraform output to read
```

### `tasks[]`

```yaml
tasks:
  - id: create-pod
    title: Create the Pod
    instructions: |
      Create a Pod named `nginx-limited` in the `default` namespace ...
    validations:
      - kind: kubectl
        args: ["get", "pod", "nginx-limited", "-o", "jsonpath={.spec.containers[0].resources.limits.cpu}"]
        expect_equals: "200m"
```

#### Validation kinds (v1)

| `kind` | Inputs | Pass condition |
|---|---|---|
| `kubectl` | `args` (list), `expect_equals` \| `expect_contains` \| `expect_exit_code` | stdout / exit code matches |
| `shell` | `script` (string, runs over SSH on the lab VM), `expect_exit_code` (default `0`) | exit code matches |
| `http` | `url`, `expect_status` (default `200`), `expect_body_contains` | response matches |

Validations within a task are evaluated in order; all must pass for the task to be marked complete.
