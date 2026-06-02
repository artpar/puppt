# Renderer Production Path

This document answers how Puppt gets from the current renderer to a
production-ready renderer. It is subordinate to `swe_skill.md`,
`docs/RENDERING.md`, and `docs/RENDERER_COMPLETION_GOAL.md`.

## Current Evidence Snapshot

Generated from the current 2026-06-01 object-debug artifacts:

```text
PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 \
PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard.json \
go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v
```

Result:

```text
slides: 61
total slide differing pixels: 9,321,023
object attribution artifacts: 61
clean object-fixture failures: 70
```

Top object-overlap primitive groups:

```text
shape geometry/fill/line/clipping/antialiasing: 7,564,625 overlap pixels across 176 objects
text shaping/font metrics/paragraph layout/anchoring: 3,470,321 overlap pixels across 288 objects
picture crop/resampling/color/media transform: 2,355,753 overlap pixels across 168 objects
connector geometry/fill/line/clipping/antialiasing: 1,250,753 overlap pixels across 34 objects
table layout/inherited table text styling: 814,498 overlap pixels across 9 objects
shape shadow geometry/blur/alpha/transform: 428,045 overlap pixels across 24 objects
```

The overlap totals are diagnostic and may exceed full-slide pixels because
object bounds overlap. Clean fixture totals are isolated object-crop evidence:

```text
picture clean failures: 46 fixtures, 1,499,584 differing pixels
shape clean failures: 24 fixtures, 550,448 differing pixels
```

Largest clean fixture families:

```text
Picture 2: 5 fixtures, 382,408 differing pixels
Picture 5: 2 fixtures, 171,420 differing pixels
TextBox 7: 2 fixtures, 152,863 differing pixels
Rectangle 5: 9 fixtures, 111,487 differing pixels
Rectangle 3: 9 fixtures, 109,530 differing pixels
```

## Conclusion

The current hand-built raster primitives are no longer the shortest path to a
production renderer. The evidence does not support landing more isolated pixel
tweaks: several source-backed candidates improved narrow diagnostics but failed
object fixtures. Completion now requires bounded production primitive backends
for vector geometry, text shaping/layout, and image sampling/color, while
preserving Puppt's owned Open XML package parser and mutation path.

The required architecture change is a renderer scene boundary. The renderer
must stop sending resolved `slideElement` values directly into handwritten paint
functions as the long-term design. Instead, each resolved OOXML object must be
lowered into a stable Puppt-owned render primitive first. Backends then consume
those primitives. This keeps `.pptx` interpretation in Puppt and makes the
replaceable parts only image, vector, text, table, and effect primitives.

The production path is:

1. Keep Puppt's `.pptx` package reader, relationship resolver, source XML
   attribution, fixture extraction, CLI, and JSON reporting under Puppt control.
2. Lower resolved DrawingML objects into a stable internal render scene:
   `Picture`, `PathShape`, `TextBox`, `Connector`, `Table`, `Shadow`, and
   unsupported-feature records.
3. Introduce small internal primitive interfaces for graphics, text, and image
   sampling. These interfaces consume render-scene primitives and return pixels
   or metrics; they must not read, write, render, or interpret `.pptx` packages
   directly.
4. Prove each backend one primitive family at a time with attributed object
   fixtures before replacing current production behavior.
5. Accept a backend only when the focused fixture passes, same-family neighbors
   do not regress, the 61-slide Apple Notes gate does not regress, and CLI/JSON
   reporting remains honest.

First code boundary started on 2026-06-01:

- `internal/render/render_scene.go` defines the initial render-scene and picture
  primitive structs.
- `renderPicturePrimitiveFromElement` lowers a resolved picture `slideElement`
  into a source-backed picture primitive preserving object identity, source
  relationship, media part/content type, integer/fractional target bounds,
  DrawingML crop percentages, flip, `alphaModFix`, rotation, `rotWithShape`,
  soft edge, custom mask, and line metadata.
- Because the existing production `renderPicture` path also handles shape-level
  blip fills, the primitive records the original object kind and accepts
  picture-backed `pic`, `sp`, and `cxnSp` elements.
- `internal/render/render_scene_test.go` proves this first boundary preserves
  those fields and reports unresolved picture relationships as conversion
  errors.
- `renderPicture` now lowers the object into `renderPicturePrimitive` and calls
  `currentPictureBackend` through a `pictureBackend` interface. This is a
  zero-diff migration stage.
- `pictureBackendInput` no longer carries the legacy resolved `slideElement`.
  The primitive now owns the paint contract for object kind, SVG fallback
  relationship id, crop, flip, `alphaModFix`, rotation/`rotWithShape`, soft
  edge, custom mask path/commands/unsupported messages, line style, shadow
  parameters, and 3-D unsupported feature metadata.
- `currentPictureBackend` calls `pictureSamplingStage`. The default
  `currentPictureSamplingStage` preserves existing sampling/soft-edge/custom
  mask/rotation behavior, and tests can inject an alternate stage before it is
  accepted as production.

## Dependency Boundary

Allowed dependency role:

- pure Go graphics, text, color, and rasterization primitives behind Puppt-owned
  interfaces.

Forbidden dependency role:

- any library or process that reads, writes, renders, mutates, validates, or
  interprets `.pptx` packages on Puppt's behalf without an explicit controlled
  dependency decision.

Current dependency checks:

```text
go list -m -json github.com/go-text/typesetting@latest
  version: v0.3.4
  time: 2026-02-25
  go version: 1.19

go list -m -json github.com/tdewolff/canvas@latest
  version: v0.0.0-20260508100355-63a7228e682d
  time: 2026-05-08
  go version: 1.25.0

go list -m -json github.com/llgcode/draw2d@latest
  version: v0.0.0-20260422081035-c4331ac66734
  time: 2026-04-22
  go version: 1.24.0

go list -m -json github.com/srwiley/rasterx@latest
  version: v0.0.0-20220730225603-2ab79fcdd4ef
  time: 2022-07-30
  go version: 1.17
```

Official package docs checked on 2026-06-01:

- `github.com/go-text/typesetting/harfbuzz`:
  https://pkg.go.dev/github.com/go-text/typesetting/harfbuzz
- `github.com/go-text/typesetting/shaping`:
  https://pkg.go.dev/github.com/go-text/typesetting/shaping
- `github.com/tdewolff/canvas`:
  https://pkg.go.dev/github.com/tdewolff/canvas
- `github.com/llgcode/draw2d`:
  https://pkg.go.dev/github.com/llgcode/draw2d

Current dependency decision:

- `github.com/go-text/typesetting` is the first candidate for a text shaping
  backend because it provides HarfBuzz-style OpenType shaping primitives without
  owning `.pptx` interpretation.
- `github.com/tdewolff/canvas` is not an immediate dependency while Puppt stays
  on Go 1.24.3 because the latest module advertises Go 1.25.0.
- `github.com/llgcode/draw2d` is the immediate vector backend candidate to test
  under the current Go toolchain. It must first be used in a diagnostic backend
  for existing shape fixtures, not directly in production.
- `github.com/srwiley/rasterx` is older and should be considered only if
  `draw2d` cannot support the required DrawingML paths/strokes under the current
  toolchain.

## Implementation Sequence

### Track A: Vector Geometry Backend

Why first: the scoreboard's largest object-overlap group is shape and connector
geometry/line/antialiasing.

First checkpoint:

1. Add a test-only `draw2d` shape rasterization backend for rectangles and
   simple polygons behind an internal interface.
2. Feed it the already parsed DrawingML geometry for `Rectangle 5`,
   `Rectangle 3`, and one connector fixture.
3. Compare backend output against the existing object fixture references.
4. Accept production replacement only if it passes a focused object fixture and
   same-family neighbors.

Do not land a backend just because aggregate channel error improves.

### Track B: Text Shaping And Layout Backend

Why second: text is the second-largest object-overlap group and current
diagnostics show hand-tuned line-box and redraw hypotheses do not explain the
fixture residuals.

First checkpoint:

1. Add a test-only text shaping adapter using `github.com/go-text/typesetting`.
2. Shape a single parsed run with the same resolved font file used by Puppt's
   current font resolver.
3. Emit glyph IDs, advances, offsets, and bounding boxes beside the current
   `golang.org/x/image/font` metrics.
4. Run the adapter on `Rectangle 5` and `TextBox 7` fixtures before changing
   production text rendering.

### Track C: Picture Sampling And Color Backend

Why third: clean failures are currently picture-heavy, but broad resampling,
gamma, metadata, and phase searches have already been rejected. The next image
work must replace the picture primitive backend through the render-scene
boundary, not run another kernel search.

First checkpoint:

1. Lower picture objects into `renderPicturePrimitive`.
2. Define a `pictureBackend` interface that consumes `renderPicturePrimitive`
   plus decoded media bytes and produces a raster layer plus explicit
   unsupported-stage records.
3. Move the existing picture renderer behind that backend interface without
   changing pixels; this parity-preserving migration gate is complete as of
   2026-06-01.
4. Remove the transitional legacy `slideElement` dependency from
   `pictureBackendInput` by promoting remaining picture paint fields into the
   primitive. This is complete as of 2026-06-01.
5. Replace the image sampling/color implementation behind the backend with a
   fixture-proven implementation. The replaceable sampling-stage boundary is
   present as of 2026-06-01. The opt-in replacement acceptance gate is also
   present as `TestCurrentPictureSamplingStageAcceptanceGate`; it renders the
   two named picture fixtures through the staged backend and currently records
   the expected unresolved residuals of 1,200 and 2,127 pixels. The actual
   replacement remains open.
6. Re-run `Picture 4`, `Google Shape;11;p15`, and the top clean `Picture 2`
   fixtures through that staged pipeline.
7. Accept a production image change only if a named stage explains the focused
   and neighbor fixture residuals.

### Track D: Tables And Diagrams

Why later: table overlap is large, but the clean-fixture scoreboard currently
shows picture and shape fixtures as the highest isolated failures.

First checkpoint:

1. Generate table-specific clean fixtures and source summaries for the largest
   table overlap objects.
2. Do not change table layout from whole-slide overlap alone.

## Completion Gates

The renderer is production-ready only when:

1. `go test ./...` passes.
2. `git diff --check` passes.
3. The real-world Apple Notes gate passes:
   `PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1`.
4. The tracked object fixture suite passes for all previously clean failures.
5. The renderer dependency grep confirms no office/browser/SaaS/image-conversion
   renderer path.
6. `puppt render ... --json` checks prove deterministic, additive, honest CLI
   output.
7. `docs/RENDERER_EXPERIMENT_LOG.md` records accepted/rejected decisions with
   commands, changed files, residual risks, and next checkpoints.
