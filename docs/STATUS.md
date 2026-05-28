# Puppt Status

## Current Checkpoint

Checkpoint 3: Targeting and Edit Planning.

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
- `puppt inspect <input.pptx> --json` is wired.
- Inspection JSON uses the `puppt.v1` envelope.
- Inspection currently reports presentation part, package part count, slide count, slide order, simple titles, shape-level visible text runs, speaker notes, image/media references, layout references and names, master references and names, core metadata, repeated visible text, and basic unsupported-part/object warnings.
- Golden JSON test exists for a deterministic minimal deck.
- Checkpoint 2 inspection exit evidence is recorded in `docs/CHECKPOINTS.md`.

## Not Implemented Yet

- Advanced non-text object extraction, richer media metadata, and broader real-world unsupported-feature warning detection beyond the current basic warning set.
- Target resolution and edit planning.
- Mutations.
- Deck creation.
- Validation.
- Review summaries.
- Fixtures and acceptance suite.

Commands other than `inspect`, `version`, and `--help` currently fail explicitly as unimplemented.

## Next Checkpoint

Checkpoint 3: Targeting and Edit Planning.
