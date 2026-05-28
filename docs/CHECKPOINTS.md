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
