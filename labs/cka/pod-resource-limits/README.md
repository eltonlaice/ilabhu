# CKA — Pod with resource limits

Create a Pod that has CPU and memory limits enforced.

See [lab.yaml](./lab.yaml) for the full manifest. See [docs/lab-schema.md](../../../docs/lab-schema.md) for the schema.

## Infrastructure

Single AWS EC2 instance (default `t3.small` in `eu-west-1`) running [k3s](https://k3s.io). The Terraform module exports a `kubeconfig` output that the control plane uses to drive `kubectl` for both the user's terminal session and validation.

## Cost (approximate)

`t3.small` in `eu-west-1` is ~\$0.0228/hour. A 2-hour TTL session costs ~\$0.05 in compute, plus negligible EBS and data transfer.
