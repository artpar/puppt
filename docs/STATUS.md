# Puppt Status

## Current State

Fixture-backed v1 checkpoint workflows are implemented. Full production-grade compliance with `swe_skill.md` is not yet claimed.

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
- `puppt render <input.pptx> --slide <N> --out <slide-N.png> --json` is wired.
- The renderer is Puppt-owned Go code that writes 72 DPI PNGs directly. It currently renders slide dimensions, solid and simplified gradient slide background color with radial interpolation, inherited master/layout parts, layout `showMasterSp="0"` master-shape suppression while preserving master inheritance data, package theme color schemes for common DrawingML scheme colors with master and override color-map aliasing, master title/center-title text-style defaults for fallback text size/color/boldness/alignment, slide placeholder text with inherited layout/master text-box properties, fallback bounds matched by placeholder type or index, inherited body/content default bullets, inherited paragraph margins/hanging indents, DrawingML default text-body insets, line spacing, fixed-point and percent-based paragraph before/after spacing, shape autofit for unrotated text boxes including no-wrap width expansion, explicit no-autofit text body state, and normal-autofit font and line-spacing reductions with bounded derived scaling whenever `a:normAutofit` is requested while preserving authored break-line counts where scaling can avoid extra soft wraps and honoring local `a:bodyPr` text anchors when present while inheriting missing placeholder anchors, embedded PNG/JPEG/GIF pictures with basic placement, source cropping including negative `srcRect` padding, quarter-turn image rotation, PNG ICC matrix/TRC and Adobe RGB (1998) JPEG source color conversion, alpha compositing, deterministic interpolated resampling selected from current golden comparison evidence, DrawingML outer shadows for supported picture geometry, Gaussian alpha-mask soft edges for rectangular pictures, and custom-geometry picture masks for single DrawingML paths made of move/line/cubic/close commands, simple SVG icon pictures from direct SVG image parts, grouped child objects with basic group transforms, style-derived fill/line/font colors with bounded DrawingML color modifiers, final sRGB-to-Display-P3 output color conversion matching the checked-in Apple Notes reference render color pipeline, solid rectangle/rounded-rectangle/triangle/ellipse/right-arrow fills, DrawingML Gaussian alpha-mask outer shadows for supported filled shape geometry, DrawingML guide-based chevron/notched-right-arrow, curved-arrow, and right-brace geometry including `darkenLess` path fill overlays, simple custom/freeform polygon fills with first-pass edge antialiasing, rectangle/ellipse/preset-polygon/custom-polygon outlines, simple connector lines including zero-width or zero-height `p:cxnSp` and triangle line-end marker sizing, first-pass shape text with paragraph breaks, bullet prefixes, auto-numbered bullets, authored bullet font families/theme bullet font references/DrawingML bullet text-font fallback, locally installed Carlito metric-compatible fonts before bundled Carlito as supported substitutes for unavailable Calibri/Calibri Light theme fonts with generic fallback reporting only when no supported substitute is available, unhinted font rasterization selected from current golden comparison evidence, shape-level italic face selection, paragraph/run-level font size, paragraph/run-level bold face selection, run-scoped text colors, run-level baseline offsets, and common Office symbol bullet drawing, with specific partial reports for unsupported vertical text, body-level text rotation, multi-column text, and anchor-center text bodies; graphic-frame table grids with authored row/column extents when they fit the graphic frame, direct cell fills/borders, parsed package table-style fills/borders/text styling, row spans/vertical merges/text, specific reports for unsupported non-solid table fills/effects/non-flat line caps/compound lines/line-end decorations, and diagram graphic-frame drawing parts using package theme colors with specific reports for unsupported non-shape diagram content. It reports unpainted or partially painted objects explicitly, including non-triangle connector line-end markers, unsupported shadow geometry, simplified effects, and unsupported picture effect/custom-geometry features.
- A gated real-world golden comparison test covers the 61 checked-in Apple Notes reference renders under `testdata/realworld-ppts/reference-renders/manual-using-apple-note`; LibreOffice exports are retained as an alternate comparison set.
- Review output includes changes, skipped/ambiguous/unsupported items from prior command results, inspection facts, and validation status.
- Command docs, support matrix, failure modes, and acceptance workflow docs exist.
- End-to-end CLI acceptance workflow is covered by tests.

## Known Gaps

- Advanced non-text object extraction, richer media metadata, and broader real-world unsupported-feature warning detection beyond the current basic warning set.
- Adding new notes parts when a slide has no notes relationship.
- Full render fidelity for text, grouped text, shapes, images, transforms, and theme/layout/master-derived styling.
- Real-world deck fixture breadth remains limited.
- General `puppt validate` does not yet accept explicit expected-content assertions.
- CI/release packaging and production rollback procedures are not implemented.

All required v1 command names are implemented: `inspect`, `plan`, `edit`, `create`, `validate`, `review`, `render`, and `version`.

## Next Work

Use `docs/HANDOFF.md` and `docs/COMPLIANCE_AUDIT.md` as the current handoff. Future work should close the known gaps above before claiming production-grade v1.
