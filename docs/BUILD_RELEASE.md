# Build And Release

This document is the release handoff for Puppt. Release readiness is about
versioned distribution, repeatable artifacts, checksums, and compatibility
disclosure. Renderer-completion milestones are independent engineering work and
do not block a CLI release; renderer status is disclosed through
`docs/RENDERING.md` and `docs/SUPPORT_MATRIX.md`.

## Version Policy

Puppt uses SemVer tags:

```text
vMAJOR.MINOR.PATCH
```

Until `v1.0.0`, minor releases may change CLI or JSON behavior. Patch releases
should be reserved for bug fixes, documentation corrections, and packaging
fixes that do not intentionally change behavior.

Current release:

```text
v0.1.0
```

## Build

```sh
make build
```

Build output:

```text
bin/puppt
```

Equivalent direct command:

```sh
go build -trimpath -ldflags "-s -w" -o bin/puppt ./cmd/puppt
```

Release builds inject the Git tag into `puppt version`:

```sh
go build -trimpath -ldflags "-s -w -X github.com/artpar/puppt/internal/cli.version=vX.Y.Z" -o bin/puppt ./cmd/puppt
```

## Verification

```sh
make verify
```

This runs:

```sh
go test -count=1 ./...
git diff --check
go run ./cmd/puppt --help >/dev/null
```

## GoReleaser

Puppt releases are packaged with GoReleaser.

Current pinned release tooling, verified from upstream release metadata on
2026-06-02:

- GoReleaser: `v2.16.0`
- GoReleaser Action: `v7.2.2`
- `actions/checkout`: `v6.0.2`
- `actions/setup-go`: `v6.4.0`

Local release checks:

```sh
make release-check
make release-snapshot
```

These targets download the pinned GoReleaser release binary into `.tools/`,
verify the official checksum, and then run that binary. The snapshot command
builds the release artifact matrix under `dist/` without publishing.

## Artifact Matrix

GoReleaser builds:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`
- `windows/amd64`
- `windows/arm64`

Unix-like artifacts are `.tar.gz`; Windows artifacts are `.zip`.

Each release includes:

- archived `puppt` binaries
- selected README/docs files
- `SHA256SUMS`

## GitHub Release

Release from a clean checkout, replacing `vX.Y.Z` with the next SemVer tag:

```sh
make verify
make release-check
make release-snapshot
git status --short
git tag -a vX.Y.Z -m "puppt vX.Y.Z"
git push origin main
git push origin vX.Y.Z
```

The tag push triggers `.github/workflows/release.yml`, which runs GoReleaser and
publishes the GitHub release artifacts.

Manual release, when needed:

```sh
GITHUB_TOKEN=... make release
```

Release notes must link:

- `README.md`
- `docs/COMMANDS.md`
- `docs/SUPPORT_MATRIX.md`
- `docs/RENDERING.md`
- `CHANGELOG.md`

## Post-Release Smoke Test

Download the published artifact, then run:

```sh
puppt version
puppt --help
```

For a fixture or sample deck, also run:

```sh
puppt inspect deck.pptx --json
```

## Rollback

There is no deployment target. Rollback is release-level:

1. Leave the bad tag intact.
2. Mark the GitHub release as withdrawn in the release notes.
3. Publish a fixed patch release, for example `v0.1.1`.
4. Do not rewrite public tags after announcement.
