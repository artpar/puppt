# Puppt Checkpoint Log

## Checkpoint 0: Repository Foundation

Changed files:

- `go.mod`
- `go.sum`
- `README.md`
- `cmd/puppt/main.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `internal/*/doc.go`
- `docs/STATUS.md`
- `docs/decisions/0001-cli-library.md`
- `project-ops.md`
- `swe_skill.md`

Implemented behavior:

- Established Go module `github.com/artpar/puppt`.
- Added thin `cmd/puppt` entrypoint.
- Added `internal/cli` command wiring using Cobra.
- Registered required v1 command names: inspect, plan, edit, create, validate, review.
- Added `version` and `--help` behavior.
- Stubbed unimplemented workflow commands with explicit errors.
- Created planned internal package layout.
- Added baseline CLI tests.
- Documented current status and first dependency decision.
- Updated doctrine to prefer reliable third-party Go libraries where they reduce risk.

Verification commands:

```text
go test ./...
go run ./cmd/puppt --help
```

Verification result:

- `go test ./...` passed.
- `go run ./cmd/puppt --help` passed and listed the required v1 commands.

Fixtures added or updated:

- None. Fixture work begins with `.pptx` package reader and inspection checkpoints.

Known risks:

- No `.pptx` package reading exists yet.
- Workflow commands other than `version` and `--help` are explicit stubs.
- Dependency evaluation for `.pptx` parsing/editing libraries has not been performed yet.

Unsupported behavior encountered:

- All `.pptx` workflows remain unsupported until later checkpoints.

Next checkpoint:

- Checkpoint 1: `.pptx` Package Reader.

## Checkpoint 1: `.pptx` Package Reader

Changed files:

- `docs/REFERENCES.md`
- `docs/STATUS.md`
- `docs/CHECKPOINTS.md`
- `docs/decisions/0002-own-pptx-reader-writer.md`
- `project-ops.md`
- `swe_skill.md`
- `internal/pptx/doc.go`
- `internal/pptx/errors.go`
- `internal/pptx/reader.go`
- `internal/pptx/reader_test.go`
- `internal/fixtures/doc.go`
- `internal/fixtures/pptx.go`

Implemented behavior:

- Clarified that Puppt owns the authoritative `.pptx` package reader/writer and mutation path.
- Added official reference map for ECMA-376, ISO/IEC 29500, Microsoft PresentationML structure, and Microsoft Office ISO/IEC 29500 implementation notes.
- Added a decision record for Puppt-owned `.pptx` reader/writer.
- Implemented explicit package error kinds for unsupported file type, invalid package, missing part, invalid XML, and missing relationship.
- Implemented initial `.pptx` package opening with extension and ZIP validation.
- Parsed `[Content_Types].xml`.
- Parsed `_rels/.rels`.
- Resolved the office document relationship to `ppt/presentation.xml`.
- Parsed `ppt/_rels/presentation.xml.rels`.
- Parsed presentation slide IDs and resolved ordered slide part names through relationships.
- Added deterministic minimal `.pptx` fixture builder for tests.

Verification commands:

```text
go test ./internal/pptx ./internal/validate
go test ./...
```

Verification result:

- `go test ./internal/pptx ./internal/validate` passed.
- `go test ./...` passed.

Fixtures added or updated:

- Added `internal/fixtures.WriteMinimalPPTX` for deterministic test-generated `.pptx` fixtures.

Known risks:

- Reader currently loads package parts into memory; streaming preservation can be added when mutation workflows require it.
- Reader only exposes package structure and slide part order; it does not inspect slide contents yet.
- No CLI command is wired to the package reader yet.

Unsupported behavior encountered:

- Real-world decks with layouts, masters, notes, media, charts, macros, or extension content are not inspected yet.

Next checkpoint:

- Checkpoint 2: Inspection Core.

## Checkpoint 2: Inspection Core, progress 1

Changed files:

- `internal/model/doc.go`
- `internal/model/result.go`
- `internal/model/inspection.go`
- `internal/report/doc.go`
- `internal/report/json.go`
- `internal/inspect/doc.go`
- `internal/inspect/inspect.go`
- `internal/inspect/inspect_test.go`
- `internal/inspect/testdata/minimal.golden.json`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/STATUS.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Added stable shared command-result and inspection models.
- Added stable indented JSON report writer.
- Implemented basic inspection workflow using the owned `.pptx` package reader.
- Extracted slide order, slide part IDs, and visible text runs from slide XML.
- Derived simple slide title from first visible text block.
- Added repeated visible text representation.
- Represented metadata, notes, images, layouts, slide warnings, top-level warnings, and errors in the JSON shape.
- Added `puppt inspect <input.pptx> --json`.
- Added focused inspect tests and a golden JSON test.
- Added CLI JSON test for `inspect`.
- Added a doctrine checklist mapping current code to the binding Puppt rules in `swe_skill.md`.

Verification commands:

```text
go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt
go test ./...
```

Verification result:

- `go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt` passed.
- `go test ./...` passed.

Fixtures added or updated:

- Added `internal/inspect/testdata/minimal.golden.json`.
- Reused deterministic generated `.pptx` fixture from `internal/fixtures.WriteMinimalPPTX`.

Known risks:

- Notes, images, layout refs, and metadata fields are represented but not populated yet.
- Visible text grouping is currently slide-level, not shape-level.
- `inspect` emits a partial-inspection warning until the remaining inspection fields are populated.

Unsupported behavior encountered:

- Real-world rich slide constructs remain uninspected beyond raw visible `a:t` text.

Next checkpoint:

- Continue Checkpoint 2 by populating shape/object-level text, notes, image/media references, metadata, layouts, repeated content fixtures, and warning detection.

## Checkpoint 2: Inspection Core, progress 2

Changed files:

- `internal/pptx/reader.go`
- `internal/fixtures/pptx.go`
- `internal/inspect/inspect.go`
- `internal/inspect/inspect_test.go`
- `internal/inspect/testdata/minimal.golden.json`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Exposed Puppt-owned package helpers for relationship-part lookup, internal target resolution, per-part relationships, and content-type lookup.
- Extended deterministic fixture generation to include core metadata, notes slides, image relationships, and slide layout relationships.
- Populated core metadata from `docProps/core.xml`.
- Populated speaker notes from notes slide relationships.
- Populated image/media references from slide relationships with content type lookup.
- Populated slide layout references from slide relationships.
- Added repeated visible text test coverage.
- Updated golden JSON warning text to reflect the narrower remaining inspection gap.

Verification commands:

```text
go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt
go test ./...
```

Verification result:

- `go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt` passed.
- `go test ./...` passed after the implementation change.

Fixtures added or updated:

- Extended `internal/fixtures.WritePPTX` for deterministic rich decks.
- Updated `internal/inspect/testdata/minimal.golden.json`.

Known risks:

- Visible text is still grouped at slide level rather than shape/object level.
- Slide master inspection is not populated yet.
- Unsupported-feature warning detection is not complete.

Unsupported behavior encountered:

- Rich real-world package constructs beyond simple notes, images, layouts, and core metadata remain unclassified.

Next checkpoint:

- Continue Checkpoint 2 with shape/object-level text extraction, slide master/layout naming, and unsupported-feature warning detection.
