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

- Slide master inspection is not populated yet.
- Full media classification and advanced non-text object extraction are not complete.
- Unsupported-feature warning detection is not complete.

Unsupported behavior encountered:

- Rich real-world package constructs beyond simple notes, images, layouts, and core metadata remain unclassified.

Next checkpoint:

- Continue Checkpoint 2 with slide master refs, full media classification, advanced object extraction, and broader unsupported-feature warning detection.

## Checkpoint 2: Inspection Core, progress 5

Changed files:

- `internal/model/inspection.go`
- `internal/pptx/reader.go`
- `internal/fixtures/pptx.go`
- `internal/inspect/inspect.go`
- `internal/inspect/inspect_test.go`
- `internal/inspect/testdata/minimal.golden.json`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Added slide `master` and `master_name` fields.
- Added broader slide `media` field while preserving existing `images` field.
- Added media kind and extension classification.
- Added relationship constants for slide masters, audio, video, and OLE objects.
- Extended fixtures with audio, video, OLE object, and slide master package relationships.
- Resolved slide master references through slide layout relationships.
- Parsed slide master names.
- Classified image, audio, video, and OLE object relationships.
- Added slide-level warning for OLE object relationships.
- Updated golden output to include the additive `media` field.

Verification commands:

```text
go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt
go test ./...
```

Verification result:

- `go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt` passed.
- `go test ./...` passed.

Fixtures added or updated:

- Extended `internal/fixtures.PPTXOptions` and `fixtures.Slide` for master, audio, video, and OLE data.
- Updated `internal/inspect/testdata/minimal.golden.json`.

Known risks:

- Media metadata is limited to relationship type, content type, target, and extension.
- Advanced non-text object extraction is not complete.
- Unsupported-feature warning detection is still intentionally conservative.

Unsupported behavior encountered:

- Embedded object internals, media dimensions/durations, chart internals, SmartArt internals, and master-inherited placeholders remain incomplete.

Next checkpoint:

- Continue Checkpoint 2 with advanced object extraction, richer unsupported-feature warning coverage, and inspection contract review before moving to targeting/planning.

## Checkpoint 2: Inspection Core, completion audit

Requirement: `puppt inspect --json` returns stable JSON.

Evidence:

- `internal/cli/root.go` wires `puppt inspect <input.pptx> --json`.
- `internal/cli/root_test.go` verifies JSON envelope fields and slide text through the CLI.
- `internal/report.WriteJSON` emits deterministic indented JSON.

Status: achieved for the Checkpoint 2 scope.

Requirement: slide order is represented.

Evidence:

- `internal/pptx.Open` resolves presentation slide order through `presentation.xml` slide IDs and presentation relationships.
- `internal/inspect.Inspect` emits ordered `slides`.
- `TestInspectReturnsSlideOrderAndVisibleText` verifies order.

Status: achieved.

Requirement: titles and visible text are represented.

Evidence:

- `internal/inspect.shapeTextBlocks` extracts text by shape and stable object ID.
- `internal/inspect.Inspect` derives simple titles from the first visible text block.
- `TestInspectReturnsSlideOrderAndVisibleText` and `minimal.golden.json` verify shape-level visible text and titles.

Status: achieved for text shapes. Advanced non-text object extraction remains a known later enhancement.

Requirement: notes are represented.

Evidence:

- `internal/inspect.inspectSlideRelationships` resolves notes slide relationships.
- `TestInspectReturnsMetadataNotesImagesAndLayout` verifies speaker-note text.

Status: achieved.

Requirement: media refs are represented.

Evidence:

- `internal/inspect.inspectSlideRelationships` classifies image, audio, video, and OLE object relationships.
- `TestInspectReturnsMetadataNotesImagesAndLayout` verifies media refs and image content type/extension.

Status: achieved for relationship-level media references. Rich media metadata remains a known later enhancement.

Requirement: metadata is represented.

Evidence:

- `internal/inspect.inspectMetadata` reads core properties.
- `TestInspectReturnsMetadataNotesImagesAndLayout` verifies title, author, and subject.

Status: achieved for core metadata.

Requirement: layouts are represented.

Evidence:

- `internal/inspect.inspectSlideRelationships` resolves slide layout refs.
- `internal/inspect.parseCommonSlideName` extracts layout names.
- `TestInspectReturnsMetadataNotesImagesAndLayout` verifies layout ref and name.

Status: achieved.

Requirement: repeated content is represented.

Evidence:

- `internal/inspect.repeatedText` emits repeated visible text counts.
- `TestInspectReportsRepeatedVisibleText` verifies count.

Status: achieved.

Requirement: warnings are represented.

Evidence:

- Inspection emits `inspection_partial` while advanced object extraction remains incomplete.
- `inspectPackageWarnings` emits warnings for macros, charts, and diagrams.
- `inspectSlideRelationships` emits OLE object warnings.
- `TestInspectWarnsForUnsupportedPreservedParts` verifies package warnings.

Status: achieved for the current warning categories. Broader real-world warning detection remains a known later enhancement.

Requirement: golden fixture tests exist.

Evidence:

- `internal/inspect/testdata/minimal.golden.json`
- `TestInspectGoldenJSON`

Status: achieved.

Checkpoint 2 decision:

- Complete enough to enter Checkpoint 3: Targeting and Edit Planning.
- Remaining inspection improvements are tracked as known risks and should be added as fixtures reveal real-world gaps.

## Checkpoint 3: Targeting and Edit Planning, progress 1

Changed files:

- `internal/model/result.go`
- `internal/model/plan.go`
- `internal/target/doc.go`
- `internal/target/resolve.go`
- `internal/target/resolve_test.go`
- `internal/edit/doc.go`
- `internal/edit/plan.go`
- `internal/edit/plan_test.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Added edit spec, target spec, edit plan, and target match models.
- Added target resolution against inspection facts.
- Supported target types: slide number, title, visible text, object ID, notes by slide number, and metadata property.
- Added ambiguity detection before mutation.
- Added deck-scope override for multi-match visible text plans.
- Added no-match classification.
- Added non-mutating `internal/edit.Plan`.
- Wired `puppt plan <input.pptx> --edit <edit.json> --json`.
- Made ambiguous/no-match CLI plan results write JSON and return non-zero.

Verification commands:

```text
go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli
go test ./...
```

Verification result:

- `go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli` passed.
- `go test ./...` passed.

Fixtures added or updated:

- Reused deterministic generated `.pptx` fixtures.
- Added inline edit specs in tests.

Known risks:

- Plan schema is still internal/pre-v1 and may need command examples before freezing.
- Planning does not yet validate operation-specific required fields beyond operation and target type.
- No mutation engine exists yet.

Unsupported behavior encountered:

- Edit application remains explicitly unsupported.

Next checkpoint:

- Continue Checkpoint 3 with operation-specific planning validation, skipped/unsupported operation reporting, and more target tests before mutation workflows.

## Checkpoint 3: Targeting and Edit Planning, progress 2

Changed files:

- `internal/edit/plan.go`
- `internal/edit/plan_test.go`
- `internal/cli/root_test.go`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Added operation-specific target validation.
- Supported planning operations: `replace_text`, `update_notes`, `update_metadata`, `replace_image`, `slide_add`, `slide_delete`, `slide_move`, and `slide_duplicate`.
- Rejected unsupported operations before inspection/mutation.
- Rejected operation/target mismatches.
- Rejected missing required replacement for text and notes planning.
- Rejected missing metadata property for metadata planning.
- Added unsupported-result JSON and non-zero CLI behavior.

Verification commands:

```text
go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli
go test ./...
```

Verification result:

- `go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli` passed.
- `go test ./...` passed.

Fixtures added or updated:

- Reused deterministic generated `.pptx` fixtures.
- Added inline unsupported and missing-field edit specs in tests.

Known risks:

- Slide operation specs still need operation-specific fields for destination/position before mutation work starts.
- Image replacement planning currently targets object IDs but image object targeting needs stronger image-object identity tests.
- Plan examples are not documented for users yet.

Unsupported behavior encountered:

- Edit application remains explicitly unsupported.

Next checkpoint:

- Continue Checkpoint 3 with documented plan examples, image object target identity, and slide-operation plan fields before mutation workflows.

## Checkpoint 3: Targeting and Edit Planning, progress 3

Changed files:

- `internal/model/plan.go`
- `internal/target/resolve.go`
- `internal/target/resolve_test.go`
- `internal/edit/plan.go`
- `internal/edit/plan_test.go`
- `docs/PLAN_EXAMPLES.md`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Added plan fields for `image_path`, `insert_after_slide`, and `destination_slide_number`.
- Included those fields in emitted edit plans.
- Resolved media object IDs for image targets.
- Added image replacement plan tests.
- Added image-path required-field validation.
- Added slide-move destination planning test.
- Added agent-facing plan examples for text replacement, deck-wide replacement, notes, metadata, image replacement, slide move, and slide duplicate.

Verification commands:

```text
go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli
```

Verification result:

- `go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli` passed.

Fixtures added or updated:

- Reused deterministic generated `.pptx` fixtures.
- Added inline image and slide operation edit specs in tests.

Known risks:

- Slide add/delete/duplicate planning does not yet validate all future mutation fields.
- Image replacement planning resolves media object IDs, but replacement image validation is not implemented yet.
- The mutation engine still does not exist.

Unsupported behavior encountered:

- Edit application remains explicitly unsupported.

Next checkpoint:

- Complete Checkpoint 3 audit or add any remaining target coverage needed before mutation workflows.

## Checkpoint 3: Targeting and Edit Planning, completion audit

Changed files:

- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Exit evidence:

- Targets by slide number, title, visible text, object ID, notes, metadata, and image refs are covered by `internal/target` and `internal/edit` tests.
- Ambiguity and no-match are detected before mutation by `internal/target.Resolve` and surfaced by `puppt plan --json`.
- `puppt plan --json` reports intended changes, skipped targets, ambiguous targets, unsupported operation/target pairs, and required-field failures.
- Image replacement planning resolves media object IDs and records `image_path` without mutating the package.

Verification commands:

```text
go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli
go test ./...
```

Verification result:

- `go test ./internal/target ./internal/edit ./cmd/puppt ./internal/cli` passed.
- `go test ./...` passed.

Known risks:

- Edit application remains unsupported.
- Replacement image validation is not implemented yet.
- Slide add/delete/duplicate planning may require additional fields once the mutation contract is finalized.

Unsupported behavior encountered:

- Mutation workflows are explicitly deferred to Checkpoint 4.

Next checkpoint:

- Continue Checkpoint 4 with text, notes, and metadata mutations plus post-edit validation.

## Checkpoint 4: Text, Notes, and Metadata Mutations, completion audit

Changed files:

- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `internal/edit/apply.go`
- `internal/edit/apply_test.go`
- `internal/edit/plan.go`
- `internal/pptx/writer.go`
- `internal/target/resolve.go`
- `internal/validate/validate.go`
- `internal/validate/validate_test.go`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Wired `puppt edit <input.pptx> --edit <edit.json> --out <output.pptx> --json`.
- Added Puppt-owned PPTX package writing that rewrites the ZIP from owned package parts.
- Applied supported mutations only after successful planning and unambiguous target resolution.
- Rejected resolved target kinds that do not match the requested operation, such as text replacement against an image object ID.
- Implemented targeted text replacement by stable shape object ID.
- Implemented deck-wide visible-text replacement with exact per-object match counts in changes.
- Implemented speaker-note update for slides with existing notes relationships.
- Implemented core metadata updates for `title`, `author`, and `subject`.
- Added structural validation after edit output is written.
- Added post-edit content verification through inspection for text, notes, and metadata.
- Kept planned-but-not-safe mutation operations explicitly unsupported in `puppt edit`.

Verification commands:

```text
go test ./internal/edit ./internal/validate ./cmd/puppt ./internal/cli
go test ./...
git diff --check
```

Verification result:

- `go test ./internal/edit ./internal/validate ./cmd/puppt ./internal/cli` passed.
- `go test ./...` passed.
- `git diff --check` passed.

Fixtures added or updated:

- Added edit tests using deterministic fixture decks for text, deck-wide replacement, notes, metadata, unsupported image mutation, and unrelated media preservation.
- Added validation tests for valid fixture decks and missing relationship targets.

Known risks:

- Mutated XML parts are re-encoded; unrelated package parts are preserved as part bytes, but changed XML parts are not byte-for-byte preserved.
- Notes update requires an existing notes relationship; adding a notes part to a slide without notes is not implemented yet.
- Metadata update requires an existing core properties part.
- Structural validation is intentionally basic and will need to deepen as slide operations and image replacement are added.

Unsupported behavior encountered:

- Slide add/delete/move/duplicate mutations remain unsupported.
- Image replacement mutation remains unsupported.
- Simple shape/textbox additions remain unsupported.

Next checkpoint:

- Continue Checkpoint 5 with slide add, delete, move, and duplicate mutation workflows plus relationship validation.

## Checkpoint 5: Slide Operations, completion audit

Changed files:

- `internal/edit/apply.go`
- `internal/edit/apply_test.go`
- `internal/edit/plan.go`
- `internal/edit/plan_test.go`
- `internal/edit/slide.go`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Implemented `slide_add` by adding a new editable slide part, presentation relationship, slide ID entry, and content-type override.
- Implemented `slide_delete` by removing the slide from presentation order, presentation relationships, slide part storage, slide relationships, and slide content-type overrides.
- Implemented `slide_move` by reordering the presentation slide ID list without changing slide parts.
- Implemented `slide_duplicate` by copying the slide part and slide relationships to a new slide part, then adding presentation relationship/order entries and a content-type override.
- Kept all slide mutations behind the same planning and unambiguous target resolution path as other edits.
- Added change summaries that identify touched slide positions and slide part IDs.

Verification commands:

```text
go test ./internal/edit ./internal/inspect ./internal/validate ./cmd/puppt
go test ./...
git diff --check
```

Verification result:

- `go test ./internal/edit ./internal/inspect ./internal/validate ./cmd/puppt` passed.
- `go test ./...` passed.
- `git diff --check` passed.

Fixtures added or updated:

- Added edit tests using deterministic fixture decks for slide add, delete, move, and duplicate.
- Slide-operation tests verify output slide order/content through inspection and relationship validity through validation.
- Added planning validation for missing `insert_after_slide` on slide duplication.

Known risks:

- Slide add currently creates a simple editable text slide from `replacement`; richer layout-aware slide creation is deferred to the creation checkpoint.
- Slide duplicate copies slide relationships as-is, so duplicated slides may share media targets with the source slide.
- Slide delete removes slide package parts and references but does not garbage-collect orphaned related media parts yet.
- XML parts touched by slide ordering/content-type updates are re-encoded.

Unsupported behavior encountered:

- Image replacement mutation remains unsupported.
- Simple shape/textbox additions remain unsupported.

Next checkpoint:

- Continue Checkpoint 6 with explicit image target replacement and simple editable additions.

## Checkpoint 6: Image Replacement and Simple Additions, completion audit

Changed files:

- `internal/edit/apply.go`
- `internal/edit/apply_test.go`
- `internal/edit/media.go`
- `internal/edit/plan.go`
- `internal/edit/plan_test.go`
- `internal/target/resolve.go`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`
- `docs/PLAN_EXAMPLES.md`

Implemented behavior:

- Implemented `replace_image` mutation for explicit image object IDs and image selectors that resolve to exactly one image.
- Replaced package media bytes in the existing image target part while preserving slide text and package relationship structure.
- Added `image` target resolution so ambiguous image targets are rejected before mutation.
- Implemented `add_text_box` for adding a simple editable text object to a target slide.
- Implemented `add_shape` for adding a simple editable rectangle shape with editable text to a target slide.
- Kept advanced visual edits outside the supported operation set, so they remain rejected before mutation.

Verification commands:

```text
go test ./internal/edit ./internal/validate ./cmd/puppt
go test ./...
git diff --check
```

Verification result:

- `go test ./internal/edit ./internal/validate ./cmd/puppt` passed.
- `go test ./...` passed.
- `git diff --check` passed.

Fixtures added or updated:

- Added edit tests using deterministic fixture decks for image replacement, ambiguous image planning, editable text-box addition, and editable shape addition.

Known risks:

- Image replacement preserves the existing package target and content type; replacing an image with a different file type is not normalized yet.
- Added text boxes and shapes use simple deterministic XML without layout positioning controls.
- Advanced visual edits remain outside v1 mutation support.

Unsupported behavior encountered:

- Deck creation is not implemented yet.
- Review summaries are not implemented yet.

Next checkpoint:

- Continue Checkpoint 7 with structured JSON deck creation, deterministic output, inspection, and validation.

## Checkpoint 7: Creation Workflow, completion audit

Changed files:

- `internal/create/create.go`
- `internal/create/create_test.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`
- `docs/CREATE_EXAMPLES.md`
- `README.md`

Implemented behavior:

- Wired `puppt create --input <deck.json> --out <output.pptx> --json`.
- Added a structured JSON creation contract with deck metadata and ordered slides.
- Created deterministic editable `.pptx` packages using Puppt-owned package writing.
- Supported `title`, `section`, and `title_body` slide layouts.
- Supported title text, body text, bullet lists, speaker notes, metadata, and local image paths.
- Added deterministic slide layout and master parts for created decks.
- Ran validation and inspection-based content checks after writing created decks.

Verification commands:

```text
go test ./internal/create ./internal/inspect ./internal/validate ./cmd/puppt ./internal/cli
go test ./...
git diff --check
```

Verification result:

- `go test ./internal/create ./internal/inspect ./internal/validate ./cmd/puppt ./internal/cli` passed.
- `go test ./...` passed.
- `git diff --check` passed.

Fixtures added or updated:

- Added creation tests for structured JSON with metadata, title slides, section slides, title/body slides, bullets, notes, and images.
- Added deterministic output test that compares two generated decks byte-for-byte.
- Added CLI JSON creation test.

Known risks:

- Created decks use simple deterministic layouts and basic shape XML rather than design-rich PowerPoint layouts.
- Created image support preserves the provided bytes and relationship target but does not normalize image dimensions.
- Creation validates expected content through current inspection coverage, so advanced visual fidelity remains outside current verification.

Unsupported behavior encountered:

- Review summaries are not implemented yet.
- Acceptance fixture workflows and final support matrix hardening remain for Checkpoint 8.

Next checkpoint:

- Continue Checkpoint 8 with review summaries, command docs, support matrix, acceptance workflows, and explicit v1 gap listing.

## Checkpoint 8: Review and v1 Hardening, completion audit

Changed files:

- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `internal/review/doc.go`
- `internal/review/review.go`
- `internal/review/review_test.go`
- `docs/COMMANDS.md`
- `docs/SUPPORT_MATRIX.md`
- `docs/FAILURE_MODES.md`
- `docs/ACCEPTANCE.md`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`
- `README.md`

Implemented behavior:

- Wired `puppt validate <input.pptx> --json`.
- Wired `puppt review <input.pptx> --changes <changes.json> --json`.
- Review accepts prior `puppt.v1` command results or raw change arrays.
- Review emits changes, skipped, ambiguous, unsupported, inspection facts, and validation status.
- Added end-to-end CLI acceptance test covering create, inspect, edit, validate, and review.
- Added command docs, support matrix, failure modes, acceptance workflow docs, and known non-v1 gap listing.

Verification commands:

```text
go test ./internal/review ./internal/cli ./internal/validate ./cmd/puppt
go test ./...
git diff --check
```

Verification result:

- `go test ./internal/review ./internal/cli ./internal/validate ./cmd/puppt` passed.
- `go test ./...` passed.
- `git diff --check` passed.

Fixtures added or updated:

- Added review tests for prior command result changes and raw change arrays.
- Added CLI JSON tests for `validate` and `review`.
- Added CLI acceptance workflow test.

Known risks:

- Validation remains structural plus workflow-specific expected-content checks; it does not perform rendered visual comparison.
- Review summarizes supplied change artifacts; it does not compute a semantic diff between two decks.
- Advanced object/media extraction remains intentionally limited to current v1 coverage.

Unsupported behavior encountered:

- Preview rendering, legacy `.ppt`, macro/VBA editing, chart editing, SmartArt editing, and design-rich visual fidelity checks remain documented non-v1 gaps.

Next checkpoint:

- v1 checkpoint sequence is complete; future work should start from a new scoped goal or from the documented non-v1 gaps.

## Checkpoint 2: Inspection Core, progress 4

Changed files:

- `internal/model/inspection.go`
- `internal/fixtures/pptx.go`
- `internal/inspect/inspect.go`
- `internal/inspect/inspect_test.go`
- `internal/inspect/testdata/minimal.golden.json`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Added `layout_name` to slide inspection output.
- Parsed layout names from slide layout parts.
- Added deterministic extra package parts to fixtures.
- Added basic unsupported-part warnings for macro projects, chart parts, and diagram/SmartArt parts.
- Added tests for layout names and unsupported-part warnings.

Verification commands:

```text
go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt
```

Verification result:

- `go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt` passed.

Fixtures added or updated:

- Extended `internal/fixtures.PPTXOptions` with `ExtraParts`.
- Updated `internal/inspect/testdata/minimal.golden.json`.

Known risks:

- Slide master references are not populated yet.
- Full media classification and advanced non-text object extraction are not complete.
- Unsupported-feature warning detection is intentionally basic.

Unsupported behavior encountered:

- Real-world embedded media, OLE objects, chart internals, SmartArt internals, and master-derived layout behavior remain incomplete.

Next checkpoint:

- Continue Checkpoint 2 with slide master refs, full media classification, advanced object extraction, and broader unsupported-feature warning detection.

## Checkpoint 2: Inspection Core, progress 3

Changed files:

- `internal/fixtures/pptx.go`
- `internal/inspect/inspect.go`
- `internal/inspect/inspect_test.go`
- `internal/inspect/testdata/minimal.golden.json`
- `docs/STATUS.md`
- `docs/DOCTRINE_CHECKLIST.md`
- `docs/CHECKPOINTS.md`

Implemented behavior:

- Added non-visual shape properties to deterministic slide and notes fixtures.
- Extracted visible text at shape level where `p:sp` and `p:cNvPr` are available.
- Generated stable text object IDs from slide part and PowerPoint shape ID.
- Applied the same shape-level extraction to speaker notes where available.
- Updated golden JSON to pin shape-level object IDs.

Verification commands:

```text
go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt
```

Verification result:

- `go test ./internal/inspect ./internal/model ./internal/report ./cmd/puppt` passed.

Fixtures added or updated:

- Updated generated slide and notes XML fixtures with `p:cNvPr` IDs/names.
- Updated `internal/inspect/testdata/minimal.golden.json`.

Known risks:

- Slide master inspection is not populated yet.
- Full media classification and advanced non-text object extraction are not complete.
- Unsupported-feature warning detection is not complete.

Unsupported behavior encountered:

- Group shapes, pictures as objects, charts, SmartArt, embedded media details, and master-derived layout names remain incomplete.

Next checkpoint:

- Continue Checkpoint 2 with slide master/layout naming, full media classification, advanced object extraction, and unsupported-feature warning detection.
