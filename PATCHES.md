# PunterDigital patches

This is PunterDigital's build of [Arkeep](https://github.com/arkeep-io/arkeep), released as
`vX.Y.Z-pd.N` on top of upstream `vX.Y.Z`. This file lists everything that differs from
upstream (Apache-2.0, section 4(b): modified files carry this notice).

## Build and release (no change to the app)

| Change | Files |
|---|---|
| Our own release: GHCR images under `ghcr.io/punterdigital`, GitHub Releases on this repo, `-pd.N` tags | `.goreleaser.pd.yml`, `.github/workflows/release-pd.yml` (new) |
| Playwright smoke test of the built server and GUI, plus a check of the release config | `e2e/`, `.github/workflows/pd-ci.yml` (new) |
| Fork notes and these docs | `CLAUDE.md`, `PATCHES.md` (new), a note at the top of `README.md` |

Upstream's `release.yml`, `registry-metadata.yml`, `sponsors.yml` and `deps-pins.yml` are
disabled on this fork (repository setting; the files are unchanged).

## App changes

None yet. Each one gets a row here: what it does, why, the files it touches, and the
upstream PR if it was sent upstream (cards tagged `upstream` are; see `CLAUDE.md`). A row
leaves this table once an upstream release includes the change.

| Change | Why | Files | Upstream |
|---|---|---|---|
