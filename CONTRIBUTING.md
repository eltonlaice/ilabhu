# Contributing to ilabhu

Thanks for considering a contribution. ilabhu is small and early — direction is still being shaped, and well-scoped pull requests are the fastest way to influence it.

## Ways to contribute

- **Add an exam.** New exams (CKA, CKAD, CKS, RHCSA, ...) are the highest-leverage contribution. See [docs/exam-schema.md](docs/exam-schema.md) for the manifest format.
- **Fix a bug** you hit while running the project locally.
- **Improve docs.** READMEs, examples, the lab schema reference, the BYO-cloud setup guide.
- **Add a cloud adapter.** GCP and Azure adapters are wanted; the AWS adapter at `control-plane/internal/cloud/aws/` is the reference shape.

## Before opening a large PR

For anything that changes public behavior — the `exam.yaml` schema, the HTTP API, the database model — please open an issue first to discuss the shape. Small fixes and docs changes don't need that step.

## Local development

Prerequisites:

- Go (the version pinned in [`control-plane/go.mod`](control-plane/go.mod))
- `terraform` 1.5+
- `kubectl`
- `ssh` and `ssh-keygen`

Build and run the control plane:

```sh
cd control-plane
go build ./...
go vet ./...
go test ./...
go run ./cmd/ilabhud -addr :8080 -exams-dir ../exams
```

See [`control-plane/README.md`](control-plane/README.md) for the full set of flags and runtime details.

## Pull request flow

1. Fork the repo and create a topic branch off `main`. Branch name convention: `feat/<thing>`, `fix/<thing>`, `docs/<thing>`, `ci/<thing>`, `chore/<thing>`.
2. Keep the change focused. Unrelated cleanups belong in separate PRs.
3. Run `go build ./...`, `go vet ./...`, and `go test ./...` from `control-plane/` before pushing. CI will run them too.
4. Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages: `feat(...)`, `fix(...)`, `docs(...)`, `ci(...)`, `chore(...)`, `test(...)`, `refactor(...)`. The first line should fit in ~70 characters; explain the *why* in the body.
5. Open the PR against `main`. Fill in what changed, why, and how it was verified. Link any related issue.
6. CI must be green before merge. A maintainer will review and merge.

## Code style

- **Go**: standard `gofmt`. Prefer `errors.New` and `fmt.Errorf` with `%w` wrapping. Use `slog` for logging — never `log.Printf`. Keep packages small and cohesive.
- **YAML**: 2-space indentation, no trailing whitespace.
- **Markdown**: use ATX headings (`#`), keep line length reasonable but don't hard-wrap mid-paragraph.

## Reporting security issues

Please do **not** open public issues for security problems. See [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
