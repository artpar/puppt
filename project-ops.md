# Puppt Project Operations Protocol

## Authority

This file is the operating protocol for building Puppt. It is binding. It translates `goal.md`, `PRODUCT_VISION.md`, `USER_EXPERIENCE.md`, and `swe_skill.md` into concrete execution rules, module boundaries, checkpoints, validation gates, and evidence requirements.

Puppt exists to inspect, edit, create, validate, and review modern `.pptx` files without destroying editability or unrelated deck content. Every implementation step MUST preserve that identity.

## Non-Negotiable Project Rules

1. Puppt v1 MUST be implemented in Go.
2. Puppt MUST operate on modern `.pptx` files, which are ZIP packages containing Open XML parts.
3. Puppt MUST inspect before mutating unless the command is pure creation.
4. Puppt MUST prefer surgical package/XML edits over deck regeneration.
5. Puppt MUST keep output editable as normal PowerPoint files.
6. Puppt MUST preserve unsupported and untargeted package parts wherever possible.
7. Puppt MUST fail explicitly on unsupported file types, unsafe targets, ambiguity, malformed packages, corrupted output, and unsupported requested edits.
8. Puppt MUST produce both machine-readable JSON and concise human-readable summaries for inspect, edit, create, validate, and review workflows.
9. Puppt MUST maintain stable CLI/API output fields once v1 contracts are introduced.
10. Puppt MUST record changed slides, changed objects/content, skipped edits, ambiguous matches, unsupported features, warnings, and validation status after every mutating operation.
11. Puppt MUST keep dependencies bounded and justified. A dependency that mutates `.pptx` files, parses XML, generates images, shells out to office software, or affects validation MUST have an explicit decision record before adoption.
12. Puppt MUST NOT flatten slides into screenshots as the product output.
13. Puppt MUST NOT silently drop notes, media, relationships, layouts, masters, themes, metadata, or unknown package parts.
14. Puppt MUST NOT hide partial success. Warnings and skipped edits are part of the result.

## Go Implementation Constraint

The product core, CLI, public API surface, tests, and fixtures MUST be Go-native.

The expected repository shape is:

```text
cmd/puppt/                  CLI entrypoint
internal/pptx/              package reader, writer, relationships, content types
internal/model/             stable deck, slide, object, media, notes, warning models
internal/inspect/           deck inspection pipeline
internal/target/            target resolution and ambiguity detection
internal/edit/              edit planner and mutation engine
internal/create/            deck creation from structured input
internal/validate/          structural and content validation
internal/report/            JSON and human summary generation
internal/fixtures/          Go fixture builders and fixture helpers
testdata/                   committed sample decks and golden outputs
docs/                       operation docs, support matrix, examples
```

Public Go package boundaries MAY be added later if embedding Puppt as a library becomes a v1 requirement. Until then, the stable external interface is the CLI contract and its JSON output.

## Architecture Principles

### Package-Level Editing

Puppt MUST treat `.pptx` as an Open XML package. The mutation engine MUST read and write only the package parts required for the requested operation whenever possible.

For existing decks, whole-deck reconstruction is not an acceptable default. It is allowed only when:

1. The requested operation is creation of a new deck.
2. The code proves all required editable structures are recreated intentionally.
3. The operation summary reports that the deck was generated rather than surgically edited.

### Stable Object Model

Inspection MUST produce stable, addressable facts. The model MUST include at minimum:

- presentation metadata
- slide count and slide order
- slide id and 1-based slide number
- layout and master references where available
- title candidates and chosen title
- visible text grouped by shape/object
- text runs where safely available
- speaker notes
- image and media references
- package relationships relevant to slides
- repeated text findings
- warnings and unsupported feature observations

Object identifiers MUST be deterministic for the same input deck. If PowerPoint-native IDs are available, include them. If generated IDs are needed, they MUST be derived from stable package path, slide id, relationship id, shape id/name, and object position.

### Explicit Targeting

All edits MUST pass through a target resolver before mutation. Supported v1 target types are:

- slide number
- slide id
- slide title
- visible text match
- object id
- speaker notes on a slide
- deck metadata property
- image/media relationship or object id
- insertion position for slide operations

The resolver MUST detect:

- no match
- single match
- multiple exact matches
- multiple fuzzy or inferred matches
- unsupported target type
- target exists but requested operation is unsupported

Mutations MAY proceed only for no-ambiguity targets unless the command explicitly declares deck-wide scope.

### Edit Planning

Every mutating command MUST produce a plan before writing. The plan MUST identify:

- input file
- output file
- operation type
- target selector
- matched slides and objects
- intended old and new content when applicable
- expected match count when provided
- risks and warnings
- unsupported or skipped requested actions

The CLI SHOULD expose dry-run planning for edits. Internally, the planner MUST be usable by tests without writing a file.

### Mutation Engine

The mutation engine MUST preserve unrelated package parts byte-for-byte where feasible. When byte-for-byte preservation is not feasible because a ZIP archive is rewritten, preservation MUST be proven structurally by package part inventory, relationship integrity, content validation, and targeted content checks.

Priority v1 mutations:

1. Replace text in a targeted object.
2. Replace text deck-wide with exact match count reporting.
3. Add or update speaker notes.
4. Update deck metadata.
5. Add, delete, move, and duplicate slides.
6. Replace an image by explicit object or relationship target.
7. Add simple editable text boxes and simple editable shapes where safely supported.

Do not implement broad "best effort" mutation that may damage deck structure. Unsupported edits MUST produce explicit skipped results.

### Creation Engine

Creation MUST generate editable `.pptx` files from structured input. v1 creation MUST support:

- title slides
- section slides
- title plus body slides
- bullet lists
- speaker notes
- images when a local image path is provided
- simple deterministic layouts
- basic metadata

Creation SHOULD use a small number of deterministic layouts. The goal is clean editable output, not design-heavy generation.

### Validation

Validation MUST check both structure and requested content. At minimum, validation MUST verify:

- file is a ZIP package
- required presentation parts exist
- content types part exists and is parseable
- root relationships exist and are parseable
- presentation relationship graph reaches slides
- each referenced slide part exists
- each slide relationship part is parseable when present
- notes relationships are valid when notes exist
- media relationships resolve to package parts when internal
- edited or generated expected content is present
- no requested edit is falsely reported as successful

Validation MUST return `valid`, `warnings`, and `errors` fields. A file with warnings MAY still be usable, but the warnings MUST be reported.

## CLI Contract

The CLI binary name is `puppt`.

Required v1 commands:

```text
puppt inspect <input.pptx> --json
puppt plan <input.pptx> --edit <edit.json> --json
puppt edit <input.pptx> --edit <edit.json> --out <output.pptx> --json
puppt create --input <deck.json> --out <output.pptx> --json
puppt validate <input.pptx> --json
puppt review <input.pptx> --changes <changes.json> --json
```

Human-readable output MAY be the default, but `--json` MUST produce stable machine-readable output suitable for agents.

CLI commands MUST exit non-zero for unsupported file types, invalid input, failed validation, ambiguous edit targets, corrupted output, and internal errors. Commands that complete with warnings but valid output MAY exit zero only if the JSON clearly reports warnings.

## JSON Result Shape

Every command result MUST include:

```json
{
  "schema_version": "puppt.v1",
  "command": "inspect",
  "status": "ok",
  "input": "deck.pptx",
  "output": null,
  "warnings": [],
  "errors": [],
  "summary": {
    "human": "Found 12 slides."
  }
}
```

Mutating command results MUST also include:

```json
{
  "plan": {},
  "changes": [],
  "skipped": [],
  "ambiguous": [],
  "unsupported": [],
  "validation": {
    "valid": true,
    "warnings": [],
    "errors": []
  }
}
```

Fields MAY grow additively. Existing v1 fields MUST NOT be removed or repurposed without a documented compatibility decision.

## v1 Support Matrix

| Capability | v1 requirement | Required evidence |
|---|---|---|
| Inspect `.pptx` package | Required | Fixture and golden JSON tests |
| Slide order and titles | Required | Inspection tests with titled and untitled slides |
| Visible text | Required | Shape/text fixtures with exact expected text |
| Speaker notes | Required | Notes fixture and round-trip tests |
| Images/media references | Required | Image fixture with relationship inspection |
| Layout/master references | Required where present | Inspection includes refs or warnings |
| Repeated content | Required for visible text | Golden repeated phrase output |
| Warnings | Required | Unsupported/unknown feature fixture |
| Targeted text replace | Required | Round-trip unchanged-slide checks |
| Deck-wide text replace | Required | Exact match count test |
| Slide add/delete/move/duplicate | Required | Structural validation and order checks |
| Notes update | Required | Notes XML and inspection checks |
| Image replacement | Required when explicitly targetable | Media relationship and content checks |
| Metadata update | Required | Core/app property checks |
| Simple text/shape addition | Required where safely supported | Editable shape inspection |
| New deck creation | Required | Generated deck validates and inspects |
| Preview rendering | Non-v1 unless already available | Documented as unsupported/non-blocking |
| Legacy `.ppt` | Out of scope | Explicit unsupported error |
| Macros/VBA editing | Out of scope | Preserve-or-explicitly-unsupported tests |
| Embedded charts/SmartArt editing | Out of scope for v1 | Preserve and warn when detected |

## Dependency Policy

Start with Go standard library support for ZIP and XML package handling unless evidence shows a maintained Go library provides safer `.pptx` support without weakening preservation.

Before adding a dependency, record:

- package name and version
- license
- maintenance state
- security posture
- what part of Puppt it owns
- whether it reads, writes, or validates `.pptx`
- how unsupported features are preserved
- rollback/removal plan

Shelling out to LibreOffice, PowerPoint, Keynote, browser renderers, or image conversion tools MUST NOT be part of the core v1 mutation path. Such tools MAY be used for optional manual verification only if the core product remains Go-native and editable-output-first.

## Fixtures and Tests

Test fixtures MUST be deterministic and committed when small enough. Large or externally sourced decks require provenance notes and a minimized derivative fixture when possible.

Required fixture categories:

1. Simple deck with title and body text.
2. Deck with repeated phrase across slides.
3. Deck with speaker notes.
4. Deck with images/media relationships.
5. Deck with metadata properties.
6. Deck with multiple layouts.
7. Deck with ambiguous repeated target text.
8. Deck with unsupported but preserved content.
9. Corrupted or invalid package.
10. Generated deck from structured JSON input.

Required test layers:

- unit tests for pure model, targeting, planning, validation, and reporting logic
- package tests for `.pptx` read/write and relationship handling
- golden tests for inspection and CLI JSON shapes
- round-trip tests proving unrelated slides and package parts remain usable
- negative tests for unsupported file types, missing slides, no matches, ambiguity, invalid edit specs, and corrupted output
- command tests for exit codes and stable result fields

## Checkpoint Workflow

Work MUST proceed in checkpoints. Each checkpoint has an entry condition, implementation scope, verification command, and progress record.

### Checkpoint 0: Repository Foundation

Exit evidence:

- `go.mod` exists.
- `cmd/puppt` builds.
- package layout exists.
- README or docs entry explains current supported status.
- initial test command is documented.

Verification:

```text
go test ./...
go run ./cmd/puppt --help
```

### Checkpoint 1: `.pptx` Package Reader

Exit evidence:

- rejects non-`.pptx` and invalid ZIP files with explicit errors
- reads package parts, content types, relationships, and presentation root
- exposes slide part order
- tests cover valid and invalid packages

Verification:

```text
go test ./internal/pptx ./internal/validate
```

### Checkpoint 2: Inspection Core

Exit evidence:

- `puppt inspect --json` returns stable JSON
- slide order, titles, visible text, notes, media refs, metadata, layouts, repeated content, and warnings are represented
- golden fixture tests exist

Verification:

```text
go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt
```

### Checkpoint 3: Targeting and Edit Planning

Exit evidence:

- targets by slide number, title, text, object id, notes, metadata, and image refs
- ambiguity and no-match are detected before mutation
- `puppt plan --json` reports intended changes and skipped/ambiguous targets

Verification:

```text
go test ./internal/target ./internal/edit ./cmd/puppt
```

### Checkpoint 4: Text, Notes, and Metadata Mutations

Exit evidence:

- targeted text replacement preserves unrelated content
- deck-wide replacement reports exact match count
- notes update round-trips through inspection
- metadata update validates
- output validation runs after edit

Verification:

```text
go test ./internal/edit ./internal/validate ./cmd/puppt
```

### Checkpoint 5: Slide Operations

Exit evidence:

- add, delete, move, and duplicate slide workflows work on fixtures
- slide relationships remain valid
- slide order and content validate after mutation
- summaries name touched slide positions and ids

Verification:

```text
go test ./internal/edit ./internal/inspect ./internal/validate ./cmd/puppt
```

### Checkpoint 6: Image Replacement and Simple Additions

Exit evidence:

- explicit image target replacement updates package media safely
- ambiguous image targets are rejected
- simple editable text boxes and shapes can be added where supported
- unsupported advanced visual edits are preserved or explicitly skipped

Verification:

```text
go test ./internal/edit ./internal/validate ./cmd/puppt
```

### Checkpoint 7: Creation Workflow

Exit evidence:

- `puppt create` builds editable decks from structured JSON
- created decks inspect and validate
- titles, body text, bullets, notes, metadata, and provided images work
- generated output is deterministic

Verification:

```text
go test ./internal/create ./internal/inspect ./internal/validate ./cmd/puppt
```

### Checkpoint 8: Review and v1 Hardening

Exit evidence:

- `puppt review` produces agent-readable and human-readable summaries
- docs describe commands, support matrix, unsupported boundaries, examples, and failure modes
- acceptance workflows run end to end on sample fixtures
- known non-v1 gaps are explicitly listed

Verification:

```text
go test ./...
```

## Progress Record Template

At the end of each checkpoint, record:

```text
Checkpoint:
Changed files:
Implemented behavior:
Verification commands:
Verification result:
Fixtures added or updated:
Known risks:
Unsupported behavior encountered:
Next checkpoint:
```

The record MAY live in a changelog, checkpoint log, issue, or final response, but it MUST be concrete enough for another engineer or agent to continue.

## Completion Audit

Before calling Puppt v1 complete, verify every acceptance criterion in `goal.md` against current evidence. The audit MUST map each requirement to a command, fixture, test, output file, or documentation section.

Minimum final audit commands:

```text
go test ./...
puppt inspect testdata/<sample>.pptx --json
puppt edit testdata/<sample>.pptx --edit testdata/edits/replace-title.json --out /tmp/puppt-title.pptx --json
puppt edit testdata/<sample>.pptx --edit testdata/edits/deck-wide-replace.json --out /tmp/puppt-wide.pptx --json
puppt edit testdata/<sample>.pptx --edit testdata/edits/slide-ops.json --out /tmp/puppt-slides.pptx --json
puppt edit testdata/<sample>.pptx --edit testdata/edits/notes.json --out /tmp/puppt-notes.pptx --json
puppt create --input testdata/create/basic-deck.json --out /tmp/puppt-created.pptx --json
puppt validate /tmp/puppt-created.pptx --json
```

If a final audit uses different fixture names, it MUST explain the mapping. Passing a narrower check is not evidence for a broader requirement.

## Known Initial State

At the time this protocol is written, the workspace contains planning documents only. There is no Go module, no product code, no fixtures, and no acceptance suite yet. The next implementation checkpoint is Checkpoint 0: Repository Foundation.
