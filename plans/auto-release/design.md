# Auto-release with semantic-release for go-infrabin

## Skills loaded and used

| Skill | Used for |
|---|---|
| skill-loader | Skill routing and provenance record |
| learning-lookup | Advisory lookup in `Datadog/Learnings.md` (terms: release, renovate, automerge, tag) |
| feature-worktree | Worktree creation for this spec |
| write | Drafting and reviewing this spec |

## Context reviewed

- `renovate.json`: automerge already enabled for minor, patch, pin, and digest update types; Go module, GitHub Actions, and Go version grouping rules exist.
- `.github/workflows/release.yml`: tag-push workflow that builds and pushes Docker images to `docker.io/maruina/go-infrabin` and `ghcr.io/maruina/go-infrabin`, then runs GoReleaser for binaries.
- `.github/workflows/release-drafter.yaml` and `.github/release-drafter.yml`: draft release notes on `master` pushes with label-based version resolution; nothing publishes the drafts.
- `.github/workflows/release-chart.yml` plus `.github/workflows/lint-chart.yml`: chart-releaser packages `charts/go-infrabin` on every `master` push and maintains a classic Helm repository on the `gh-pages` branch.
- `charts/go-infrabin/Chart.yaml`: chart `version: 2.0.2` (independent numbering), `appVersion: v3.8.0` (already drifted behind the `v3.8.2` tag).
- `README.md`: links to a broken path (`./chart/go-infrabin/README.md`, the real path is `charts/`).
- `charts/go-infrabin/README.md`: instructs `helm repo add https://maruina.github.io/go-infrabin` (gh-pages).
- Branch protection on `master`, read live via `gh api repos/maruina/go-infrabin/branches/master/protection` and changed during this design session (see Changes already applied).
- Tag history: all commits between `v3.8.1` and `v3.8.2` were renovate PRs; tags were cut by hand.
- Web and GitHub search found no maintained open-source tool that bumps versions from PR authorship (bot versus human). The maintained ecosystem (semantic-release, release-please, go-semantic-release) derives versions from conventional commit types instead.

## Problem

go-infrabin has no automated release path. Tags are cut by hand, release-drafter drafts notes nobody publishes, and renovate PRs accumulated without releases because branch protection required one approving review, which the renovate[bot] app cannot self-satisfy.

The goal: renovate PRs auto-merge when CI passes, and every merge to `master` that warrants a release produces a tag, GitHub release notes, Docker images, GoReleaser binaries, and a Helm chart automatically.

## Goal

semantic-release runs on every push to `master`. It analyzes commits since the last tag, computes the next version from commit types, pushes the tag, and publishes the GitHub release notes. In the same workflow job, GoReleaser builds and publishes Docker images and binaries for the new tag, and a Helm push publishes the chart as an OCI artifact to ghcr.io.

## Non-goals

- Daily batching of releases. Releases happen per merge; a burst of merges batches naturally because semantic-release releases whatever accumulated since the last tag in one run.
- Author-based (bot versus human) bump classification. Commit-type-based rules replace it.
- Chart `appVersion` automation beyond the release-time sync. The chart version and app version become the same tag.
- Keeping the gh-pages classic Helm repository.

## Known facts and assumptions

Verified facts:

- Branch protection on `master` now has required status checks `["Build"]`, `required_approving_review_count: 0`, `allow_auto_merge: true`, `enforce_admins: false`, and required linear history. Renovate native auto-merge (`platformAutomerge`) can complete for renovate PRs when `Build` passes.
- GitHub skips workflow runs triggered by events created with the default `GITHUB_TOKEN`. A tag pushed by the semantic-release step therefore does not trigger the existing `on: push tags` `release.yml`. semantic-release and GoReleaser must run in the same job.
- Renovate with `config:recommended` writes plain commit subjects ("Update module google.golang.org/grpc to v1.82.1") because the repository does not use conventional commits. semantic-release would ignore these commits and never release. Enabling `"semanticCommits": "enabled"` makes renovate emit `chore(deps): ...` subjects, which a `releaseRules` entry maps to patch.
- GitHub restricts `bypass_pull_request_allowances` to organization repositories, so a targeted bot exemption was impossible on this personal repository; the approval requirement is now 0.
- Collaborators with push access: `maruina` (owner) and `Smirl`.

Assumptions:

- The `WIP` required check was removed from the gate because it only reported on PR titles starting with "WIP" and never applied to renovate PRs. It still runs on PRs as a signal.
- The chart's only consumer is the maintainer installing by hand; nobody depends on the gh-pages Helm repository URL.
- semantic-release runs via the `cycjimmy/semantic-release-action` action or `npx semantic-release` with Node 22; GoReleaser steps in `release.yml` (protoc generation, buildx, logins) carry over unchanged.

## Design overview

New `.github/workflows/release.yml` (replacing the tag-push workflow), triggered on push to `master`:

1. `actions/checkout` with full history and tags (semantic-release needs all commits since the last tag).
2. Existing setup steps carried over: Go, QEMU, buildx, protoc, `make tools`, `make protoc`, Docker Hub and ghcr logins.
3. `cycjimmy/semantic-release-action` with a `.releaserc.json` configured for the `master` branch. Plugins: `@semantic-release/commit-analyzer` with `releaseRules`, `@semantic-release/release-notes-generator`, and `@semantic-release/github` (creates the tag and release with notes). No publish plugins; GoReleaser and Helm push own the artifacts.
4. Conditional on the action reporting a new version (`cycjimmy/semantic-release-action` outputs `new_release_published` / `new_release_version`): run GoReleaser (`goreleaser release --clean`), then sync `Chart.yaml` (`version` and `appVersion` set to the released tag with `yq`), `helm package charts/go-infrabin`, and `helm push` to `oci://ghcr.io/maruina/go-infrabin`. The workflow needs `packages: write` and `contents: write` permissions.

`.releaserc.json`:

- `branches: ["master"]` (the default is `main`).
- `releaseRules`: any `chore` commit releases as patch. With renovate emitting `chore(deps): ...`, dependency-only merge windows produce patch bumps. `feat` still produces minor and `feat!:` or a `BREAKING CHANGE` footer still produces major.
- No changelog file generation (`@semantic-release/changelog` not used); release notes go to the GitHub release, and GoReleaser's changelog stays disabled in `.goreleaser.yml`.

`renovate.json` changes:

- Add `"semanticCommits": "enabled"`.
- Add an automerge rule for `major` updates so all renovate PRs merge when `Build` passes. This is an accepted risk: major Go-version and breaking-dependency PRs merge unreviewed; CI is the only gate.

Deletions: `.github/workflows/release-drafter.yaml`, `.github/release-drafter.yml`, `.github/workflows/release-chart.yml`. semantic-release replaces release-drafter; OCI push replaces chart-releaser. `gh-pages` deletion is a follow-up outside this change.

Documentation: `README.md` gains an Installation section that points to the real chart README path and gives the OCI install command (`helm install go-infrabin oci://ghcr.io/maruina/go-infrabin --version <tag>`). `charts/go-infrabin/README.md` replaces the `helm repo add` instructions with the OCI reference.

## Smallest user-feedback slice

One PR containing: `.releaserc.json`, the rewritten `release.yml`, `renovate.json` changes, README and chart README updates, and the three workflow deletions.

- What the user sees: after merge, the first `feat:`/`fix:`-prefixed squash merge produces a real tag, GitHub release notes, images on Docker Hub and ghcr, binaries via GoReleaser, and a chart on ghcr.io.
- What the team learns: whether the `chore → patch` rule classifies renovate commits correctly, whether the token-chained GoReleaser step picks up the tag, and whether renovate native auto-merge completes end to end.
- Why no smaller slice: the semantic-release → tag → GoReleaser → OCI chart chain only proves itself when run end to end against real GitHub events.

## Success criteria and validation

1. A `fix:` commit merged to `master` produces a patch tag, a GitHub release with notes, `docker.io` and `ghcr.io` images, GoReleaser binaries, and an OCI chart within one workflow run.
2. A renovate-only merge window produces a patch release; a `feat:` merge produces a minor release; a `feat!:` merge produces a major release.
3. renovate PRs merge themselves when `Build` passes, including major updates.
4. No release-drafter or chart-releaser workflows remain, and `origin/gh-pages` stops updating.
5. Before the first real release, run semantic-release with `--dry-run` in the worktree (via `cycjimmy/semantic-release-action` `dry_run: true` or locally with `GITHUB_TOKEN`) and confirm the computed version and notes against expectation.

## Changes already applied outside the repository

Branch protection on `maruina/go-infrabin` `master` was changed during the design session:

- Required status checks reduced from `["Build", "WIP", "coverage/coveralls"]` to `["Build"]`. `coverage/coveralls` and `WIP` were third-party checks that could stall auto-merge without gating correctness; `Build` runs the tests and is the "CI passing" gate.
- `required_approving_review_count` reduced from `1` to `0` because GitHub does not support per-actor bypass allowances on personal repositories, and renovate[bot] cannot self-approve. Accepted cost: `Smirl`'s PRs no longer require an approval. The alternative (Mend's `renovate-approve` helper app) would keep the rule at the cost of a third-party app dependency.
- All other protection settings were preserved exactly: `enforce_admins: false`, required linear history, force pushes and deletions disallowed.

## Risks and mitigations

- **Unconventional commit titles produce no release.** semantic-release fails closed: commits without a recognized type release nothing. Mitigation: a `feat:` or `fix:` prefix on squash merge titles, and verification by checking the release existence after merges. The dry-run validates the analyzer rules before the first real release.
- **A major dependency update merges unreviewed.** Accepted per the maintainer's instruction that all passing-CI renovate PRs auto-merge. The daily pipeline (`Build` running tests and Docker builds) remains the gate. Revisit trigger: the first major update that breaks `Build` after merge and forces a revert.
- **GITHUB_TOKEN does not trigger the tag-push workflow.** Eliminated by chaining GoReleaser in the same job rather than relying on a second workflow.
- **Chart README and README drift.** Both updated in the same PR; the install command now references the OCI artifact whose version equals the app tag.
- **First release version jump.** semantic-release continues from the latest existing tag (`v3.8.2`), so the next version is `v3.8.x` or `v3.9.0`; no restart from `v1`.

## Operability

- Failure mode: a failed `release` workflow run in GitHub Actions. semantic-release errors on unknown commit analysis rather than publishing a wrong version, so failures are loud and observable.
- Rollback: revert the PR; manual tagging (`git tag && git push --tags`) continues to work because nothing in the new workflow prevents it. Existing tags, images, and chart versions remain published.
- gh-pages: becomes stale after the switch; delete the branch in a follow-up to avoid a half-updated Helm repository.

## Security and data handling

- The workflow uses the default `GITHUB_TOKEN` (no long-lived PAT) plus Docker Hub credentials already stored as repository secrets. `packages: write` grants chart publication to ghcr.io.
- No new secrets are introduced. Dropping `renovate-approve` avoided adding a third-party app with write access.

## Testing strategy

- Dry-run semantic-release against the real history to validate `releaseRules` classification for representative commits: `chore(deps): ...` (renovate), `feat: ...`, `fix: ...`, and a plain human title (expect no release).
- Validate the workflow YAML with `actionlint` if available, plus a first run on a merge that is expected to produce a patch release.
- Verify the OCI chart with `helm pull oci://ghcr.io/maruina/go-infrabin --version <tag>` and a local `helm template` render.

## Open questions

- None blocking. Residual: delete `origin/gh-pages` in a follow-up; verify renovate auto-merge behavior on the first major PR (GitHub native auto-merge versus renovate self-merge) and record the observed behavior.
