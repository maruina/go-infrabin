# Auto-release Implementation Plan

> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every merge to `master` that warrants a release automatically produces a tag, GitHub release notes, Docker images, GoReleaser binaries, and an OCI Helm chart, with renovate PRs auto-merging when `Build` passes.
**Smallest user-feedback slice:** One PR (`.releaserc.json`, rewritten `release.yml`, `renovate.json` edits, README updates, three workflow deletions); after merge, the first conventional squash merge produces a full release in one workflow run.
**Out of Scope:** Deleting `origin/gh-pages` (follow-up); daily release batching; author-based (bot versus human) bump classification; chart `appVersion` automation beyond release-time sync; keeping the gh-pages classic Helm repository.
**Architecture:** semantic-release runs on every push to `master`, computes the next version from commit types since the last tag, and publishes the tag plus release notes through `@semantic-release/github`. Conditional steps in the same job build and push Docker images with buildx, run GoReleaser for binaries, sync `Chart.yaml` with `yq`, and push the packaged chart as an OCI artifact to ghcr.io. Renovate emits `chore(deps):` commit titles and auto-merges when `Build` passes.
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

### Execution

| Skill | Source | Why loaded | How used |
|---|---|---|---|
| `skill-loader` | `prompt-required` | Required before editing | Routed skill selection; no language or domain skill trigger matches the affected files |
| `resolve-worktree` | `prompt-required` | Plan path in another worktree | Resolved and switched to `maruina-auto-release` |
| `feature-worktree` | `prompt-required` | Worktree policy before writes | Confirmed the worktree is the correct feature worktree, clean, based on `master` |
| `write` | `agent-selected` | README prose edits in Task 4 | Applied clarity and mechanism rules to the README edits |

Execution notes: no execution skill was needed beyond the above. No unrelated code was touched. The Task 1 `git reset --hard HEAD~1` also discarded this uncommitted ledger section once; re-added after Task 1.

**Current task:** Tasks 1–7 local work complete; Task 7's post-merge acceptance items remain open and are gated on the merge.

## Implementation Contract

### Components Affected

| Component | Files | Responsibility | Verification |
|---|---|---|---|
| Release analysis configuration | `.releaserc.json` | Define `master` as the release branch, the three plugins, and the `chore` → `patch` release rule | Local `semantic-release --dry-run` (Task 1) |
| Release workflow | `.github/workflows/release.yml` | Trigger on push to `master`; run semantic-release; conditionally chain image builds, GoReleaser, chart sync and OCI push | `actionlint` (Task 5) plus first post-merge run |
| Renovate configuration | `renovate.json` | Emit `chore(deps):` commit titles; auto-merge `major` updates | `npx renovate-config-validator` (Task 3) |
| Removed legacy workflows | `.github/workflows/release-drafter.yaml`, `.github/release-drafter.yml`, `.github/workflows/release-chart.yml` | No longer draft notes or publish to gh-pages | Files absent after Task 4; `gh-pages` stops updating (Task 7) |
| Documentation | `README.md`, `charts/go-infrabin/README.md` | Fix the broken chart README link; document OCI installation | Link resolves; commands match the pushed artifact (Task 6) |

Consumers of shared state: `charts/go-infrabin/Chart.yaml` is read by `ct.yaml`-driven `lint-chart.yml` (unchanged, still lints PR chart changes) and by the `helmlint`/`helm-docs` pre-commit hooks (unchanged); the release-time `yq` sync only rewrites `version` and `appVersion`, so no consumer needs changes. `.goreleaser.yml` keeps `changelog: disable: true` and carries over unchanged.

### Key Decisions

1. **Single-job chaining (semantic-release → tag → artifacts).** `@semantic-release/github` pushes the tag through the GitHub API, and GitHub skips workflows triggered by `GITHUB_TOKEN`-created events, so a tag-triggered workflow would never run; everything runs in the same job. Rejected alternative: a second `on: push tags` workflow — benefit is cleaner separation; rejected because it never triggers.
2. **`chore` → `patch` release rule.** Renovate with `semanticCommits: "enabled"` emits `chore(deps): ...`, so dependency-only merge windows release as patch. Rejected alternative: no rule — benefit is stricter control; rejected because renovate-only windows would then never release, which is the current manual bottleneck.
3. **Renovate `major` auto-merge.** Accepted risk per the maintainer: CI (`Build`) is the only gate. Revisit trigger: the first major update that breaks `Build` after merge and forces a revert.
4. **Explicit plugins array in `.releaserc.json` without `@semantic-release/npm`.** The repository has no `package.json`; the default npm plugin fails with `ENOPKG` even in dry-run. The action pre-installs the three configured plugins, so no `extra_plugins` input is needed. Rejected alternative: adding a stub `package.json` — benefit is tooling compatibility; rejected as scope creep.
5. **No tag-fetch step.** semantic-release tags the checked-out HEAD locally (`git tag` runs in the job workspace) and pushes the tag, so GoReleaser sees the tag on the current commit without an extra fetch. Rejected alternative: a `git fetch origin <tag>` step — benefit is resilience if semantic-release stops tagging locally; rejected because the tag is already present and fetching an identical ref is a no-op.
6. **OCI chart artifact location.** `helm push` targets `oci://ghcr.io/maruina`, so the artifact lands at `ghcr.io/maruina/go-infrabin` (the chart name supplies the final path segment), where the Docker image already lives as a separate artifact type. Chart `version` has no `v` prefix; `appVersion` carries `v` (matches the existing `appVersion: v3.8.0`).
7. **Neutral task commit types.** Because `chore` now releases as patch, all PR commits use `ci:` or `docs:` types, which release nothing; the first release comes from the squash-merge title the maintainer chooses (a `feat:` title gives `v3.9.0`).
8. **Serialize Release runs.** Renovate auto-merge can merge several PRs in quick succession; a run that starts before the previous run's tag landed computes the same version and fails at `git push --tags`. A `concurrency` group queues runs instead, so each run sees the tags the previous run pushed.

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
- **GoReleaser fails with no tag** — impossible: semantic-release tags the checked-out HEAD locally before the conditional block runs (Decision 5). Step order is a first-run proof, not something `actionlint` catches.
- **Chart push rejected by ghcr.io** — prevented by `helm registry login` with `GITHUB_TOKEN` before `helm push`; requires `packages: write`.
- **Repeated chart version on workflow re-run** — impossible: a re-run publishes no new version, `new_release_published` is `"false"`, and the conditional block is skipped.
- **Unconventional merge title produces no release silently** — fail-closed by design; mitigated by the `feat:`/`fix:` title discipline, the Task 1 dry-run, and the post-merge check that a release exists (Requirement 1).

### Rollout and Rollback

Owner: Matteo Ruina (maintainer). Rollout is the single merged PR; the first merge with a conventional title produces the first automated release, continuing from `v3.8.2` (never restarting at `v1`). Rollback: revert the PR — a revert merge titled `Revert ...` produces no release, manual `git tag && git push` continues to work, and already-published tags, images, and chart versions stay valid. `gh-pages` goes stale until the follow-up branch deletion. Partial-failure recovery: if the run fails after semantic-release published the tag and release (for example a transient `helm push` failure), re-running the workflow repairs nothing because `new_release_published` is `"false"`; publish the missing artifacts by hand against the existing tag with `goreleaser release --clean` and `helm package`/`helm push`.

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
**Files:** `.releaserc.json` (new), `.gitignore` (edit)

- [x] Install the tooling without touching repository files: `npm install --no-save semantic-release @semantic-release/commit-analyzer @semantic-release/release-notes-generator @semantic-release/github`. `deliberate:` latest plugin versions may differ from the action's bundled versions; rule-level validation is unaffected.
- [x] Add `node_modules/` to `.gitignore` so the dry-run tooling never shows as untracked files; include the change in the Task 1 commit.
- [x] Create `.releaserc.json` with `{"branches": ["master"], "plugins": ["@semantic-release/commit-analyzer", "@semantic-release/release-notes-generator", "@semantic-release/github"]}` — first without release rules.
- [x] Run the focused failing check: `git commit --allow-empty -m "chore(deps): dry-run probe" && GITHUB_TOKEN=$(gh auth token) npx semantic-release --dry-run --branches "$(git branch --show-current)"`; expect `There are no relevant changes, so no new version is released` (default angular rules do not release `chore`). The `--branches` flag is load-bearing: the worktree branch is `maruina/auto-release`, not `master`, and semantic-release skips branches outside its configured list.
- [x] Add the smallest passing change: `"releaseRules": [{"type": "chore", "release": "patch"}]` inside the `@semantic-release/commit-analyzer` entry in `.releaserc.json`.
- [x] Rerun the dry-run command; expect `Published release 3.8.3 on default channel` and `Skip v3.8.3 tag creation in dry-run mode`.
- [x] Remove the probe commit: `git reset --hard HEAD~1` (keep `node_modules` until Task 7 cleanup).
- [x] Run the dry-run again without the probe commit; expect no release (the real history since `v3.8.2` contains only non-releasable commits: plain-titled renovate merges and `docs:` commits).
- [x] Commit with `ci: add semantic-release configuration`.

Task 1 deviations:
- The plan's `--branches` mechanism was incomplete: semantic-release's branch expansion matches configured branch names against `git ls-remote --heads`, so `maruina/auto-release` had to exist on the remote. Fix: pushed the feature branch to `origin` before the dry-run (needed for the PR anyway), then re-ran the plan's exact command. With `--branches master` alone the run still reports `triggered on the branch maruina/auto-release ... a new version won't be published`, confirming `--branches` alone cannot select the worktree branch.
- Plugin options use the array form `[@semantic-release/commit-analyzer, {"releaseRules": [...]}]`; the `{pluginName: options}` object form silently had no effect (`semantic-release/lib/plugins/utils.js` `parseConfig` does not support it). Discovered on the first green-check rerun, which still reported `no release`.
- `git reset --hard HEAD~1` discarded the unstaged `.gitignore` edit, so the first commit missed it; restored the edit and amended the commit. Final commit `8bee7cf` contains both files.
- After the probe reset, the local branch was behind the remote (the probe had been pushed), so semantic-release refused with `is behind the remote one`; cleaned the remote probe with `git push --force-with-lease`, then the final no-release check passed (7 commits analyzed, no release).

### Task 2: Rewrite `release.yml` around semantic-release
**Delivers:** A `push` to `master` workflow that publishes tag and release, then conditionally builds images, binaries, and the OCI chart in one job.
**Blocked by:** Task 1
**Traces to:** "A conventional merge produces a full release"
**Files:** `.github/workflows/release.yml`

- [x] Trigger on `push` to `branches: [master]`; set `permissions: {contents: write, packages: write}`. Delete the tag trigger, the `olegtarasov/get-tag` step, and all tag-based tag interpolations.
- [x] Carry over, in order: `actions/checkout@v7` with `fetch-depth: 0` (full history and tags for semantic-release), `actions/setup-go@v7`, `docker/setup-qemu-action@v4`, `docker/setup-buildx-action@v4`, `arduino/setup-protoc@v3.0.0`, `make tools`, `make protoc`.
- [x] Add the semantic-release step `cycjimmy/semantic-release-action@v8` with `id: semantic`, env `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`; no `extra_plugins` or `semantic_version` input (Decision 4).
- [x] Gate every following step with `if: steps.semantic.outputs.new_release_published == 'true'` (string comparison — verified against the action README).
- [x] Add `concurrency: {group: release, cancel-in-progress: false}` so merge bursts queue instead of racing the tag push (Decision 8).
- [x] Carry over the Docker Hub and ghcr.io logins; build and push with buildx (`platforms: linux/amd64,linux/arm64`) tagging `docker.io/maruina/go-infrabin:v${{ steps.semantic.outputs.new_release_version }}` and the ghcr.io equivalent (`v` prefix — the output is `3.8.3`, not `v3.8.3`); keep the `docker buildx imagetools inspect` checks.
- [x] Run GoReleaser (`goreleaser/goreleaser-action@v7.3.0`, `version: '~> v2'`, `args: release --clean`, env `GITHUB_TOKEN`) after the semantic-release step; the tag is already on the checked-out HEAD (Decision 5).
- [x] Add chart publication: install `yq` (`curl -sL -o yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64`, `chmod +x`, move to `/usr/local/bin`); run `yq -i '.version = "${{ steps.semantic.outputs.new_release_version }}" | .appVersion = "v${{ steps.semantic.outputs.new_release_version }}"' charts/go-infrabin/Chart.yaml`; add `azure/setup-helm@v5.1.0`; login with `echo "${{ secrets.GITHUB_TOKEN }}" | helm registry login ghcr.io -u ${{ github.repository_owner }} --password-stdin`; run `helm package charts/go-infrabin`; run `helm push go-infrabin-${{ steps.semantic.outputs.new_release_version }}.tgz oci://ghcr.io/maruina` (Decision 6).
- [x] Commit with `ci: chain GoReleaser and OCI chart push after semantic-release`.

### Task 3: Configure renovate semantic commits and major automerge
**Delivers:** Renovate emits `chore(deps):` titles and auto-merges all update types when `Build` passes.
**Blocked by:** None
**Traces to:** "Commit types map to release levels"; "Renovate PRs auto-merge"
**Files:** `renovate.json`

- [x] Add `"semanticCommits": "enabled"` at the top level.
- [x] Add a package rule `{"matchUpdateTypes": ["major"], "automerge": true}` with a description recording the accepted risk (Decision 3).
- [x] Run `npx --yes --package renovate renovate-config-validator` from the repository root; expect the file validated with no errors and exit code 0.
- [x] Commit with `ci: enable renovate semantic commits and major automerge`.

### Task 4: Remove legacy release workflows and update documentation
**Delivers:** No release-drafter or chart-releaser mechanisms remain; both READMEs describe OCI installation.
**Blocked by:** None
**Traces to:** "Legacy release mechanisms are removed"
**Files:** `.github/workflows/release-drafter.yaml`, `.github/release-drafter.yml`, `.github/workflows/release-chart.yml` (delete); `README.md`, `charts/go-infrabin/README.md` (edit)

- [x] `git rm .github/workflows/release-drafter.yaml .github/release-drafter.yml .github/workflows/release-chart.yml`; keep `lint-chart.yml`, `ct.yaml`, and the pre-commit chart hooks untouched.
- [x] Delete the stale release-drafter draft release so it cannot collide with the first `v3.8.3` release: list drafts with `gh api repos/maruina/go-infrabin/releases --jq '.[] | select(.draft) | "\(.id) \(.tag_name)"'`, then `gh api -X DELETE repos/maruina/go-infrabin/releases/<id>`; re-run the listing and expect no draft releases.
- [x] In `README.md`, fix the chart README link from `./chart/go-infrabin/README.md` to `./charts/go-infrabin/README.md` and add an Installation section pointing at `charts/go-infrabin/README.md` with the OCI command `helm install go-infrabin oci://ghcr.io/maruina/go-infrabin --version <chart-version>` (no `v` prefix).
- [x] In `charts/go-infrabin/README.md`, replace the `helm repo add` block with the OCI install command, and replace the "Adding a new version" instructions (`helm package` + `helm repo index`) with a note that the release workflow packages and pushes the chart on every release. Leave the generated Values table and the KIND testing section untouched.

Task 4 note: the chart README is generated from `charts/go-infrabin/README.md.gotmpl` by the `helm-docs` pre-commit hook, so the OCI changes went into the template (commit `b0877cf`) and the committed `README.md` was regenerated and matched; running `helm-docs` surfaced an unrelated stale `ingress.hosts` row in the Values table, which was reverted because the plan leaves the generated Values table untouched.
- [x] Commit with `ci: remove release-drafter and chart-releaser workflows` and `docs: document OCI chart installation`.

Task 4 note: the stale draft release id was `424649103` (`v3.8.3`); the delete succeeded and the re-run listing returned no draft releases.

### Task 5: Validate the workflow YAML
**Delivers:** Static proof that the rewritten workflow parses and its expressions are well-formed.
**Blocked by:** Task 2
**Traces to:** "A conventional merge produces a full release"
**Files:** None (validation only)

- [x] `go install github.com/rhysd/actionlint/cmd/actionlint@latest && "$(go env GOPATH)/bin/actionlint"`; expect no errors across `.github/workflows/`.

Task 5 note: `release.yml` is clean (`actionlint .github/workflows/release.yml` exits 0). The full-directory run reports pre-existing issues in `build.yml` (shellcheck SC2086/SC2046) and `lint-chart.yml` (deprecated `set-output`); both files are untouched by this feature and left unchanged.

### Task 6: Check documentation surfaces
**Delivers:** Proof that user-facing docs match the shipped mechanism and no repository guidance file is stale.
**Blocked by:** Task 4
**Traces to:** "Legacy release mechanisms are removed"
**Files:** None (verification only)

- [x] Confirm `README.md` contains no reference to `maruina.github.io/go-infrabin`, `release-drafter`, or `chart-releaser`.
- [x] Confirm the repository has no `AGENTS.md`, `CLAUDE.md`, or similar guidance file; record here that none exists to update (the READMEs are the user-facing documentation surface).

### Task 7: Final verification and cleanup
**Delivers:** Cleanup of dry-run tooling and the named feature-level checks that only the completed feature satisfies.
**Blocked by:** Tasks 1–6
**Traces to:** All requirements

- [x] Remove dry-run tooling: `rm -rf node_modules package-lock.json`; confirm `git status` shows no untracked tooling files.
- [ ] The following feature-level acceptance criteria only the merged feature can satisfy, observed post-merge (Task 2's first real run and subsequent merges). These remain the open items after merge:
  - A `fix:` squash merge produces tag, release notes, both images, GoReleaser binaries, and the OCI chart in one run (Requirement 1).
  - The release page shows exactly one entry for the first published version and no stale release-drafter drafts (Requirement 1).
  - A renovate-only merge window produces a patch release; `feat:` produces minor; `feat!:` produces major (Requirement 2).
  - A renovate PR, including a major update, merges itself when `Build` passes; record the observed merge mechanism (GitHub native auto-merge versus renovate self-merge) in the PR (Requirement 3).
  - `origin/gh-pages` receives no new commits after the switch; its deletion is the recorded follow-up (Requirement 4).
  - `helm pull oci://ghcr.io/maruina/go-infrabin --version <chart-version>` and `helm template` render succeeds (Requirement 1).

## Learning candidates

- 2026-11-11: The plan's Decision 5 premise was wrong — semantic-release tags the checked-out HEAD locally in the CI workspace and pushes the tag itself, so the checkout already contains the tag and the planned `git fetch origin <tag>` step was a no-op; the review removed the step. Evidence: semantic-release `index.js` calls `tag()` then `push()` before the publish plugins (https://github.com/semantic-release/semantic-release/blob/main/index.js); read during /systematic-review.
- 2026-11-11: semantic-release's local dry-run can only analyze branches that exist on the remote — its branch expansion matches configured names against `git ls-remote --heads`, so `--branches maruina/auto-release` failed with `The release branches are invalid` until the branch was pushed; pushing a feature branch before local release validation is required, and with `--branches master` alone the run still reports `triggered on the branch maruina/auto-release ... a new version won't be published`. Evidence: worktree `node_modules/semantic-release/lib/branches/expand.js` and `lib/definitions/branches.js`; command `GITHUB_TOKEN=$(gh auth token) npx semantic-release --dry-run --branches "$(git branch --show-current)"` before and after `git push -u origin maruina/auto-release`.
- 2026-11-11: In a semantic-release JSON config, per-plugin options only work in the two-element array form `["@semantic-release/commit-analyzer", {"releaseRules": [...]}]`; the `{pluginName: options}` object form silently has no effect. Evidence: `node_modules/semantic-release/lib/plugins/utils.js` `parseConfig` handles arrays, `{path, ...}` objects, and bare strings only; the object form first rerun reported `Analysis of 8 commits complete: no release` with the rule present.
