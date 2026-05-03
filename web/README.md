# ilabhu web

Next.js 16 (App Router) frontend for the ilabhu control plane.

## Prerequisites

- Node.js 22 LTS or later (Next 16 requires 22+)
- A running [`ilabhud`](../control-plane/README.md) (default `http://127.0.0.1:8080`)

## Run

```sh
cd web
npm install
npm run dev
```

Open http://localhost:3000.

By default the browser hits `/api/*` on the same origin, and Next.js rewrites that to `http://127.0.0.1:8080`. Override with:

```sh
ILABHU_API_BASE=https://my-ilabhud.internal npm run dev
```

## Pages

| Route | Purpose |
|---|---|
| `/` | Exam catalog. Lists every exam the control plane reports under `GET /v1/exams`. |
| `/exams/[...slug]` | Exam detail + start session form (provider selector). |
| `/sessions/[id]` | Session monitor: status polling, kubeconfig download, per-task Validate, destroy. |

## Build

```sh
npm run build
npm run start
```

## Lint

```sh
npm run lint
```
