# Auto-release Implementation Plan

> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every merge to `master` that warrants a release automatically produces a tag, GitHub release notes, Docker images, GoReleaser binaries, and an OCI Helm chart, with renovate PRs auto-merging when `Build` passes.
**Smallest user-feedback slice:** One PR (`.releaserc.json`, rewritten `release.yml`, `renovate.json` edits, README updates, three workflow deletions); after merge, the first conventional squash merge produces a full release in one workflow run.
**Out of Scope:** Deleting `origin/gh-pages` (follow-up); daily release batching; author-based (bot versus human) bump classification; chart `appVersion` automation beyond release-time sync; keeping the gh-pages classic Helm repository.
**Architecture:** semantic-release runs on every push to `master`, computes the next version from commit types since the last tag, and publishes the tag plus release notes through `@semantic-release/github`. Conditional steps in the same job fetch the new tag, build and push Docker images with buildx, run GoReleaser for binaries, sync `Chart.yaml` with `yq`, and push the packaged chart as an OCI artifact to ghcr.io. Renovate emits `chore(deps):` commit titles and auto-merges when `Build` passes.
**Tech Stack:** semantic-release (`cycjimmy/semantic-release-action` v8), GitHub Actions, GoReleaser, docker buildx, Helm (OCI push), yq, Renovate.

---

## Skills loaded and used

| Skill | Source | Why loaded | How used |
|---|---|---|---|
| `skill-loader` | `prompt-required` | Required before planning | Routed skill selection; built provenance |
| `resolve-worktree` | `prompt-required` | Input design path in another worktree | Resolved and switched to `maruina-auto-release` |
| `feature-worktree` | `prompt-required` | Durable-plan worktree rule | Confirmed the worktree is correct and based on latest `master` |
| `learning-lookup` | `prompt-required` | Advisory lookup before feasibility decisions | Ran lookup (terms: semantic-release, renovate, automerge, goreleaser, helm, github-actions); three sections returned, all about `ddoghq` repository patterns; no material guidance applied; no sources skipped |
| `write` | `agent-selected` (plan prose) | Drafting and reviewing plan prose | Applied clarity and mechanism-tracing rules to this document |

No `.go`, `.sh`, or Terraform files change, and `skill-loader` defines no best-practice skill for GitHub Actions workflow YAML, so no language skill applies.

## Implementation Contract

### Components Affected

| Component | Files | Responsibility | Verification |
|---|---|---|---|
| Release analysis configuration | `.releaserc.json` | Define `master` as the release branch, the three plugins, and the `chore` → `patch` release rule | Local `semantic-release --dry-run` (Task 1) |
| Release workflow | `.github/workflows/release.yml` | Trigger on push to `master`; run semantic-release; conditionally chain tag fetch, image builds, GoReleaser, chart sync and OCI push | `actionlint` (Task 5) plus first post-merge run |
| Renovate configuration | `renovate.json` | Emit `chore(deps):` commit titles; auto-merge `major` updates | `npx renovate-config-validator` (Task 3) |
| Removed legacy workflows | `.github/workflows/release-drafter.yaml`, `.github/release-drafter.yml`, `.github/workflows/release-chart.yml` | No longer draft notes or publish to gh-pages | Files absent after Task 4; `gh-pages` stops updating (Task 7) |
| Documentation | `README.md`, `charts/go-infrabin/README.md` | Fix the broken chart README link; document OCI installation | Link resolves; commands match the pushed artifact (Task 6) |

Consumers of shared state: `charts/go-infrabin/Chart.yaml` is read by `ct.yaml`-driven `lint-chart.yml` (unchanged, still lints PR chart changes) and by the `helmlint`/`helm-docs` pre-commit hooks (unchanged); the release-time `yq` sync only rewrites `version` and `appVersion`, so no consumer needs changes. `.goreleaser.yml` keeps `changelog: disable: true` and carries over unchanged.

### Key Decisions

1. **Single-job chaining (semantic-release → tag → artifacts).** `@semantic-release/github` pushes the tag through the GitHub API, and GitHub skips workflows triggered by `GITHUB_TOKEN`-created events, so a tag-triggered workflow would never run; everything runs in the same job. Rejected alternative: a second `on: push tags` workflow — benefit is cleaner separation; rejected because it never triggers.
2. **`chore` → `patch` release rule.** Renovate with `semanticCommits: "enabled"` emits `chore(deps): ...`, so dependency-only merge windows release as patch. Rejected alternative: no rule — benefit is stricter control; rejected because renovate-only windows would then never release, which is the current manual bottleneck.
3. **Renovate `major` auto-merge.** Accepted risk per the maintainer: CI (`Build`) is the only gate. Revisit trigger: the first major update that breaks `Build` after merge and forces a revert.
4. **Explicit plugins array in `.releaserc.json` without `@semantic-release/npm`.** The repository has no `package.json`; the default npm plugin fails with `ENOPKG` even in dry-run. The action pre-installs the three configured plugins, so no `extra_plugins` input is needed. Rejected alternative: adding a stub `package.json` — benefit is tooling compatibility; rejected as scope creep.
5. **Tag fetch before GoReleaser.** The workflow checkout does not contain the tag semantic-release created mid-job, and GoReleaser requires the tag on the checked-out commit, so a `git fetch origin <tag>` step runs after the semantic-release step.
6. **OCI chart artifact location.** `helm push` targets `oci://ghcr.io/maruina`, so the artifact lands at `ghcr.io/maruina/go-infrabin` (the chart name supplies the final path segment), where the Docker image already lives as a separate artifact type. Chart `version` has no `v` prefix; `appVersion` carries `v` (matches the existing `appVersion: v3.8.0`).
7. **Neutral task commit types.** Because `chore` now releases as patch, all PR commits use `ci:` or `docs:` types, which release nothing; the first release comes from the squash-merge title the maintainer chooses (a `feat:` title gives `v3.9.0`).

### Implementation Constraints

- Required linear history on `master` means squash or rebase merges only: exactly one commit title per PR decides the release. The `Build` required check, `required_approving_review_count: 0`, and `allow_auto_merge: true` were set during design (outside this repository).
- Workflow permissions: `contents: write` (tag, release, GoReleaser) and `packages: write` (ghcr.io chart push). No long-lived PAT; the default `GITHUB_TOKEN` plus the existing `DOCKER_USERNAME`/`DOCKER_PASSWORD` secrets.
- Carry over the existing action pins (`actions/checkout@v7`, `docker/*`, `arduino/setup-protoc@v3.0.0`, `goreleaser/goreleaser-action@v7.3.0`, `azure/setup-helm@v5.1.0`); pin `cycjimmy/semantic-release-action@v8`.
- Stop condition: do not merge until Task 1's dry-run validates the release rules. If the dry-run cannot run (npm or network unavailable), report the blocker instead of merging unvalidated.
- Performance: the workflow runs on every `master` push; pushes without a releasable commit pay only the semantic-release step and skip the conditional block. No caching beyond what `checkout` provides.

### Security and Observability Requirements

- No new secrets; no third-party apps with write access (the `renovate-approve` helper was rejected during design).
- Failure visibility: semantic-release fails closed on unrecognized analysis and errors loudly; a failed `Release` run in GitHub Actions is the alert. There is no additional telemetry to add for a personal repository.

### Failure Modes to Handle

- **`ENOPKG` from the npm plugin** — prevented by the explicit plugins array (Decision 4).
- **GoReleaser fails with no tag** — prevented by the tag fetch step (Decision 5); step order is a first-run proof, not something `actionlint` catches.
- **Chart push rejected by ghcr.io** — prevented by `helm registry login` with `GITHUB_TOKEN` before `helm push`; requires `packages: write`.
- **Repeated chart version on workflow re-run** — impossible: a re-run publishes no new version, `new_release_published` is `"false"`, and the conditional block is skipped.
- **Unconventional merge title produces no release silently** — fail-closed by design; mitigated by the `feat:`/`fix:` title discipline, the Task 1 dry-run, and the post-merge check that a release exists (Requirement 1).

### Rollout and Rollback

Owner: Matteo Ruina (maintainer). Rollout is the single merged PR; the first merge with a conventional title produces the first automated release, continuing from `v3.8.2` (never restarting at `v1`). Rollback: revert the PR — a revert merge titled `Revert ...` produces no release, manual `git tag && git push` continues to work, and already-published tags, images, and chart versions stay valid. `gh-pages` goes stale until the follow-up branch deletion.

### Test Strategy

- **Analysis/rules seam:** local `semantic-release --dry-run` against the real git history with the release rules config; deterministic, no publishing. Limitation: the dry-run validates commit classification only; actual tag push, release creation, image publication, and chart push are GitHub-dependent behaviors proven by the first post-merge run.
- **Static checks:** `actionlint` for `release.yml`; `renovate-config-validator` for `renovate.json`.
- **Real end-to-end (post-merge):** one expected patch release observed through the Actions run, release page, `docker buildx imagetools inspect`, GoReleaser artifacts, and `helm pull`/`helm template` of the OCI chart; renovate auto-merge observed on the next renovate PR, including the first major.
- No Go code changes, so no Go test seam is needed.

## Acceptance criteria

### Requirement: A conventional merge produces a full release
The system SHALL, when a squash merge with a conventional title (`fix:`, `feat:`, `feat!:`) lands on `master`, produce within one `Release` workflow run: a git tag, a GitHub release with generated notes, `docker.io/maruina/go-infrabin` and `ghcr.io/maruina/go-infrabin` images, GoReleaser binaries, and an OCI chart at `ghcr.io/maruina/go-infrabin`.

#### Scenario: patch release from a fix merge
- GIVEN `master` at tag `v3.8.2` and the merged workflow
- WHEN a squash merge titled `fix: ...` lands on `master`
- THEN the `Release` run publishes tag `v3.8.3`, release notes, both images, GoReleaser binaries, and the OCI chart `3.8.3`

#### Scenario: non-releasable merge produces nothing
- GIVEN `master` at tag `v3.8.2`
- WHEN a squash merge titled `docs: ...` lands on `master`
- THEN no tag, release, image, binary, or chart is published, and the conditional steps are skipped

### Requirement: Commit types map to release levels
The system SHALL release a patch for `chore` commits (including renovate's `chore(deps):`), a minor for `feat:`, and a major for `feat!:` or a `BREAKING CHANGE` footer.

#### Scenario: renovate-only window releases a patch
- GIVEN renovate emits `chore(deps): ...` commit titles
- WHEN a renovate PR auto-merges to `master`
- THEN the `Release` run computes a patch version from `v3.8.2`

#### Scenario: plain title releases nothing
- GIVEN a squash merge titled `Update module google.golang.org/grpc to v1.82.1` (no conventional prefix)
- WHEN the `Release` run analyzes commits since `v3.8.2`
- THEN no release is published

### Requirement: Renovate PRs auto-merge
The system SHALL auto-merge renovate PRs of `minor`, `patch`, `pin`, `digest`, and `major` update types when the `Build` check passes, with no approving review.

#### Scenario: major update merges unreviewed
- GIVEN branch protection requires only `Build` and allows auto-merge
- WHEN a renovate major-update PR passes `Build`
- THEN the PR merges without an approving review

#### Scenario: red build blocks merge
- GIVEN a renovate PR whose `Build` check fails
- WHEN auto-merge is requested
- THEN the PR does not merge

### Requirement: Legacy release mechanisms are removed
The system SHALL no longer run release-drafter or chart-releaser on `master` pushes, and `origin/gh-pages` SHALL stop updating.

#### Scenario: workflow files absent
- GIVEN the merged PR
- WHEN listing `.github/workflows/`
- THEN `release-drafter.yaml`, `release-chart.yml`, and the root `.github/release-drafter.yml` are gone, and `build.yml`, `code-quality.yaml`, `lint.yaml`, `lint-chart.yml`, `release.yml` remain

### Requirement: Release rules are validated before the first real release
The system SHALL validate the `chore` → `patch` rule and fail-closed behavior with a semantic-release dry-run before merging.

#### Scenario: dry-run classification
- GIVEN the real repository history since `v3.8.2`
- WHEN `semantic-release --dry-run` runs with the new `.releaserc.json` and a `chore(deps):` probe commit
- THEN the computed next version is `3.8.3`, and without the probe commit it reports no release

## Tasks

### Task 1: Add `.releaserc.json` and validate release rules with a local dry-run
**Delivers:** A validated release configuration that maps `chore(deps):` to a patch bump and ignores plain commit titles.
**Blocked by:** None
**Traces to:** "Commit types map to release levels"; "Release rules are validated before the first real release"
**Files:** `.releaserc.json` (new)

- [ ] Install the tooling without touching repository files: `npm install --no-save semantic-release @semantic-release/commit-analyzer @semantic-release/release-notes-generator @semantic-release/github`. `deliberate:` latest plugin versions may differ from the action's bundled versions; rule-level validation is unaffected.
- [ ] Create `.releaserc.json` with `{"branches": ["master"], "plugins": ["@semantic-release/commit-analyzer", "@semantic-release/release-notes-generator", "@semantic-release/github"]}` — first without release rules.
- [ ] Run the focused failing check: `git commit --allow-empty -m "chore(deps): dry-run probe" && GITHUB_TOKEN=$(gh auth token) npx semantic-release --dry-run --branches "$(git branch --show-current)"`; expect `There are no relevant changes, so no new version is released` (default angular rules do not release `chore`).
- [ ] Add the smallest passing change: `"releaseRules": [{"type": "chore", "release": "patch"}]` inside the `@semantic-release/commit-analyzer` entry in `.releaserc.json`.
- [ ] Rerun the dry-run command; expect `The next release version is 3.8.3` and tag `v3.8.3`.
- [ ] Remove the probe commit: `git reset --hard HEAD~1` (keep `node_modules` until Task 7 cleanup).
- [ ] Run the dry-run again without the probe commit; expect no release (real history contains only `docs:`/`ci:` commits since `v3.8.2`).
- [ ] Commit with `ci: add semantic-release configuration`.

### Task 2: Rewrite `release.yml` around semantic-release
**Delivers:** A `push` to `master` workflow that publishes tag and release, then conditionally builds images, binaries, and the OCI chart in one job.
**Blocked by:** Task 1
**Traces to:** "A conventional merge produces a full release"
**Files:** `.github/workflows/release.yml`

- [ ] Trigger on `push` to `branches: [master]`; set `permissions: {contents: write, packages: write}`. Delete the tag trigger, the `olegtarasov/get-tag` step, and all tag-based tag interpolations.
- [ ] Carry over, in order: `actions/checkout@v7` with `fetch-depth: 0` (full history and tags for semantic-release), `actions/setup-go@v7`, `docker/setup-qemu-action@v4`, `docker/setup-buildx-action@v4`, `arduino/setup-protoc@v3.0.0`, `make tools`, `make protoc`.
- [ ] Add the semantic-release step `cycjimmy/semantic-release-action@v8` with `id: semantic`, env `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`; no `extra_plugins` or `semantic_version` input (Decision 4).
- [ ] Gate every following step with `if: steps.semantic.outputs.new_release_published == 'true'` (string comparison — verified against the action README).
- [ ] Fetch the new tag so GoReleaser sees it: `git fetch origin "refs/tags/v${{ steps.semantic.outputs.new_release_version }}:refs/tags/v${{ steps.semantic.outputs.new_release_version }}"` (Decision 5).
- [ ] Carry over the Docker Hub and ghcr.io logins; build and push with buildx (`platforms: linux/amd64,linux/arm64`) tagging `docker.io/maruina/go-infrabin:v${{ steps.semantic.outputs.new_release_version }}` and the ghcr.io equivalent (`v` prefix — the output is `3.8.3`, not `v3.8.3`); keep the `docker buildx imagetools inspect` checks.
- [ ] Run GoReleaser (`goreleaser/goreleaser-action@v7.3.0`, `version: '~> v2'`, `args: release --clean`, env `GITHUB_TOKEN`) after the tag fetch.
- [ ] Add chart publication: install `yq` (`curl -sL -o yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64`, `chmod +x`, move to `/usr/local/bin`); run `yq -i '.version = "${{ steps.semantic.outputs.new_release_version }}" | .appVersion = "v${{ steps.semantic.outputs.new_release_version }}"' charts/go-infrabin/Chart.yaml`; add `azure/setup-helm@v5.1.0`; run `helm registry login ghcr.io -u ${{ github.repository_owner }} --password ${{ secrets.GITHUB_TOKEN }}`; run `helm package charts/go-infrabin`; run `helm push go-infrabin-${{ steps.semantic.outputs.new_release_version }}.tgz oci://ghcr.io/maruina` (Decision 6).
- [ ] Commit with `ci: chain GoReleaser and OCI chart push after semantic-release`.

### Task 3: Configure renovate semantic commits and major automerge
**Delivers:** Renovate emits `chore(deps):` titles and auto-merges all update types when `Build` passes.
**Blocked by:** None
**Traces to:** "Commit types map to release levels"; "Renovate PRs auto-merge"
**Files:** `renovate.json`

- [ ] Add `"semanticCommits": "enabled"` at the top level.
- [ ] Add a package rule `{"matchUpdateTypes": ["major"], "automerge": true}` with a description recording the accepted risk (Decision 3).
- [ ] Run `npx --yes --package renovate renovate-config-validator` from the repository root; expect the file validated with no errors and exit code 0.
- [ ] Commit with `ci: enable renovate semantic commits and major automerge`.

### Task 4: Remove legacy release workflows and update documentation
**Delivers:** No release-drafter or chart-releaser mechanisms remain; both READMEs describe OCI installation.
**Blocked by:** None
**Traces to:** "Legacy release mechanisms are removed"
**Files:** `.github/workflows/release-drafter.yaml`, `.github/release-drafter.yml`, `.github/workflows/release-chart.yml` (delete); `README.md`, `charts/go-infrabin/README.md` (edit)

- [ ] `git rm .github/workflows/release-drafter.yaml .github/release-drafter.yml .github/workflows/release-chart.yml`; keep `lint-chart.yml`, `ct.yaml`, and the pre-commit chart hooks untouched.
- [ ] In `README.md`, fix the chart README link from `./chart/go-infrabin/README.md` to `./charts/go-infrabin/README.md` and add an Installation section pointing at `charts/go-infrabin/README.md` with the OCI command `helm install go-infrabin oci://ghcr.io/maruina/go-infrabin --version <chart-version>` (no `v` prefix).
- [ ] In `charts/go-infrabin/README.md`, replace the `helm repo add` block with the OCI install command, and replace the "Adding a new version" instructions (`helm package` + `helm repo index`) with a note that the release workflow packages and pushes the chart on every release. Leave the generated Values table and the KIND testing section untouched.
- [ ] Commit with `ci: remove release-drafter and chart-releaser workflows` and `docs: document OCI chart installation`.

### Task 5: Validate the workflow YAML
**Delivers:** Static proof that the rewritten workflow parses and its expressions are well-formed.
**Blocked by:** Task 2
**Traces to:** "A conventional merge produces a full release"
**Files:** None (validation only)

- [ ] `go install github.com/rhysd/actionlint/cmd/actionlint@latest && "$(go env GOPATH)/bin/actionlint"`; expect no errors across `.github/workflows/`.

### Task 6: Check documentation surfaces
**Delivers:** Proof that user-facing docs match the shipped mechanism and no repository guidance file is stale.
**Blocked by:** Task 4
**Traces to:** "Legacy release mechanisms are removed"
**Files:** None (verification only)

- [ ] Confirm `README.md` contains no reference to `maruina.github.io/go-infrabin`, `release-drafter`, or `chart-releaser`.
- [ ] Confirm the repository has no `AGENTS.md`, `CLAUDE.md`, or similar guidance file; record here that none exists to update (the READMEs are the user-facing documentation surface).

### Task 7: Final verification and cleanup
**Delivers:** Cleanup of dry-run tooling and the named feature-level checks that only the completed feature satisfies.
**Blocked by:** Tasks 1–6
**Traces to:** All requirements

- [ ] Remove dry-run tooling: `rm -rf node_modules package-lock.json`; confirm `git status` shows no untracked tooling files.
- [ ] The following feature-level acceptance criteria only the merged feature can satisfy, observed post-merge (Task 2's first real run and subsequent merges):
  - A `fix:` squash merge produces tag, release notes, both images, GoReleaser binaries, and the OCI chart in one run (Requirement 1).
  - A renovate-only merge window produces a patch release; `feat:` produces minor; `feat!:` produces major (Requirement 2).
  - A renovate PR, including a major update, merges itself when `Build` passes; record the observed merge mechanism (GitHub native auto-merge versus renovate self-merge) in the PR (Requirement 3).
  - `origin/gh-pages` receives no new commits after the switch; its deletion is the recorded follow-up (Requirement 4).
  - `helm pull oci://ghcr.io/maruina/go-infrabin --version <chart-version>` and `helm template` render succeeds (Requirement 1).
