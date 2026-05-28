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
| Tests cover the behavior at the right risk level | Inspection has focused tests for slide order, visible text, metadata, notes, image refs, layout refs, repeated text, and CLI JSON |
| Round-trip and preservation tests are required before mutations | No mutation support exists yet; preservation tests are a gate for future edit checkpoints |
| Progress records must state changes, verification, risks, and next checkpoint | `docs/CHECKPOINTS.md` records Checkpoints 0, 1, and current Checkpoint 2 progress |

## Current Known Gaps

- Shape-level text grouping is not implemented yet.
- Slide masters, full media classification, and real-world unsupported-feature warning detection are not implemented yet.
- No mutation path exists yet, so round-trip preservation tests have not started.
- `inspect` emits `inspection_partial` until the rest of Checkpoint 2 is complete.
