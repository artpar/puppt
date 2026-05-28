# Puppt Status

## Current Checkpoint

Checkpoint 1: `.pptx` Package Reader.

## Implemented

- Go module exists.
- `cmd/puppt` CLI entrypoint exists.
- Required v1 command names are registered.
- `puppt --help` and `puppt version` work.
- Internal package layout exists for the planned modules.
- Initial test command is `go test ./...`.
- Official reference map exists for ECMA-376, ISO/IEC 29500, Microsoft PresentationML structure, and Microsoft Office implementation notes.
- Puppt-owned `.pptx` package reader opens ZIP packages, reads content types, reads root relationships, resolves the presentation part, reads presentation relationships, and exposes slide part order.
- Invalid extension, invalid ZIP, and missing required part cases fail explicitly.

## Not Implemented Yet

- Deck inspection.
- Target resolution and edit planning.
- Mutations.
- Deck creation.
- Validation.
- Review summaries.
- Fixtures and acceptance suite.

Commands other than `version` and `--help` currently fail explicitly with a repository-foundation message. The package reader is available internally but is not yet wired to `puppt inspect`.

## Next Checkpoint

Checkpoint 2: Inspection Core.
