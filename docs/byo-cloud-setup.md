# BYO-cloud setup (AWS)

ilabhu provisions every exam in *your* AWS account. The control plane never holds long-lived credentials for your account — instead, it assumes a role you create, with an external id only it knows, and gets short-lived credentials for each exam session.

This guide walks you through the one-time setup. It takes about 10 minutes.

> GCP and Azure adapters are on the roadmap; only AWS is wired up today.

## Concepts at a glance

- **Trusted principal** — the AWS identity that runs `ilabhud`. For self-hosters that's typically the IAM user or role under which the daemon process runs.
- **Lab role** — an IAM role *you* create in your account, trusted by the trusted principal. ilabhu assumes this role to provision labs.
- **External id** — a per-deployment shared secret embedded in the role's trust policy. Defeats the [confused deputy](https://docs.aws.amazon.com/IAM/latest/UserGuide/confused-deputy.html) class of attack.

## Prerequisites

- An AWS account where you're willing to spend a few cents per exam session (a `t3.small` k3s node for two hours costs roughly \$0.05).
- AWS CLI configured locally for that account, or access to the AWS Console.
- The ARN of the trusted principal:
  - **Self-host on a workstation**: your IAM user, e.g. `arn:aws:iam::123456789012:user/elton`. Find it with `aws sts get-caller-identity --query Arn --output text`.
  - **Self-host on EC2 / ECS / EKS**: the role attached to the instance/task/pod (instance profile or IRSA role).
- An external id. **Generate a fresh one per deployment.** Anything unguessable works:

  ```sh
  openssl rand -hex 32
  ```

## Step 1 — Create the exam-runner role

Pick an external id and a trusted principal ARN, then export them:

```sh
export ILABHU_EXTERNAL_ID="$(openssl rand -hex 32)"
export ILABHU_TRUSTED_PRINCIPAL="arn:aws:iam::123456789012:user/elton"
```

Write the trust policy:

```sh
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "AWS": "${ILABHU_TRUSTED_PRINCIPAL}" },
    "Action": "sts:AssumeRole",
    "Condition": {
      "StringEquals": {
        "sts:ExternalId": "${ILABHU_EXTERNAL_ID}"
      }
    }
  }]
}
EOF
```

Create the role:

```sh
aws iam create-role \
  --role-name ilabhu-lab-runner \
  --assume-role-policy-document file://trust-policy.json
```

## Step 2 — Attach a permissions policy

The role needs to be allowed to do exactly what the exam Terraform modules do — and nothing more. The policy below covers the current catalog (single-node k3s on EC2). Expand it if you add exams that touch other services (RDS, S3, etc.).

```sh
cat > permissions-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EC2Compute",
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:TerminateInstances",
        "ec2:Describe*",
        "ec2:CreateTags",
        "ec2:DeleteTags"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2Networking",
      "Effect": "Allow",
      "Action": [
        "ec2:CreateSecurityGroup",
        "ec2:DeleteSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:AuthorizeSecurityGroupEgress",
        "ec2:RevokeSecurityGroupIngress",
        "ec2:RevokeSecurityGroupEgress"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2KeyPairs",
      "Effect": "Allow",
      "Action": [
        "ec2:ImportKeyPair",
        "ec2:DeleteKeyPair",
        "ec2:DescribeKeyPairs"
      ],
      "Resource": "*"
    }
  ]
}
EOF

aws iam put-role-policy \
  --role-name ilabhu-lab-runner \
  --policy-name ilabhu-lab-runner-inline \
  --policy-document file://permissions-policy.json
```

## Step 3 — Note the role ARN

```sh
aws iam get-role --role-name ilabhu-lab-runner --query 'Role.Arn' --output text
```

You will pass this ARN — together with the external id from Step 1 — to ilabhu when starting a session. **Treat the external id as a secret.**

## Step 4 — Test the assume-role from the trusted principal

Verify the trust before letting ilabhu try:

```sh
aws sts assume-role \
  --role-arn "$(aws iam get-role --role-name ilabhu-lab-runner --query 'Role.Arn' --output text)" \
  --role-session-name ilabhu-test \
  --external-id "${ILABHU_EXTERNAL_ID}" \
  --query 'Credentials.[AccessKeyId,Expiration]' \
  --output text
```

If you get back an access key id and an expiration timestamp, the setup is correct. If you get `AccessDenied`, double-check the trust policy's principal ARN and external id.

## Step 5 — Start a session

With `ilabhud` running locally:

```sh
curl -sX POST http://127.0.0.1:8080/v1/sessions \
  -H 'content-type: application/json' \
  -d "{
    \"exam_id\": \"cka/warmup\",
    \"provider\": \"aws\",
    \"aws\": {
      \"role_arn\": \"$(aws iam get-role --role-name ilabhu-lab-runner --query 'Role.Arn' --output text)\",
      \"external_id\": \"${ILABHU_EXTERNAL_ID}\"
    }
  }"
```

The response includes a `session.id`. Poll for status:

```sh
curl -s http://127.0.0.1:8080/v1/sessions/<id>
```

When `status` becomes `ready`, the response also carries `kubeconfig_b64`. Decode and use it:

```sh
curl -s http://127.0.0.1:8080/v1/sessions/<id> \
  | jq -r .kubeconfig_b64 | base64 -d > /tmp/lab.kubeconfig
KUBECONFIG=/tmp/lab.kubeconfig kubectl get nodes
```

When you're done, destroy the session — and remember, the underlying EC2 instance keeps billing until then:

```sh
curl -sX DELETE http://127.0.0.1:8080/v1/sessions/<id> \
  -H 'content-type: application/json' \
  -d "{
    \"provider\": \"aws\",
    \"aws\": {
      \"role_arn\": \"$(aws iam get-role --role-name ilabhu-lab-runner --query 'Role.Arn' --output text)\",
      \"external_id\": \"${ILABHU_EXTERNAL_ID}\"
    }
  }"
```

## Operational guidance

### Cost ceiling

ilabhu does not enforce per-account budgets. Set an [AWS Budget alert](https://docs.aws.amazon.com/cost-management/latest/userguide/budgets-create.html) at, say, \$5/month on the account so a forgotten session can't surprise you. A `t3.small` left running for a week is about \$4.

### Rotating the external id

Treat the external id like any other secret: rotate it if you suspect leakage.

```sh
NEW_ID="$(openssl rand -hex 32)"
# Edit trust-policy.json to use $NEW_ID, then:
aws iam update-assume-role-policy \
  --role-name ilabhu-lab-runner \
  --policy-document file://trust-policy.json
```

Update your `ilabhud` configuration with the new id. Old in-flight sessions that already obtained credentials remain valid until those credentials expire (1 hour by default).

### Tearing it all down

```sh
aws iam delete-role-policy --role-name ilabhu-lab-runner --policy-name ilabhu-lab-runner-inline
aws iam delete-role --role-name ilabhu-lab-runner
```

Make sure no sessions are in flight first — without the role, ilabhu cannot run `terraform destroy` and you'll have to clean up resources manually.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `AccessDenied: not authorized to perform: sts:AssumeRole` | Trust policy principal ARN does not match the caller. Run `aws sts get-caller-identity` and compare. |
| `AccessDenied: ... external id` | Mismatch between the external id sent by ilabhu and the one in the trust policy. |
| `UnauthorizedOperation` during `terraform apply` | Permissions policy is missing an action the lab module needs. Check CloudTrail for the exact action. |
| `Could not locate AMI` | The exam module is region-pinned. Confirm `infrastructure.providers.aws.inputs.region` in the exam's `exam.yaml` is enabled in your account. |
