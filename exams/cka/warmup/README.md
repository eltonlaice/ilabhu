# CKA — Warmup

Single-task warmup that exercises the ilabhu pipeline end-to-end. Not the real CKA exam; the full simulation lives at `exams/cka/full/` (TODO).

See [`exam.yaml`](./exam.yaml) for the manifest. See [`docs/exam-schema.md`](../../../docs/exam-schema.md) for the schema.

## Infrastructure

| Provider | Status |
|---|---|
| AWS | ✅ Single EC2 (default `t3.small`, `eu-west-1`) running [k3s](https://k3s.io). |
| DigitalOcean | ⏳ Planned. |
| GCP | ⏳ Planned. |
| Azure | ⏳ Planned. |
| BYO-hosts | ⏳ Planned. |

## Cost (AWS, approximate)

`t3.small` in `eu-west-1` is ~\$0.0228/hour. A 2-hour TTL session costs ~\$0.05 in compute, plus negligible EBS and data transfer.
