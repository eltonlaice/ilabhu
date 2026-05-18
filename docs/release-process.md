# Release process

How ilabhu cuts a tagged release end-to-end. Two CI workflows do most of the work; the human steps fit on a Post-it.

## Mental model

Three things have to line up:

1. A **tag** on `main` matching `v*` (the source of truth for "this is what shipped").
2. A **GitHub Release** with notes, attached to that tag.
3. **Multi-arch images** at `ghcr.io/eltonlaice/{ilabhud,ilabhu-web}:<tag>` plus a verifiable build attestation.

The pieces are wired up so each step triggers the next. The only deliberate human checkpoint is "cut the tag".

## How PRs build the next release

Every PR that lands on `main` triggers [release-drafter](https://github.com/release-drafter/release-drafter) (see [`.github/release-drafter.yml`](../.github/release-drafter.yml)). It:

1. **Auto-labels the PR** from its conventional-commit title prefix:
   - `feat:` → `feat` → "Added"
   - `fix:` → `fix` → "Fixed"
   - `docs:` → `docs` → "Documentation"
   - `refactor:`, `perf:` → "Changed"
   - `ci:`, `chore:`, `build:`, `test:`, `dependencies` → "CI / chores"
   - `security:` → "Security"
2. **Updates a rolling draft release** with the merged PRs grouped by category, each line `- $TITLE (#$NUMBER) @$AUTHOR`.
3. **Resolves the next version** from those labels:
   - `breaking` or `major` → major bump
   - `feat` → minor bump
   - everything else → patch bump

You can see the current draft at <https://github.com/eltonlaice/ilabhu/releases> (top of the list, marked Draft).

## Cutting a tag

When the draft looks ready:

```sh
git checkout main && git pull
NEXT=v0.1.2            # match the draft's resolved version
```

1. **Update the CHANGELOG.** Move everything from `[Unreleased]` into a new `[NEXT]` section with today's date, and update the compare links at the bottom of the file.
2. **Open a release PR.** Branch name `release/$NEXT`, single commit, conventional title:
   ```sh
   git checkout -b release/$NEXT
   git commit -am "docs: cut $NEXT"
   gh pr create --title "docs: cut $NEXT" --body "$(awk -v t="$NEXT" '/^## /{p=0} $0 ~ ("\\[" substr(t,2) "\\]") {p=1} p' CHANGELOG.md)"
   ```
3. **Wait for required checks** (`Build, vet, test`, `golangci-lint`, `Analyze (go)`, `Build, lint`, `govulncheck`, `Smoke`, `Compose smoke`, `Site lint`). Auto-merge handles the rest.
4. **Create the tagged release**:
   ```sh
   git checkout main && git pull
   notes=$(awk -v t="$NEXT" '/^## /{p=0} $0 ~ ("\\[" substr(t,2) "\\]") {p=1; next} p' CHANGELOG.md)
   gh release create "$NEXT" \
     --target main \
     --title "$NEXT — <one-line summary>" \
     --notes "$notes"
   ```

The release-drafter draft is automatically reset for the next version after this lands.

## What happens after the tag lands

Pushing `v$X.$Y.$Z` triggers [`.github/workflows/release-images.yml`](../.github/workflows/release-images.yml):

1. Builds `control-plane/Dockerfile` and `web/Dockerfile` for `linux/amd64` and `linux/arm64` via Buildx + QEMU.
2. Pushes to GHCR with the tag plus semver aliases (`v0.1.2`, `0.1.2`, `0.1`, and `latest` on non-prerelease tags).
3. Generates SBOM + provenance attestations via `docker/build-push-action` and pushes them to the registry alongside the images.
4. Calls `actions/attest-build-provenance` so users can verify with:
   ```sh
   gh attestation verify oci://ghcr.io/eltonlaice/ilabhud:v$X.$Y.$Z --owner eltonlaice
   ```

Total wall time is ~5–7 minutes per architecture, sequential. The two images run in parallel via the matrix.

## Backfilling a tag (e.g. images failed)

If the publish workflow fails on a tag you've already created, fix the underlying bug on `main` (don't re-tag) and dispatch the workflow against the tag:

```sh
gh workflow run release-images.yml --ref main -f tag=v$X.$Y.$Z
```

If the bug requires source changes that v$X.$Y.$Z does not include, cut a patch (v$X.$Y.$Z+1) — never force-update a tag that's been announced.

> v0.1.0 → v0.1.1 was this exact path: v0.1.0 shipped before the publish workflow existed; the v0.1.0 source tree also had a `web/public/` checkout-from-empty bug that broke the web image. We fixed the bug on `main`, cut v0.1.1, and let the publish workflow handle it from there.

## Verifying a release

Anyone, including users:

```sh
# Verify the image's provenance
gh attestation verify oci://ghcr.io/eltonlaice/ilabhud:v$X.$Y.$Z --owner eltonlaice

# Inspect the multi-arch manifest
docker manifest inspect ghcr.io/eltonlaice/ilabhud:v$X.$Y.$Z

# Run the bundle with that exact version
ILABHU_IMAGE_TAG=v$X.$Y.$Z docker compose \
  -f deploy/docker-compose.yml up --no-build --pull always
```

## Versioning policy

ilabhu follows [SemVer](https://semver.org/spec/v2.0.0.html). Pre-1.0 we treat:

- **Patch (`0.0.x`)** — bug fixes, doc changes, CI / chore work, new tests, new docs, dependency bumps.
- **Minor (`0.x.0`)** — new features that don't break existing manifests or HTTP API contracts. New provider adapters, new validator kinds, new endpoints, new exam content.
- **Major (`x.0.0`)** — breaking changes to `exam.yaml` schema, the HTTP API, or the `make`/`docker compose` entry points. We will call these out explicitly in the CHANGELOG and bump major only once we're at 1.0.

Anything labelled `breaking` on a PR bumps major.
