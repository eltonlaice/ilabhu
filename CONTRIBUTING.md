# Contributing to ilabhu

Thanks for considering a contribution. ilabhu is small and early — direction is still being shaped, and well-scoped pull requests are the fastest way to influence it.

## Ways to contribute

- **Add an exam.** New exams (CKA, CKAD, CKS, RHCSA, ...) are the highest-leverage contribution. See [docs/authoring-an-exam.md](docs/authoring-an-exam.md) for a step-by-step walkthrough and [docs/exam-schema.md](docs/exam-schema.md) for the manifest reference.
- **Fix a bug** you hit while running the project locally.
- **Improve docs.** READMEs, examples, the lab schema reference, the BYO-cloud setup guide.
- **Add a provider adapter.** All five declared providers ship today (AWS, DigitalOcean, GCP, Azure, BYO-hosts); the next wave is Hetzner Cloud, Linode, Vultr, OVH, Scaleway. The path is mechanical — see [docs/adding-a-provider.md](docs/adding-a-provider.md).

## Before opening a large PR

For anything that changes public behavior — the `exam.yaml` schema, the HTTP API, the database model — please open an issue first to discuss the shape. Small fixes and docs changes don't need that step.

## Local development

Prerequisites:

- Go (the version pinned in [`control-plane/go.mod`](control-plane/go.mod))
- `terraform` 1.5+
- `kubectl`
- `ssh` and `ssh-keygen`

From the repo root, the bundled `Makefile` covers the common workflows:

```sh
make help        # list every target with its description
make build       # build ilabhud into control-plane/bin/
make test        # go test ./...
make lint        # golangci-lint
make run         # run ilabhud against ./exams on :8080
make smoke       # start ilabhud, curl /healthz and /v1/exams, then stop
make web-dev     # next dev on :3000
make check       # vet + lint + test + web-lint + web-build (everything CI checks)
```

Or invoke each step manually — see the `Makefile` and [`control-plane/README.md`](control-plane/README.md) for the underlying commands.

See [`control-plane/README.md`](control-plane/README.md) for the full set of flags and runtime details.

## Pull request flow

1. Fork the repo and create a topic branch off `main`. Branch name convention: `feat/<thing>`, `fix/<thing>`, `docs/<thing>`, `ci/<thing>`, `chore/<thing>`.
2. Keep the change focused. Unrelated cleanups belong in separate PRs.
3. Run `go build ./...`, `go vet ./...`, and `go test ./...` from `control-plane/` before pushing. CI will run them too.
4. Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages: `feat(...)`, `fix(...)`, `docs(...)`, `ci(...)`, `chore(...)`, `test(...)`, `refactor(...)`. The first line should fit in ~70 characters; explain the *why* in the body.
5. Open the PR against `main`. Fill in what changed, why, and how it was verified. Link any related issue.
6. CI must be green before merge. A maintainer will review and merge.

### Coverage

The Go workflow enforces a minimum overall coverage of **45%** for the `control-plane` packages. Run `make cover` locally to see the current number before opening a PR. If your PR adds significant code, please add tests so the gate continues to pass.

## Code style

- **Go**: standard `gofmt`. Prefer `errors.New` and `fmt.Errorf` with `%w` wrapping. Use `slog` for logging — never `log.Printf`. Keep packages small and cohesive.
- **YAML**: 2-space indentation, no trailing whitespace.
- **Markdown**: use ATX headings (`#`), keep line length reasonable but don't hard-wrap mid-paragraph.

## Reporting security issues

Please do **not** open public issues for security problems. See [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
