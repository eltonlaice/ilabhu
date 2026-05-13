# Adding a provider adapter

ilabhu ships with five provider adapters (AWS, DigitalOcean, GCP, Azure, BYO-hosts). Adding a sixth — Hetzner Cloud, Linode, Vultr, OVH, Scaleway, OCI, IBM Cloud — follows a fixed shape. This guide walks through it from the inside out.

> See [`docs/authoring-an-exam.md`](authoring-an-exam.md) for the *content* side (adding a new exam). This document is about the *transport* side (adding a new way for the control plane to provision an environment).

## 1. Decide which kind of adapter you're writing

ilabhu has two adapter shapes:

| Shape | When | Examples |
|---|---|---|
| **Terraform-driven** | The provider has a Terraform provider that can spin up a Linux VM with `user_data` (or equivalent) and expose a public IPv4. | aws, digitalocean, gcp, azure |
| **SSH-driven** | The provider gives you ssh access to already-running Linux hosts; ilabhu runs setup / teardown scripts. | byo-hosts |

Almost every new public-cloud adapter is Terraform-driven. SSH-driven is for the "your own hosts" path and for niche cases where Terraform support is missing.

A provider is a candidate for inclusion in this repo if it gives you:

- A Terraform provider on the [Terraform Registry](https://registry.terraform.io/browse/providers) that can `apply` a Linux VM with cloud-init / user-data.
- An auth model that fits in a single struct (token, key-pair, JSON blob, etc.) — long-lived enough that the user can paste it once per session.
- A public IPv4 (or v6) the kubeconfig fetch can reach over SSH.
- An "Ubuntu 24.04 LTS" or "Rocky 9" image identifiable by a known name / family.

## 2. Three layers to touch

A provider adapter cuts through:

```
control-plane/internal/cloud/<provider>/      ← new package
control-plane/internal/session/manager.go     ← two switch cases
control-plane/internal/api/sessions.go        ← request-body block + plumbing
web/src/lib/api.ts                            ← credential type
web/src/components/StartSessionForm.tsx       ← chip + fields component
exams/<exam>/<slug>/<provider>/               ← per-exam Terraform module
exams/<exam>/<slug>/exam.yaml                 ← providers.<provider> block
docs/byo-cloud-setup.md                       ← walkthrough section
docs/cost-comparison.md                       ← row + per-month projections
```

Skipping any of these = the chip in the UI is a dead-end at session start.

## 3. Naming conventions

| Surface | Convention | Examples |
|---|---|---|
| Go package | provider keyword, lowercase | `aws`, `digitalocean`, `gcp`, `azure` |
| Credentials struct | `Credentials` (each package owns its own) | `aws.Credentials` |
| Env var prefix | follow the Terraform provider's expectations | `AWS_`, `DIGITALOCEAN_`, `GOOGLE_`, `ARM_`, `HCLOUD_` |
| Request body block | `lower_snake_case`, matches the chip id | `aws`, `digitalocean`, `gcp`, `azure`, `byo_hosts` |
| Chip id (frontend) | matches `infrastructure.providers.<key>` exactly | `aws`, `digitalocean`, `gcp`, `azure`, `byo-hosts` |
| Terraform module dir | matches the chip id | `exams/cka/warmup/digitalocean/` |

The chip id is the one identifier that bridges every layer — keep it identical everywhere.

## 4. Backend skeleton

### 4.1 `internal/cloud/<provider>/client.go`

```go
package hetzner

import "errors"

type Credentials struct {
    Token string
}

func (c Credentials) Validate() error {
    if c.Token == "" {
        return errors.New("hetzner.token is required")
    }
    return nil
}

// AsEnv returns env vars in the shape the Terraform provider for this
// cloud reads natively. For Hetzner Cloud that's HCLOUD_TOKEN.
func (c Credentials) AsEnv() []string {
    return []string{"HCLOUD_TOKEN=" + c.Token}
}
```

If your auth model needs server-side parsing (e.g. extract a project id from a JSON SA key, like `cloud/gcp`), do it in a constructor:

```go
func NewCredentials(raw string) (Credentials, error) {
    // parse raw, return Credentials{...} or an error
}
```

Tests live next to it: `client_test.go`. Cover the AsEnv shape and every per-field validation path.

### 4.2 `internal/session/manager.go`

Two switch cases — one in `validateProviderInput`, one in `resolveEnv`:

```go
case "hetzner":
    if input.Hetzner == nil || input.Hetzner.Token == "" {
        return errors.New("hetzner.token is required")
    }
```

```go
case "hetzner":
    return hetznercloud.Credentials{Token: input.Hetzner.Token}.AsEnv(), nil
```

Add the credentials type to `StartInput`:

```go
type StartInput struct {
    Provider     string
    AWS          *AWSCredentials
    DigitalOcean *DOCredentials
    GCP          *GCPCredentials
    Azure        *AzureCredentials
    Hetzner      *HetznerCredentials   // ← new
    BYOHosts     *BYOHostsCredentials
}

type HetznerCredentials struct {
    Token string
}
```

The `provisionTerraform` and `Destroy` paths are already provider-agnostic — they just call `resolveEnv` and pass the result to `terraform`.

### 4.3 `internal/api/sessions.go`

Add the request-body block and plumb it through `toStartInput`:

```go
type hetznerCredsBody struct {
    Token string `json:"token"`
}

type sessionRequest struct {
    // ...existing fields
    Hetzner *hetznerCredsBody `json:"hetzner,omitempty"`
}

func (r *sessionRequest) toStartInput() session.StartInput {
    // ...existing
    if r.Hetzner != nil {
        in.Hetzner = &session.HetznerCredentials{Token: r.Hetzner.Token}
    }
    return in
}
```

Add a `TestCreateSession_SuccessHetzner` mirroring the other provider tests.

## 5. Frontend

### 5.1 `web/src/lib/api.ts`

```ts
export type Provider = "aws" | "digitalocean" | "gcp" | "azure" | "hetzner" | "byo-hosts";

export type ProviderCredentials = {
  provider: Provider;
  // ...existing
  hetzner?: { token: string };
};
```

### 5.2 `web/src/components/StartSessionForm.tsx`

Add the chip to `IMPLEMENTED` and a small `HetznerFields` component:

```tsx
const IMPLEMENTED: Provider[] = ["aws", "digitalocean", "gcp", "azure", "hetzner", "byo-hosts"];
```

```tsx
{provider === "hetzner" ? (
  <HetznerFields token={hetznerToken} setToken={setHetznerToken} />
) : ...}
```

`HetznerFields` takes one `<input type="password">` for the token plus a link to the Hetzner Cloud token settings page. Reuse the styling pattern from `DOFields`.

## 6. Per-exam Terraform module

Each existing exam needs a module under `exams/<exam>/<slug>/<provider>/` for your new provider. Start with the warmup:

```
exams/cka/warmup/hetzner/
├── main.tf
├── variables.tf
└── outputs.tf
```

Required outputs (the control plane reads these): `public_ip`, `ssh_user`, `kubeconfig_path_on_host`.

Required user_data behaviour: install k3s, copy `/etc/rancher/k3s/k3s.yaml` to `~/kubeconfig` with `127.0.0.1` substituted for the public IP. See `exams/cka/warmup/digitalocean/main.tf` for a minimal reference.

Update `exam.yaml`:

```yaml
infrastructure:
  providers:
    hetzner:
      module: ./hetzner
      inputs:
        location: nbg1
        server_type: cx22
```

## 7. Docs to update in the same PR

| File | What |
|---|---|
| `docs/byo-cloud-setup.md` | New section with step-by-step token creation. |
| `docs/cost-comparison.md` | New row in the per-session table; updated decision tree if your provider has a unique angle (cheapest in a region, free tier, etc.). |
| `docs/exam-schema.md` | Update the example directory tree if the layout changes. |
| `README.md` | Add the provider to the Status table and Roadmap. |

The site linter (CI workflow `Site lint`) does not yet enforce these but a maintainer will ask for them on review.

## 8. PR checklist

Before opening:

- [ ] `make check` clean — Go build, vet, lint, test, web build, web lint
- [ ] `make cover` ≥ the current gate (45%); a new credentials package usually pushes this *up*
- [ ] Smoke test: `make smoke` resolves the new provider on `/v1/exams/<id>` outputs
- [ ] New tests: at least `Validate_*`, `AsEnv`, and `TestCreateSession_Success<Provider>`
- [ ] `docs/cost-comparison.md` updated
- [ ] `docs/byo-cloud-setup.md` walk-through added
- [ ] `README.md` Status table updated
- [ ] Frontend chip is selectable; submitting with empty fields shows a useful error
- [ ] Tear-down works (provider's Terraform `destroy` returns 0)

## 9. Worked references

The cleanest references in the codebase:

- **Single token** — [`internal/cloud/digitalocean/`](../control-plane/internal/cloud/digitalocean/). Mirror this for any provider that takes a PAT.
- **Server-side credential parsing** — [`internal/cloud/gcp/`](../control-plane/internal/cloud/gcp/). Project id extracted from the SA-key JSON.
- **Multi-field secret** — [`internal/cloud/azure/`](../control-plane/internal/cloud/azure/). Four fields, per-field validation.
- **SSH-driven** — [`internal/session/byohosts.go`](../control-plane/internal/session/byohosts.go). Different shape; only relevant if your "provider" is "the user's own hosts".

The shortest path from zero to a merged adapter PR is to copy the DigitalOcean adapter (Go + Terraform + frontend) and rename. Most providers are PAT-shaped; the rest is naming.
