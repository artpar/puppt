# Puppt Status

## Current Checkpoint

Checkpoint 0: Repository Foundation.

## Implemented

- Go module exists.
- `cmd/puppt` CLI entrypoint exists.
- Required v1 command names are registered.
- `puppt --help` and `puppt version` work.
- Internal package layout exists for the planned modules.
- Initial test command is `go test ./...`.

## Not Implemented Yet

- `.pptx` package reading.
- Deck inspection.
- Target resolution and edit planning.
- Mutations.
- Deck creation.
- Validation.
- Review summaries.
- Fixtures and acceptance suite.

Commands other than `version` and `--help` currently fail explicitly with a repository-foundation message.

## Next Checkpoint

Checkpoint 1: `.pptx` Package Reader.
