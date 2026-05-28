# Build And Release

This document is the current build and release handoff for Puppt. It is intentionally conservative: no production release is declared until the verification commands here pass from a clean checkout.

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

## Release Status

No tagged production release exists yet. Current artifacts are local build artifacts only. A production release still needs:

- version/tag policy
- changelog entry
- signed or checksummed binary artifacts
- CI evidence from a clean checkout
- rollback plan

## Rollback

There is no deployment target yet. Rollback for local use is source-level: return to the previous Git commit and rebuild.
