# Security policy

## Reporting a vulnerability

If you find a security problem in ilabhu, please report it privately. Do **not** open a public GitHub issue.

Use GitHub's private vulnerability reporting:

> https://github.com/eltonlaice/ilabhu/security/advisories/new

You will get an acknowledgement within 72 hours. Critical issues that affect users with deployed environments will be patched and disclosed coordinated with the report.

## Scope

ilabhu provisions infrastructure in the user's own cloud account using credentials supplied by that user. The pieces that matter most for security review:

- **Credential handling.** The control plane assumes short-lived roles via `sts:AssumeRole` and never persists long-lived AWS keys. Reports of paths where credentials could leak into logs, on-disk state, or process environment outside their intended scope are highest priority.
- **Lab manifest execution surface.** A `lab.yaml` declares `validations` that run on the control plane (e.g. `kubectl`). Reports of ways a malicious manifest could escape its expected sandbox are highest priority.
- **Per-session isolation.** Each session has its own working directory, SSH key, and Terraform state. Reports of ways one session can interfere with another are highest priority.

## Out of scope

- Issues in third-party tools (`terraform`, `kubectl`, the AWS SDK). Report those upstream.
- Denial-of-service issues against a self-hosted instance through expected user actions (running too many sessions, etc.). Operators are expected to set their own quotas.

## Supported versions

ilabhu is pre-1.0. Only the `main` branch receives security fixes.
