<!--
Thanks for the PR. Keep this template structure so reviewers can scan it
quickly. Delete sections that don't apply.
-->

## What

<!-- One-paragraph summary of what this PR changes. -->

## Why

<!-- The motivation. Skip if the title already says it. -->

## Verified

<!--
Concrete evidence this works:
- `make check` clean
- `make smoke` green
- Manual flow exercised: ...
-->

## Out of scope

<!--
Things you considered but deliberately left out. Helps reviewers stop
asking for them.
-->

## Checklist

- [ ] Branch name follows convention (`feat/`, `fix/`, `docs/`, `ci/`, `chore/`, `test/`, `refactor/`)
- [ ] Commit messages are conventional and explain the *why* in the body
- [ ] `make check` passes locally
- [ ] If the PR adds significant code, it also adds tests (coverage gate at 45%)
- [ ] If the PR changes the manifest schema, it updates `docs/exam-schema.md` and `docs/exam-schema.json`
- [ ] If the PR adds a user-facing surface, it links to the relevant doc
