# Arkeep (PunterDigital fork)

`PunterDigital/arkeep` is a GitHub fork of [arkeep-io/arkeep](https://github.com/arkeep-io/arkeep)
(Apache-2.0): self-hosted backup management on Restic and Rclone. A Go **server** (REST API,
gRPC hub for agents, embedded Vue GUI) and a Go **agent** that runs on each machine.

We ship **our own patched builds** and send fixes upstream now and then (Shay, Nov 13 2026).
Everything we change on top of upstream is listed in `PATCHES.md`.

## Rules for a fork

- **Keep the diff against upstream small.** Don't edit upstream's files when a new file will
  do: our CI and release live in `pd-ci.yml`, `release-pd.yml` and `.goreleaser.pd.yml`, and
  upstream's `ci.yml` / `security.yml` run unchanged. Every edit to an upstream file is a
  future merge conflict.
- **Licence stays Apache-2.0** (never MIT). Each patch goes in `PATCHES.md` (what, why, files,
  upstream PR if any): that's also the Apache 4(b) notice for modified files.
- **Runners: GitHub-hosted only.** The repo is public and a fork, so a self-hosted runner would
  run fork-PR code on Shay's machine (Shay agreed, Nov 13).
- **Merge strategy.** Our own PRs: squash. Upstream syncs: a **merge commit**
  (`gh pr merge --merge`), never squash, or the next sync conflicts on everything.
- Workflows disabled on the fork (`gh workflow disable`, a repo setting, so no file change):
  `release.yml` (upstream's, Docker Hub + arkeep-io), `registry-metadata.yml`, `sponsors.yml`,
  `deps-pins.yml`. Dependabot version updates stay off: dependency bumps arrive with upstream
  syncs. Re-check after a sync that adds a new workflow.

## Versions and releases

- Tag `vX.Y.Z-pd.N`: our N-th build on top of upstream `vX.Y.Z`. The upstream tag must be in
  our repo too (`git push origin refs/tags/vX.Y.Z`); `release-pd.yml` checks it's merged.
- The server's update check (`server/internal/api/version.go`) ignores pre-release suffixes,
  so `0.11.0-pd.1` counts as equal to upstream `0.11.0` (no false "update available"). It
  still asks **upstream's** releases, so a new upstream version shows the update badge before
  we've shipped it: that's our cue to sync.
- `release-pd.yml` (on `v*-pd.*` tags): GoReleaser with `.goreleaser.pd.yml` builds the GUI,
  server (linux/darwin) and agent (linux incl. armv7/mips, darwin, windows) archives, pushes
  `ghcr.io/punterdigital/arkeep-server` / `-agent` (`:<tag>` and `:latest`), attests
  provenance, then an asset-check job confirms all 13 assets. No Docker Hub, no Helm chart
  (upstream's chart points at upstream images).
- Release notes: changelog since the upstream base for `-pd.1` (so every patch shows), since
  the previous `-pd` tag after that.
- Upstream's `.goreleaser.yml` stays as-is. When an upstream sync changes it, port the change
  to `.goreleaser.pd.yml` in the same sync PR.

## Syncing upstream (weekly maintenance, or when upstream releases)

```bash
git fetch upstream --tags
git switch -c claude/sync-vX.Y.Z main
git merge vX.Y.Z            # the upstream release tag, not upstream/main (that's unreleased develop work)
# resolve conflicts (keep our patches), port .goreleaser.yml changes, update PATCHES.md if a patch landed upstream
git push origin refs/tags/vX.Y.Z claude/sync-vX.Y.Z
gh pr create … ; wait for CI; gh pr merge --merge
```

Then release `vX.Y.Z-pd.1`. A patch that upstream has merged comes out of `PATCHES.md`.

## Contributing upstream: the `upstream` tag

Shay decides per card (Nov 13). A GroundStation card tagged **`upstream`** goes upstream; that
tag is his OK to open the public PR under his name. A card without it is a fork-only patch:
don't offer it upstream unasked (suggesting the tag in a triage comment is fine).

For an `upstream` card:

1. Branch from upstream's `develop` (they merge features there and release from `main`):
   `git fetch upstream && git switch -c fix/<slug> upstream/develop`. Follow `CONTRIBUTING.md`
   (Conventional Commits, their PR template, their tests).
   **Code only** (Shay, Nov 13): the app change and its tests, nothing else. No `CLAUDE.md`,
   `PATCHES.md`, `e2e/`, `pd-*` workflows, `.goreleaser.pd.yml`, `.claude/` or other notes of
   ours, and no edits to upstream's docs unless the card asks for it. Before pushing, check:
   `git diff --name-only upstream/develop...HEAD` must list only code/test files.
   **No Claude attribution** in anything that goes upstream (Shay, Nov 13): no
   `Co-Authored-By: Claude …` trailer in the commits and no "Generated with Claude Code" line in
   the PR body, whatever the session's default attribution says. Check with
   `git log upstream/develop..HEAD --format=%B | grep -i claude` (must print nothing). Our own
   carry-over PR on `main` (step 3) keeps the normal attribution.
2. Push the branch to our fork and open the PR against `arkeep-io/arkeep:develop`.
3. Carry it in our build: cherry-pick onto a `claude/<card>-<slug>` branch from our `main`, add a
   `PATCHES.md` row with the upstream PR link, our PR, squash-merge, then release `-pd.N+1`.
4. When an upstream release includes it, the sync PR drops the `PATCHES.md` row (resolve any
   conflict in upstream's favour). If upstream closes the PR, it stays a fork patch: say so on
   the card.

## Commands

No Go on this PC: run Go in `golang:1.27` (version from `go.work`) with the tree piped in
(`DOCKER_CONTEXT=desktop-linux`):

```bash
tar --exclude=./gui/node_modules --exclude=./e2e/node_modules --exclude=./.git -cf - . \
  | docker run --rm -i golang:1.27 sh -c 'mkdir /src && tar xf - -C /src && cd /src/agent && go run ./scripts/download_deps.go && cd /src/server && go test ./...'
```

- GUI (`gui/`, pnpm from `packageManager`): `corepack pnpm@<version> install --frozen-lockfile`,
  `… build` (includes vue-tsc), `… test` (vitest).
- E2E (`e2e/`, Playwright, npm): needs a server binary in `bin/`. Build the GUI, copy
  `gui/dist/.` into `server/cmd/server/dist/`, then cross-compile from the container
  (`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOWORK=off go build -o /tmp/arkeep-server.exe ./cmd/server`,
  `cat` it out to `bin/arkeep-server.exe`). `cd e2e && npm ci && npx playwright test`.
  `start-server.mjs` runs it on 127.0.0.1:18080/19090 with a temp SQLite DB and test-only
  secrets. CI builds a Linux binary the same way (`pd-ci.yml`).
- Pins (ADR 0013): `bash scripts/check-pins.sh` covers our workflows too. Actions by full SHA
  (look it up with `gh api repos/<action>/commits/<tag> --jq .sha`, never from memory), npm
  versions exact.

## Quirks

- On Windows, 3 GUI test files fail to load (`file:///icons/...` / `file:///login-bg.mp4`
  isn't an absolute path to Node): only Windows, CI on Linux is green.
- In the golang container the agent's `TestEnsureWritableDir/read-only_leaf` fails: it runs as
  root, which ignores read-only permissions. CI runs as a normal user.
- A signed-out GUI page logs `401 (Unauthorized)` console errors (session probe): the E2E
  ignores exactly those.
- First-run setup has a window (900 s) after server start; a restart reopens it.
