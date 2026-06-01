# Renderer Completion Goal

This document defines the remaining build path for completing Puppt's renderer
parity work. It is scoped by `swe_skill.md`, `docs/RENDERING.md`,
`docs/RENDERER_PRODUCTION_PATH.md`,
`docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`, and the current object-attributed
investigation log.

## Goal Statement

Complete Puppt's renderer as a production-grade, source-backed Go renderer for
practical PowerPoint Open XML viewing, review, and transformation workflows. The
goal is not a perfect clone of PowerPoint, Apple Notes, LibreOffice, or any other
host renderer. The goal is a first-principles renderer: Puppt interprets OOXML
semantics into its own render model, implements each visual primitive from the
source model, and uses perceptual metrics only to validate practical
compatibility after the spec-backed implementation is in place. The primary
target is conformance to the applicable ECMA-376/Open XML and DrawingML source
semantics for the supported feature set. Metrics and reference renders are
acceptance and diagnostic tools; they are not a license to approximate
undocumented behavior or tune pixels without a source-backed model.

Puppt must continue to own `.pptx` package reading, relationship resolution,
inheritance/theme interpretation, source attribution, CLI behavior, JSON
reporting, and preservation semantics. Rendering may use bounded primitive
libraries behind Puppt-owned interfaces, but production code must not delegate
`.pptx` interpretation to LibreOffice, PowerPoint, Keynote, Apple Notes, browser
engines, SaaS renderers, or image-conversion shells. Browser-style or SVG/canvas
pipelines may be studied as references, but they are not production renderer
backends unless this goal is explicitly revised.

The completed renderer must lower resolved Open XML content into stable internal
render primitives and render those primitives through production backends for the
maintained support matrix: picture/media, vector geometry and stroke,
connectors, text shaping/layout, tables, shadows/effects, fills, diagrams where
fallback data is available, and unsupported-feature reporting. A visual fix is
complete only when the implementation is justified from the authoritative source
XML and the relevant rendering model, passes the attributed object fixture under
the structural, exact-diagnostic, and perceptual gates, does not regress
same-family neighboring fixtures, and survives the full real-world render gate.

The renderer goal is achieved only when current evidence proves all of these
requirements:

1. `go test ./...` passes.
2. The real-world render suite passes the maintained perceptual compatibility
   gate against the checked reference set.
3. Exact-pixel comparison remains available as a diagnostic and regression
   signal, but it is not the sole production acceptance criterion for real-world
   PPTX rendering.
4. `puppt render <deck> --slide N --out <png> --json` remains deterministic,
   compatible, and honest about unsupported or partial behavior.
5. Production renderer code has no office/browser/SaaS renderer dependency path.
6. Every previously tracked clean object fixture either passes its structural,
   exact-diagnostic, and perceptual gate or has an explicit, source-backed
   accepted residual recorded in the renderer docs.
7. Every renderer behavior change that closes a visual gap is proven first by an
   attributed object fixture and then by the full 61-slide no-regression gate.
8. The support matrix records each PPTX feature family as supported, partial, or
   unsupported, with source spec references, fixture coverage, and accepted
   deviations.
9. `docs/RENDERER_EXPERIMENT_LOG.md` and
   `docs/RENDERER_COMPLETION_CHECKLIST.md` record the accepted/rejected backend
   decisions, validation commands, changed files, residual risks, and next
   maintenance checkpoint.

## Spec Conformance Policy

Renderer work must match the OOXML source model first. For every supported
feature family, the maintained support matrix must identify:

- the relevant package parts, relationships, and XML elements/attributes
- the ECMA-376/Open XML schema or prose reference used for implementation
- the Puppt semantic structure that represents the resolved source behavior
- the render primitive emitted from that semantic structure
- deterministic synthetic fixtures that prove the source behavior independent of
  any host renderer
- real-world fixtures that prove practical compatibility against reference
  renders
- unsupported or partially supported clauses, with explicit reporting behavior

When the standard defines structure but not exact rasterization, Puppt must still
make a principled renderer choice and document it. Examples include antialiasing
coverage, font fallback, platform font metrics, color-management transforms,
image resampling kernels, and effect compositing. Those choices must be stable,
testable, and isolated behind Puppt-owned primitive boundaries.

No production behavior should be described as supported merely because it looks
close to a reference image. It is supported only when the source semantics are
implemented, the fixture proves that implementation, and the visual gate shows no
unaccepted practical regression.

## First-Principles Implementation Policy

Puppt must implement renderer behavior from the source document model outward:

1. OOXML package parts, relationships, themes, layouts, masters, placeholders,
   and object properties are resolved into Puppt-owned semantic structures.
2. Semantic structures are lowered into stable render primitives with explicit
   units, coordinate spaces, transforms, fills, strokes, text runs, media,
   effects, clipping, and unsupported records.
3. Primitive backends implement known rendering algorithms directly or through
   bounded primitive libraries: path filling/stroking, image sampling, color
   conversion, text shaping, font metrics, compositing, clipping, gradients,
   shadows, and table layout.
4. If the OOXML standard does not fully define a pixel behavior, the
   implementation must choose and document a renderer model such as Office-like
   DrawingML geometry, CSS/SVG-equivalent path math, HarfBuzz/OpenType shaping,
   or a named color-management transform.
5. A production change may not be accepted because a parameter search improves a
   screenshot. It must explain which source primitive or rendering model was
   wrong and why the new implementation is more correct.

Perceptual metrics do not replace this policy. They answer whether the resulting
first-principles implementation is visually compatible enough for maintained
real-world decks after source-backed correctness has been established.

## Visual Acceptance Policy

Puppt uses perceptual metrics for real-world visual compatibility and exact
pixel checks for deterministic synthetic fixtures, object diagnostics, and
regression triage. This keeps the renderer honest without pretending that every
host renderer, font rasterizer, antialiasing kernel, and color pipeline can be
made byte-identical.

The real-world gate must record at least:

- exact output dimensions
- render status and unsupported/partial counts
- differing-pixel count and changed bounds
- total and max channel delta
- whole-slide perceptual similarity
- object-crop perceptual similarity for tracked fixtures
- object ownership classification for residual differences

Initial production thresholds should be conservative and tightened only with
evidence:

- exact dimensions for every rendered slide
- no render errors for supported corpus slides
- no new unsupported/partial reports unless explicitly accepted
- no large localized object regression hidden by a good whole-slide score
- whole-slide perceptual similarity at or above the maintained project threshold
- tracked clean object fixtures at or above stricter object-level thresholds, or
  documented as accepted residuals with source-backed rationale

The threshold implementation should live in tests or test helpers, not only in
docs. Each gate failure must report enough artifact paths and object attribution
data to explain the visual miss.

## First-Principles Rule

The 61-slide visual comparison is the final gate, not the diagnostic method. The
working loop must start from source structure and end with object-level evidence:

1. Read the object's authoritative OOXML: part path, XML path, relationships,
   cNvPr id/name, transform, geometry, fill, stroke, text body, media, theme,
   layout, and master dependencies.
2. Define the expected primitive behavior from the source, using the Open XML
   model and justified Go libraries for non-core primitives when needed.
3. Render only the target object in controlled modes:
   background before the object, object only, and objects through the object.
4. Compare the object crop against the Apple Notes object reference artifact.
5. Attribute the residual to a specific primitive, such as picture contour
   coverage, crop geometry, source color conversion, shape edge coverage,
   shadow composition, text metrics, line spacing, anchoring, or clipping.
6. Define the source-backed rendering model for that primitive before editing.
7. Change only that primitive in the production renderer.
8. Accept the change only when the object fixture passes its structural and
   perceptual gate and the full corpus does not regress.

No broad font, color, table, autofit, or resampling experiment is valid unless
it starts from an attributed object failure and names the source primitive it is
testing. No perceptual score can justify a change that lacks a source-backed
primitive model.

## Build Path

### Phase 1: Stabilize The Worktree

Finish the interrupted `internal/render/render_text_styles_test.go` split until
the repo compiles again. Keep this as enabling cleanup only; it must not become
the parity strategy.

Required evidence:

- `go test ./internal/render -count=1`
- `go test ./...`
- `git diff --check`
- preserved `docs/2026-05-31-renderer-8h-investigation.md`
- preserved `docs/RENDERER_EXPERIMENT_LOG.md`

### Phase 2: Lock The Attribution And Perceptual Harness

Make the object-attributed render harness and perceptual comparison output the
default diagnostic tools for parity work. The harness must record every painted
object with enough provenance to move from a visual residual back to source XML.

Required object record fields:

- slide part and source part
- XML path
- cNvPr id and name
- object kind
- z-order
- EMU, fractional, pixel, and output bounds
- resolved fill, stroke, text, image, shadow, and unsupported summaries
- artifact paths
- whether the object painted visible output

Required render modes:

- normal production render
- background plus objects before target
- target object only on transparent or flat background
- objects through target

Required artifacts:

- per-slide got/reference/diff PNGs
- per-object PNGs
- object attribution JSON
- ownership summary JSON
- perceptual metric summary JSON
- per-object perceptual metric records for tracked fixtures

### Phase 3: Rank Real Failures By Ownership

Generate the ownership summary from current artifacts and select the smallest
clean failures before making renderer changes. A clean failure means the
differing visible pixels belong to the target object and are not confounded by
partial-alpha underpaint or later-object occlusion.

Current leading failures from the investigation log:

- WHO HIV slide 015 `Picture 4`: 1200 differing pixels, clean picture contour
  coverage residual.
- EPA Residential Wood slide 004 `Google Shape;11;p15`: 2127 differing pixels,
  same picture contour coverage family.
- WHO HIV slide 012 `Rectangle 5`: 7423 visible differing pixels, clean shape
  target with stroke/fractional edge and centered-text vertical placement
  components.
- WHO HIV slide 015 `TextBox 7`: 19868 differing pixels, mixed fill height,
  color, text, and antialias residual.

Do not advance to a new broad renderer topic until the selected object has:

- a focused manifest
- source XML preserved in the fixture
- visible crop/reference/diff artifacts
- a residual profile tied to source fields
- an experiment-log entry saying what was accepted or rejected

### Phase 4: Complete Micro-Fixture Extraction

Each failing object must be reducible to a deterministic fixture containing the
target object and only required dependencies: theme, layout, master, media,
relationships, and any source parts necessary to preserve renderer semantics.

Acceptance target:

- object crop/reference artifact, not the whole slide
- object-level structural and perceptual metrics, not only differing-pixel count

Fixture requirements:

- preserve raw source XML for the target object unless a documented extractor
  transformation is proven equivalent
- preserve relationship ids where meaningful
- preserve theme/color-map dependencies
- preserve media bytes and image metadata
- include visible-mask handling for later-object occlusion
- keep generated artifacts deterministic

### Phase 5: Fix Renderer Primitives In Failure-Family Order

Implement production fixes only after a primitive is source-backed and fixture
isolated.

Initial failure-family order:

1. Opaque grayscale picture contour coverage.
   Evidence: `Picture 4` and `Google Shape;11;p15` both show grayscale
   edge-coverage residuals where the current render keeps too many hard pixels.
   Generic kernel, gamma, area, phase, palette-model, and contour-threshold
   searches have been rejected because they do not pass the object fixtures.
2. Rectangle shape edge and stroke coverage.
   Evidence: `Rectangle 5` has full-width top/bottom and side-column residuals
   plus fractional edge/stroke symptoms.
3. Centered text vertical placement and font metrics.
   Evidence: `Rectangle 5` text is about four pixels low, but a broad text shift
   does not pass the fixture and is not acceptable alone.
4. TextBox fill height, text antialias, and anchor behavior.
   Evidence: `TextBox 7` improves with fill/height normalization but remains a
   mixed residual, so no fill-only or font-only change is justified.

For each primitive:

- add or tighten a micro-fixture test first
- inspect the current production render path before editing
- implement the smallest coherent source-backed change
- run the focused object fixture
- run the relevant neighboring fixtures in the same failure family
- run the full real-world perceptual and no-regression gate before accepting the
  change
- record the decision in `docs/RENDERER_EXPERIMENT_LOG.md`

### Phase 6: Preserve CLI And JSON Contracts

Renderer parity work must not make the command-line interface less honest.
Unsupported content must be preserved where possible and reported when relevant.
JSON output must remain deterministic and additive.

Required evidence:

- focused `puppt render ... --json` check on at least one supported render
- focused `puppt render ... --json` check on one deck with known partial or
  unsupported reporting
- no removal or incompatible renaming of existing JSON fields

### Phase 7: Final Completion Audit

Before claiming completion, audit every explicit requirement rather than relying
on passing narrow tests.

Final evidence packet:

- `go test ./...`
- `git diff --check`
- real-world reference gate passing all 61 slides under the maintained
  perceptual thresholds
- exact-pixel diagnostic summary recorded for the same run
- object fixture suite passing structural and perceptual gates for all
  previously tracked clean failures
- external renderer grep over production Go code
- `puppt render ... --json` stability checks
- updated support matrix with supported, partial, unsupported, and accepted
  residual entries
- updated `docs/RENDERER_EXPERIMENT_LOG.md`
- a summary of changed files, verification results, residual risks, and next
  maintenance checkpoint

## Non-Goals

- Do not claim full OOXML or PowerPoint pixel-perfect completeness from the
  current 61-slide corpus.
- Do not use Apple Notes, LibreOffice, PowerPoint, Keynote, browsers, SaaS APIs,
  or image converters as production renderers.
- Do not tune against whole-slide pixel totals before object attribution.
- Do not land a renderer change because it improves one inspected crop.
- Do not hide localized object failures behind a passing whole-slide perceptual
  score.
- Do not hide uncertainty in JSON or human output.
- Do not replace Puppt's owned `.pptx` package reader/writer or mutation path
  with a dependency that obscures preservation.

## Dependency Rule

Third-party Go libraries are allowed and encouraged where they replace weak
non-core primitives: text shaping, font metrics, color management, geometry
rasterization, clipping, resampling, and SVG parsing. They must be isolated
behind Puppt-owned interfaces and justified by the primitive they improve.

Dependencies that read, write, render, mutate, validate, or interpret `.pptx`
content remain controlled dependencies and require explicit justification.
