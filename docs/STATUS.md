# Puppt Status

## Current Checkpoint

Checkpoint 8: Review and v1 Hardening.

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
- `puppt plan <input.pptx> --edit <edit.json> --json` is wired.
- Target resolution supports slide number, title, visible text, object ID, notes by slide number, metadata property, and image object refs.
- Planning detects ready, no-match, and ambiguous targets before mutation.
- Ambiguous plan results emit JSON and return non-zero.
- Planning validates supported operation/target combinations and required fields for text, notes, metadata, image replacement, and slide-move plans.
- Unsupported plan results emit JSON and return non-zero.
- Plan examples are documented in `docs/PLAN_EXAMPLES.md`.
- Checkpoint 3 targeting and edit-planning exit evidence is recorded in `docs/CHECKPOINTS.md`.
- `puppt edit <input.pptx> --edit <edit.json> --out <output.pptx> --json` is wired for supported text, notes, and metadata mutations.
- Owned PPTX package writing exists in `internal/pptx`.
- Targeted text replacement, deck-wide text replacement, speaker-note update, and metadata update run through planning before mutation.
- Edit output validation runs after writing and reports `validation.valid`.
- Checkpoint 4 text, notes, metadata, and validation exit evidence is recorded in `docs/CHECKPOINTS.md`.
- Slide add, delete, move, and duplicate mutations work on deterministic fixtures.
- Slide operations update presentation slide order, presentation relationships, and slide content-type overrides where needed.
- Slide operation outputs validate through relationship checks and round-trip inspection.
- Checkpoint 5 slide-operation exit evidence is recorded in `docs/CHECKPOINTS.md`.
- Explicit image target replacement updates package media bytes safely.
- Ambiguous image targets are rejected before mutation.
- Simple editable text boxes and rectangle shapes can be added to existing slides.
- Unsupported advanced visual edits remain rejected before mutation.
- Checkpoint 6 image replacement and simple additions exit evidence is recorded in `docs/CHECKPOINTS.md`.
- `puppt create --input <deck.json> --out <output.pptx> --json` is wired.
- Creation supports title slides, section slides, title/body slides, bullet lists, speaker notes, metadata, and provided images.
- Created decks inspect and validate after writing.
- Created deck output is deterministic for the same structured input.
- Checkpoint 7 creation workflow exit evidence is recorded in `docs/CHECKPOINTS.md`.
- `puppt validate <input.pptx> --json` is wired.
- `puppt review <input.pptx> --changes <changes.json> --json` is wired.
- Review output includes changes, skipped/ambiguous/unsupported items from prior command results, inspection facts, and validation status.
- Command docs, support matrix, failure modes, and acceptance workflow docs exist.
- End-to-end CLI acceptance workflow is covered by tests.

## Not Implemented Yet

- Advanced non-text object extraction, richer media metadata, and broader real-world unsupported-feature warning detection beyond the current basic warning set.
- Adding new notes parts when a slide has no notes relationship.
- Rich visual fidelity verification beyond editable package structure.

All required v1 command names are implemented: `inspect`, `plan`, `edit`, `create`, `validate`, `review`, and `version`.

## Next Checkpoint

Checkpoint 8: Review and v1 Hardening.
