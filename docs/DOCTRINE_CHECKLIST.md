# Puppt Doctrine Checklist

This checklist maps current implementation practice to the binding Puppt layer in `swe_skill.md`. It is updated when a checkpoint changes the engineering surface.

## Current Enforcement

| Doctrine rule | Current evidence |
|---|---|
| Puppt v1 MUST be implemented in Go | `go.mod`, `cmd/puppt`, and all product code are Go |
| CLI code in `cmd/puppt` MUST stay thin | `cmd/puppt/main.go` only delegates to `internal/cli` |
| Business logic belongs in internal packages | `.pptx` package reading is in `internal/pptx`; inspection is in `internal/inspect`; JSON output is in `internal/report` |
| Use `context.Context` for I/O operations | `cli.Execute`, `inspect.Inspect`, and `pptx.Open` accept context |
| Return explicit errors for malformed decks and unsupported operations | `internal/pptx.PackageError` classifies package failures; unimplemented commands return explicit errors |
| Prefer reliable libraries for non-core infrastructure | Cobra is used for CLI routing and documented in `docs/decisions/0001-cli-library.md` |
| Puppt MUST own authoritative `.pptx` reader/writer | Documented in `docs/decisions/0002-own-pptx-reader-writer.md`; reader is implemented in `internal/pptx` |
| Do not shell out to office software in core path | No shell-out dependency exists |
| Deterministic JSON and tests | `internal/report.WriteJSON` uses stable indentation; `internal/inspect/testdata/minimal.golden.json` pins output shape |
| Tests cover the behavior at the right risk level | Inspection has focused tests for slide order, shape-level visible text, metadata, notes, image/audio/video/OLE refs, layout and master refs/names, repeated text, unsupported warnings, and CLI JSON; edit planning has focused tests for target resolution, ambiguous/no-match handling, unsupported operations, required fields, image refs, and slide-operation fields; mutation tests cover text, notes, metadata, slide add/delete/move/duplicate, image replacement, simple editable additions, preservation of unrelated media, and post-edit validation |
| Detect ambiguity before mutation | `internal/target.Resolve` classifies ready, no-match, and ambiguous targets; `puppt plan --json` emits ambiguity details and exits non-zero |
| Unsupported behavior is explicit | `internal/edit.Plan` rejects unsupported operations, operation/target mismatches, resolved target-kind mismatches, and missing required fields before mutation |
| Documentation required for operation is part of the system | `docs/PLAN_EXAMPLES.md`, `docs/STATUS.md`, and `docs/CHECKPOINTS.md` document current command behavior and limits |
| Round-trip and preservation tests are required before mutations | Text, notes, metadata, slide, image, and simple-addition mutations inspect and validate edited output; targeted text replacement verifies unrelated media part preservation |
| Progress records must state changes, verification, risks, and next checkpoint | `docs/CHECKPOINTS.md` records Checkpoints 0, 1, 2, 3, 4, 5, and 6 completion evidence |

## Current Known Gaps

- Advanced non-text object extraction, richer media metadata, and broader real-world unsupported-feature warning detection are not implemented yet.
- Deck creation and review workflows are not implemented yet.
- Notes updates require an existing notes relationship; adding new notes parts is not implemented yet.
- `inspect` emits `inspection_partial` for advanced non-text object extraction.
