# Troubleshooting

Common things that go wrong while running ilabhu locally or against a real cloud account, and how to fix them.

Found something not covered here? Open an issue with the `troubleshooting` label and we'll fold it in.

## Control plane

### `ilabhud` fails to start: `load catalog`

The daemon walked `-exams-dir` and one of the `exam.yaml` files didn't parse, or two manifests share the same `id`.

```
{"level":"ERROR","msg":"load catalog","err":"exams/cka/two/exam.yaml: duplicate exam id \"cka/example\" (also defined in exams/cka/one)"}
```

**Fix.** Re-read the error path. Validate the offending file against [`docs/exam-schema.json`](exam-schema.json):

```sh
python3 -m venv /tmp/jsv && /tmp/jsv/bin/pip install -q jsonschema PyYAML
/tmp/jsv/bin/python3 -c '
import json, yaml, jsonschema, sys
schema = json.load(open("docs/exam-schema.json"))
exam = yaml.safe_load(open(sys.argv[1]))
jsonschema.validate(exam, schema)
print("ok")
' exams/cka/your-exam/exam.yaml
```

### `/healthz` returns but `/v1/exams` is empty

`ilabhud` started with no walkable `exam.yaml` files. Either the `-exams-dir` flag points at the wrong directory or the manifests are not named `exam.yaml`.

```sh
ls -la <your-exams-dir>/**/exam.yaml
```

Re-run with the correct flag:

```sh
./bin/ilabhud -addr :8080 -exams-dir ../exams
```

### `terraform: command not found`

The control plane shells out to your local Terraform binary. Install it from [hashicorp.com/terraform](https://developer.hashicorp.com/terraform/install) (or OpenTofu — the HCL is compatible) and confirm it's on `PATH`:

```sh
terraform -version
```

### `kubectl: command not found`

Same shape: the validator shells out to `kubectl`. Install via your package manager and confirm:

```sh
kubectl version --client
```

## AWS — provisioning a session

### `AccessDenied: not authorized to perform: sts:AssumeRole`

The IAM role's trust policy doesn't allow your caller's ARN.

Check who you are:

```sh
aws sts get-caller-identity --query Arn --output text
```

Then read back the trust policy:

```sh
aws iam get-role --role-name ilabhu-lab-runner \
  --query 'Role.AssumeRolePolicyDocument' --output json
```

The `Principal.AWS` in there must equal what `sts get-caller-identity` returned.

### `AccessDenied: ... external id`

The external id `ilabhud` sends doesn't match the trust policy's `sts:ExternalId` condition. Re-export the value you used at setup time and pass it on the API call:

```json
{"exam_id": "cka/warmup", "provider": "aws",
 "aws": {"role_arn": "...", "external_id": "PASTE_THE_RIGHT_ONE"}}
```

If you've lost the external id, generate a new one and update the trust policy with `aws iam update-assume-role-policy`. See [`docs/byo-cloud-setup.md`](byo-cloud-setup.md#rotating-the-external-id).

### `UnauthorizedOperation` mid-`terraform apply`

The permissions policy attached to the lab role is missing an action your exam's Terraform module needs. The exact action is in the CloudTrail error or in the apply output. Add it to the inline policy:

```sh
aws iam put-role-policy \
  --role-name ilabhu-lab-runner \
  --policy-name ilabhu-lab-runner-inline \
  --policy-document file://permissions-policy.json
```

The reference policy in [`docs/byo-cloud-setup.md`](byo-cloud-setup.md#step-2--attach-a-permissions-policy) covers EC2 compute, networking and key pairs. Add S3/RDS/etc as your exam touches them.

### `Could not locate AMI`

The exam's Terraform module is region-pinned. Confirm the region in `infrastructure.providers.aws.inputs.region` is enabled in your account, or pick a different region in the inputs:

```yaml
infrastructure:
  providers:
    aws:
      inputs:
        region: eu-west-1   # try us-east-1 if you only have it enabled
```

### Session stuck in `provisioning` for more than ~5 minutes

Check the control-plane logs — they tail Terraform output. If the EC2 instance is up but the kubeconfig fetch never completes, SSH into the node yourself with the per-session key under `~/.ilabhu/sessions/<id>/ssh_key` and inspect `/var/log/cloud-init-output.log`:

```sh
ssh -i ~/.ilabhu/sessions/<id>/ssh_key ubuntu@<public-ip> \
  sudo cat /var/log/cloud-init-output.log
```

Usually one of:

- k3s install failed (network glitch on the first apt update) — `sudo /usr/local/bin/k3s-uninstall.sh && curl -sfL https://get.k3s.io | sh -`
- security group is blocking 22 from your control plane's egress IP — widen `var.allowed_cidr` temporarily
- the AMI's user_data hasn't reached the `cp /etc/rancher/k3s/k3s.yaml ...` step yet — wait a bit longer

### Destroy hangs or fails

```
terraform destroy: ... InvalidParameterValue: ...
```

`terraform destroy` is best-effort. If it fails partway, the EC2 instance and its security group may still exist and **continue to bill**. Inspect:

```sh
aws ec2 describe-instances \
  --filters "Name=tag:ilabhu.session_id,Values=<session-id>" \
  --query 'Reservations[].Instances[].[InstanceId,State.Name]'
```

Delete by hand with `aws ec2 terminate-instances --instance-ids <id>`, then delete the security group and key pair (also tagged with the session id).

## Web (Next.js)

### Catalog shows "Could not reach the control plane"

`ilabhud` is not running on the configured base URL.

```sh
# default base
curl -sf http://127.0.0.1:8080/healthz
```

Either start the daemon (`make run` or `go run ./cmd/ilabhud -addr :8080 -exams-dir ../exams`) or point the web at a remote one:

```sh
ILABHU_API_BASE=https://my-ilabhud.internal npm run dev
```

### "Failed to parse URL from /api/v1/exams"

The fetch is running on the server side and tried to resolve a relative URL. This was [fixed in #23](https://github.com/eltonlaice/ilabhu/pull/23) — pull `main` and rebuild. If you've kept a local fork, see `web/src/lib/api.ts` for the runtime check (`typeof window === "undefined"`).

### `/sessions/<id>` says "Credentials are no longer in this browser session"

\`StartSessionForm\` stashes provider credentials in `sessionStorage` keyed by session id, scoped to the browser tab. Closing the tab — or opening the session URL in a new tab — loses them.

**Workaround.** Destroy the session via `curl`:

```sh
curl -X DELETE http://127.0.0.1:8080/v1/sessions/<id> \
  -H 'content-type: application/json' \
  -d '{"provider":"aws","aws":{"role_arn":"...","external_id":"..."}}'
```

Persistent server-side state will land with the Postgres backing store (roadmap).

## CI

### `Coverage X.X% is below the 45.0% threshold`

Your PR removed tests or added significant code without tests. Run `make cover` locally and add tests until total ≥ 45%. The gate exists to keep the trend up.

### `govulncheck` flagging stdlib vulns

`stdlib@goX.Y.Z` CVEs are fixed by bumping the Go toolchain. CI runs `go-version: stable` which usually picks up the latest patch. If you see this locally:

```sh
brew upgrade go         # macOS
# or: go install golang.org/dl/go1.X.Y@latest && go1.X.Y download
```

### `Build, lint` (web) fails with `set-state-in-effect`

A `useEffect` is calling a function that internally calls `setState`. Restructure to drive the update via a `refreshTick` state bump instead — see `web/src/app/sessions/[id]/page.tsx` for the pattern.

## Anything else

Open an issue. Include:

- the output of `make smoke` (when reproducible)
- the full ilabhud log lines around the failure
- the result of `curl -sf http://127.0.0.1:8080/v1/exams/<your-exam>` if it's a manifest issue
