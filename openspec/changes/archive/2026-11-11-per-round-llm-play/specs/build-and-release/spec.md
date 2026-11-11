# Spec Delta

## Purpose

Defines how Prompt Fighter is built and released: one Makefile for local builds, the web app built as part of every Go build, version stamping, and automatic semver releases on GitHub derived from commit messages.

## ADDED Requirements

### Requirement: Makefile
The repository SHALL have a root `Makefile` whose default target builds everything into `bin/pf`, building the web app first. It SHALL provide these targets:
- `web`: install the web dependencies if needed and build the web app into `web/dist`
- `build`: `web`, then the `pf` binary into `bin/pf`, with the version stamped in
- `test`: `go vet ./...`, `go test ./...` and the web tests, building the web app first
- `release`: archives for linux, darwin and windows on amd64 and arm64 in `dist/`, with `SHA256SUMS`
- `smoke`: the MCP smoke test against a temporary `pf serve`
- `assets`: re-encode the audio and cover sources (needs ffmpeg)
- `clean`: remove `bin/`, `dist/` and `web/dist`

`VERSION` SHALL default to the output of `git describe --tags --always --dirty`, or `dev` outside a git checkout.

#### Scenario: Build from a fresh clone
- **WHEN** `make` runs in a fresh clone with Go and Node installed
- **THEN** `web/dist` is built, and `bin/pf` is produced and serves the web app

#### Scenario: Release archives
- **WHEN** `make release VERSION=v1.2.3` runs
- **THEN** `dist/` holds six archives named `pf-v1.2.3-<os>-<arch>` (zip for windows, tar.gz otherwise) and `SHA256SUMS`

### Requirement: Web build before Go builds
`web/dist` SHALL NOT be committed. Every Go build, test and release path (the Makefile, the CI workflow and the release workflow) SHALL build the web app before compiling or testing Go code, because `pf` embeds it.

#### Scenario: CI on a clean checkout
- **WHEN** the CI workflow runs on a checkout without `web/dist`
- **THEN** it builds the web app, and `go vet` and `go test ./...` pass

### Requirement: Version stamping
Builds SHALL stamp the version into the binary at link time. `pf version` SHALL print it, or `dev` for an unstamped build.

#### Scenario: Released binary
- **WHEN** the release for `v0.3.0` is built and its `pf version` is run
- **THEN** it prints `v0.3.0`

### Requirement: Next version from commits
A tested tool SHALL compute the next version from the latest `v*` tag and the commits since it, using conventional-commit subjects and bodies:
- a breaking change (`type!:` or a `BREAKING CHANGE:` footer) bumps major, or minor while the major version is 0
- otherwise any `feat` bumps minor
- otherwise any commit bumps patch
- no commits since the tag means no new version

With no tag yet, the first version SHALL be `v0.1.0`.

#### Scenario: Feature and fix
- **WHEN** the last tag is `v0.4.2` and the new commits are `fix: x` and `feat(web): y`
- **THEN** the next version is `v0.5.0`

#### Scenario: Breaking change before 1.0
- **WHEN** the last tag is `v0.4.2` and a commit is `feat!: new protocol`
- **THEN** the next version is `v0.5.0`

#### Scenario: Breaking change after 1.0
- **WHEN** the last tag is `v1.4.2` and a commit body contains `BREAKING CHANGE: removed join script`
- **THEN** the next version is `v2.0.0`

#### Scenario: Nothing new
- **WHEN** no commits follow the last tag
- **THEN** no version is produced and nothing is released

### Requirement: Automatic releases
On every push to `main`, a GitHub workflow SHALL run the tests and, if they pass and a next version exists, tag that version on the pushed commit, build the release archives with that version stamped in, and publish a GitHub release with the archives, `SHA256SUMS` and generated notes. This SHALL happen in the same workflow run (tags pushed with the workflow token start no other workflows). Pushing a `v*` tag by hand SHALL build and publish that version.

#### Scenario: Push a feature
- **WHEN** a commit `feat: spectator mode` is pushed to main after `v0.2.1`
- **THEN** the workflow tags `v0.3.0` and publishes a release with six archives and `SHA256SUMS`

#### Scenario: Tests fail
- **WHEN** a pushed commit fails `go test`
- **THEN** no tag and no release are created
