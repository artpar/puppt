# Renderer Experiment Log

This log keeps renderer parity work tied to attributed object failures. The
61-slide real-world comparison is the final gate, not the diagnostic loop.

## Rejected Broad Paths

The 2026-05-31 investigation in `docs/2026-05-31-renderer-8h-investigation.md`
recorded broad experiments that did not produce accepted parity progress:

- Fractional EMU placement changes.
- `spAutoFit` resize suppression.
- Table-border alpha changes.
- Broad image resampling changes.
- Broad font/color/autofit/table adjustments not tied to a specific object
  failure.

Do not repeat these as corpus-wide experiments. New renderer changes need an
object-level failure, a micro-fixture target, and a full-corpus regression check.

## 2026-06-01 M01 Scope, Gates, And Ledger Closeout

Milestone: `docs/renderer-milestones/01-scope-gates-and-ledger.md`.

Objective: freeze the static renderer scope, status promotion policy, fixture
and perceptual gate policy, final evidence packet, and unsupported-content
reporting rule before further renderer primitive work.

Accepted decisions:

- The renderer target remains static PresentationML/DrawingML rendering from
  OOXML source semantics, not host-renderer pixel cloning.
- Coverage status changes must follow explicit promotion rules for Supported,
  Partial, Unsupported, Out of renderer scope, and Unimplemented / no evidence.
- Perceptual metrics are validation evidence only; production changes still
  require a named schema row, source XML object, render primitive, and fixture.
- Unsupported visible content must be preserved where possible and reported
  explicitly, or rejected before mutation.

Rejected decisions:

- No renderer primitive, pixel, font, color, sampling, or layout behavior was
  changed in M01.

Verification:

```text
python3 tools/generate_ooxml_drawingml_audit.py
```

Result: passed; regenerated `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` with
status promotion rules.

```text
git diff --check
```

Result: passed.

Changed files:

- `tools/generate_ooxml_drawingml_audit.py`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/RENDERER_COMPLETION_GOAL.md`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`
- `docs/RENDERER_EXPERIMENT_LOG.md`

Residual risk: M01 freezes scope and accounting policy only; it does not make
fixture queues, perceptual metrics, or all-clean-fixture execution complete.
Next checkpoint: M02 fixtures, metrics, and work queues.

## 2026-06-01 M02 Fixtures, Metrics, And Work Queues Closeout

Milestone: `docs/renderer-milestones/02-fixtures-metrics-and-work-queues.md`.

Objective: make the proof system executable before additional renderer
implementation work: schema-row queues, deterministic spec-fixture metadata,
object-attributed clean-fixture execution, perceptual metrics, and current
real-world no-regression evidence.

Accepted decisions:

- Coverage accounting now emits both the human matrix and
  `docs/renderer-coverage-summary.json`.
- Matrix rows are assigned to five work queues: `core-static`,
  `common-partial`, `hard-rendering`, `unsupported-preserve`, and
  `out-of-scope`.
- Micro-fixture manifests now have a formal `spec_fixture` block containing
  schema anchors, source XML part/path, expected semantic model, expected
  render primitive, and expected unsupported records.
- Exact pixel diffs remain the diagnostic gate, while additive deterministic
  luma/RGB-RMS perceptual metrics are recorded for slide and object comparisons.
- The all-clean fixture suite can rerender every currently tracked clean object
  fixture in one command and produce a JSON summary.

Rejected decisions:

- No renderer primitive, pixel, font, color, layout, sampling, fixture
  rebaseline, or support-status promotion was accepted in M02.
- The clean fixture suite was run in expected-failure accounting mode; the 70
  current failures are not waived as renderer passes.

Verification:

```text
python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

rg -n '<a:(alphaOutset|reflection)\b' testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 -g 'source-object.xml' -g '*.xml'

go test ./... -count=1

git diff --check

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result: passed; queue totals are 16 `core-static`, 90 `common-partial`, 383
`hard-rendering`, 444 `unsupported-preserve`, and 74 `out-of-scope`.

```text
go test ./internal/render -run 'TestMicroFixture|TestRendererProductionFailureScoreboard' -count=1
```

Result: passed.

```text
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 \
PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 \
PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-current.json \
go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v
```

Result: passed; 70 tracked clean fixtures ran, 0 passed and 70 failed as the
expected current renderer state.

```text
PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 \
PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-current.json \
go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v
```

Result: passed; 61 slides measured, 61 differing slides, mean luma similarity
0.950955502, mean channel-RMS similarity 0.829145432, and 9,321,023 total
differing pixels.

```text
PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 \
PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-current.json \
go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v
```

Result: passed; 61 slides, 9,321,023 total slide differing pixels, 8 object
groups, and 70 clean fixture failures.

```text
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1

# post-revert validation
python3 tools/generate_ooxml_drawingml_audit.py --print-summary
go test ./internal/render -count=1
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v
go test ./...
git diff --check
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1
```

Result: expected failure; 61/61 slides differ from the Apple Notes references,
total differing pixels are 9,321,023, the worst slide is
`EPA-generate-2021-presentation.pptx` slide 001 with 308,113 differing pixels,
and top unsupported rendering gaps are `none`.

```text
git diff --check
```

Result: passed.

Changed files:

- `tools/generate_ooxml_drawingml_audit.py`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/renderer-coverage-summary.json`
- `internal/render/render_m02_test.go`
- `internal/render/render_realworld_test.go`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`
- `docs/RENDERER_EXPERIMENT_LOG.md`

Residual risk: the proof system is executable, but current renderer parity is
still unresolved: all 61 real-world slides differ and all 70 currently tracked
clean object fixtures fail. Next checkpoint: M03 render scene IR.

## 2026-06-01 M03 Render Scene IR Closeout

Milestone: `docs/renderer-milestones/03-render-scene-ir.md`.

Objective: finish the Puppt-owned render scene boundary so supported and
partial PresentationML/DrawingML objects lower into stable primitives with
source provenance, schema anchors, relationship ids, and backend-swap seams
before later primitive rendering work.

Accepted decisions:

- `renderSceneFromElements` now lowers pictures, picture-backed shape/connector
  blip fills, shapes, connectors, graphic frames, tables, diagrams, groups, and
  unsupported objects.
- Every primitive carries provenance: object kind, cNvPr id/name, source part,
  XML path, relationship ids where applicable, and schema anchors.
- Shape primitives carry geometry, path, fill, stroke, text, and effect
  sub-primitives. Graphic-frame primitives carry table and diagram sub-primitives.
- Scene lowering runs before the existing paint loop as a zero-diff boundary.
- Existing picture rendering remains behind `renderPicturePrimitive`; M03 adds
  primitive-consuming interfaces for shape, connector, and graphic-frame
  backend replacement in later milestones.

Rejected/deferred decisions:

- No visual renderer primitive, pixel, font, color, sampling, layout, or
  reference baseline change was accepted in M03.
- Non-picture backend migration is intentionally deferred. The matrix records
  these rows as Partial because legacy shape, connector, table, diagram, text,
  and effect paint paths still produce pixels.

Verification:

```text
python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

rg -n '<a:effectDag\b' testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 -g 'source-object.xml' -g '*.xml'

go test ./... -count=1

git diff --check

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result: passed; regenerated the coverage matrix and
`docs/renderer-coverage-summary.json` with M03 primitive-lowering evidence.

```text
go test ./internal/render -run 'TestRenderScene|TestRender.*Primitive|TestRenderPicture|TestRenderShape|TestRenderGraphicFrame' -count=1
```

Result: passed.

```text
go test ./internal/render -count=1
```

Result: passed.

```text
git diff --check
```

Result: passed.

Changed files:

- `internal/render/render_scene.go`
- `internal/render/render_scene_test.go`
- `internal/render/render_paint.go`
- `tools/generate_ooxml_drawingml_audit.py`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/renderer-coverage-summary.json`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`
- `docs/RENDERER_EXPERIMENT_LOG.md`

Residual risk: the scene IR exists, but non-picture primitive families still
paint through legacy functions. This is deliberate M03 scope control; backend
replacement and geometry correctness start in later milestones. Next checkpoint:
M04 coordinates, transforms, and clipping.

## 2026-05-31 Object Attribution Harness

Added a debug/test-only object attribution path. When
`PUPPT_REALWORLD_ARTIFACT_DIR` is set, failing real-world slides now write:

- `objects.json`: every painted object with slide part, source part, cNvPr
  id/name, kind, z-order, EMU bounds, integer pixel bounds, fractional pixel
  bounds, output pixel bounds, and a resolved style summary.
- `objects/*-object.png`: each object isolated on transparent or flat debug
  background.
- `objects/*-before.png`: the cumulative slide render immediately before that
  object paints, used to inspect underpaint without adding other objects to an
  extracted fixture.
- `objects/*-through.png`: cumulative render through that object.
- `object-attribution.json`: object-level overlap counts against the full
  slide diff, cumulative through-X diff counts, a z-order cumulative probe
  timeline, the largest absolute cumulative-diff jump, a binary-search z-order
  probe, observed diff text, and a suspected renderer gap.
- `micro-fixtures/<object>/`: for the top attributed picture object when its
  media dependency is resolvable, a tiny PPTX containing only that object and
  its media part, plus `got-crop.png`, `reference-crop.png`, and
  `micro-diff.json`. Fixture renders also write `fixture-objects.json` and
  `fixture-objects/*` debug PNGs so fixture-local before/object/through state
  can be compared with source-deck diagnostics.
- `micro-fixtures/shape-<object>/`: for the top attributed shape object, a
  tiny PPTX containing the raw source `<p:sp>` object XML plus stripped
  layout/master dependencies and required theme parts, with the same crop,
  diff, and manifest artifacts. Dependency layout/master parts keep
  backgrounds and placeholder sources but remove non-placeholder drawable
  objects so the fixture remains object-scoped. Shape fixtures also preserve
  the actual slide `<p:bg>` when present, falling back to the source part
  background only when the slide has no direct background, so inherited
  layout/master objects render against the same background as the source slide.
- Shape fixture emission also follows earlier intersecting shape underpaints
  from the top attributed shape and writes `underpaint-shape-<object>`
  micro-fixtures. This makes underpaint contamination executable instead of
  leaving it as a diagnostic note on the later target.
- Occluded object fixtures also write `got-visible-crop.png`,
  `reference-visible-crop.png`, and `visible-micro-diff.json`. These mask
  pixels covered by later z-order objects, because a raw reference crop can
  include unrelated objects that an isolated fixture cannot reproduce. The
  occlusion records include `mask_padding_pixels=1` so visible crops also mask
  the antialias fringe around later objects.
- Fixtures also write `got-geometry-crop.png`,
  `reference-geometry-crop.png`, and `geometry-diff.json` when object geometry
  bounds are known. These use the full DrawingML-derived pixel bounds rather
  than the changed-output alpha bounds, which is important for pictures with
  white or transparent margins.
- Fixtures also write source-deck diagnostic crops when object debug artifacts
  are available: `source-before-crop.png` from the cumulative render before the
  target object and `source-through-crop.png` plus `source-through-diff.json`
  from the cumulative render through the target object. These are diagnostics
  for underpaint and subtraction; they are not fixture acceptance targets.
  Occluded targets also get `source-through-visible-crop.png` and
  `source-through-visible-diff.json`, using the same later-object mask as the
  executable visible fixture target.
- Picture fixture manifests include `source_image` metadata with the resolved
  package part, decoded format, width, and height.
- Picture fixture manifests also include `sampling` metadata with the integer
  geometry target size, fractional geometry size and offset, output-crop offset,
  and source-to-geometry scale factors.
- Each extracted micro-fixture also writes `target-scope.json` and embeds the
  same data in `manifest.json`. This diagnostic splits crop differences into
  pixels inside the current object artifact alpha mask and pixels outside it.
  It also splits object-mask pixels into full-alpha and partial-alpha pixels.
  Partial-alpha object-mask pixels are further bucketed into low (`1..80`),
  mid (`81..200`), and high (`201..254`) alpha ranges, which separates faint
  shadow fringes from stronger edge coverage.
  It is diagnostic only, not an acceptance mask: outside-mask differences flag
  likely background, earlier-object, or crop-contamination issues, while
  partial-alpha differences flag edge pixels whose reference value may include
  underpaint.
- Micro-fixture manifests also record `underpainted_by` for earlier z-order
  object artifacts whose output bounds intersect the target. `target-scope.json`
  counts how many partial-alpha target pixels and partial-alpha differences sit
  over earlier object mask pixels, separating earlier-object underpaint from
  plain background blending.

Latest artifact run:

```text
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 \
PUPPT_REALWORLD_ARTIFACT_DIR=testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31 \
go test ./internal/render -run TestRealWorldGoldenComparison -count=1
```

Result: expected failure, `61/61` slides differed with `9321023` total
differing pixels. Worst slide was
`testdata/realworld-ppts/EPA-generate-2021-presentation.pptx` slide `001` with
`308113` differing pixels. The run produced `61` `object-attribution.json`
files, `41` extracted picture micro-fixtures, `20`
largest-cumulative-delta `cumulative-picture-*` fixtures, `61` top shape
micro-fixtures, `27` largest-cumulative-delta `cumulative-shape-*` fixtures,
`14` largest-cumulative-delta `cumulative-connector-*` fixtures, and `7`
additional `underpaint-shape-*` fixtures where a top shape target had earlier
intersecting shape underpaint. All `61` object attribution files now include
`cumulative_probes` and `largest_cumulative_delta`; the worst slide's largest
cumulative jump is z-order `1`, cNvPr id `7`, name `Freeform 6`, with `315850`
cumulative-delta pixels. Representative connector largest-cumulative fixture
verification:
`PUPPT_MICRO_FIXTURE_MANIFEST=.../EPA-metal-coil-NESHAP-2018/slide-001/micro-fixtures/cumulative-connector-0001-32-Straight-Connector-31/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1`
passes exactly, which is a useful control showing that a large cumulative
through-X jump can come from missing later objects/background in the cumulative
comparison rather than an object-local renderer failure. Representative
picture largest-cumulative fixture verification:
`PUPPT_MICRO_FIXTURE_MANIFEST=.../EPA-residential-wood-MacCarty/slide-001/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1`
fails with `3069` visible-crop differing pixels.

All `14` cumulative connector fixtures were then verified after extraction
and all passed exactly:

```text
find ... -path '*/micro-fixtures/cumulative-connector-*/manifest.json' | while read manifest; do
  PUPPT_MICRO_FIXTURE_MANIFEST="$manifest" go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
done
```

Those `14` connector objects accounted for `2008889` summed
largest-cumulative-delta pixels across their slides, with max single-slide
delta `165718`, but none are object-local fixture failures. Treat connector
largest-cumulative hits as cumulative-context controls unless a connector
fixture itself fails.

All `20` cumulative picture fixtures were also verified after extraction and
all failed:

```text
find ... -path '*/micro-fixtures/cumulative-picture-*/manifest.json' | while read manifest; do
  PUPPT_MICRO_FIXTURE_MANIFEST="$manifest" go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
done
```

The `20` picture largest-cumulative objects accounted for `2244100` summed
largest-cumulative-delta pixels and `171317` summed fixture-diff pixels. The
worst picture fixture is
`WHO-HIV-testing-algorithms-toolkit/slide-010/micro-fixtures/cumulative-picture-0001-3-Picture-2/manifest.json`,
which fails with `64711` crop differing pixels. That object is a `2112x892`
PNG (`ppt/media/image9.png`) rendered into a `691x337` output crop; all
`64711` differences are inside the current object mask. Treat picture
largest-cumulative hits as real object-level picture targets until their
fixtures pass.

## Attributed Failures

### EPA Generate Slide 001, Freeform 6 Largest Cumulative Delta

```text
deck: testdata/realworld-ppts/EPA-generate-2021-presentation.pptx
slide: 001
object: cNvPr id=7 name="Freeform 6"
kind: sp
z-order: 1
XML part/path: ppt/slides/slide1.xml
geometry: custom path trapezoid, fill=#FFFFFF/FF no_line=true
outer shadow: color #000000/66, blur=127000 EMU, distance=63500 EMU, direction=1800000, alignment=tl, rotateWithShape=0
source custom path: (0,0) -> (2039815,0) -> (2963007,923192) -> (0,923192) within 2963007x923192
object attribution: largest cumulative jump on worst slide, 315850 cumulative-delta pixels
micro-fixture: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-001/micro-fixtures/cumulative-shape-0001-7-Freeform-6/fixture.pptx
visible micro-fixture crop diff: 6607 / 22448 crop pixels, bounds x=0..243 y=0..91
target scope: 5042 differing pixels inside current object mask, 1565 outside; 5042 inside partial-alpha mask; 4970 inside low-alpha dark partial-alpha mask
shadow alpha scope: 4445 analyzed shadow pixels; reference alpha greater for 3704 pixels, reference alpha less for 741 pixels, absolute alpha delta 21996
shadow render summary: target points (0,27), (160,27), (233,99), (0,99); shadow points (4,29), (164,29), (237,101), (4,101); blur=10 px
verifier: `PUPPT_MICRO_FIXTURE_MANIFEST=.../cumulative-shape-0001-7-Freeform-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1` fails with 6607 visible-crop differing pixels
```

This is now the executable target for the largest cumulative-delta object on
the worst current slide. The source XML and renderer-derived shadow geometry
match the same trapezoid/shadow family already seen on EPA slide 007, but this
fixture is not cleanly object-only yet: `1565` differing pixels are outside
the current object artifact alpha mask. Treat it as a valid subtraction target
for the dominant shadow failure, while keeping that outside-mask warning in
view before accepting any renderer change.

#### EPA Generate Slide 001, Freeform 6: opt-in shadow phase/composite search

Ran the existing shadow phase and full composite diagnostics against the
newly extracted largest-cumulative-delta fixture:

```text
PUPPT_SHADOW_PHASE_SEARCH_MANIFEST=.../cumulative-shape-0001-7-Freeform-6/manifest.json \
PUPPT_SHADOW_PHASE_SEARCH_OUTPUT=/tmp/slide001-shadow-phase-search.json \
go test ./internal/render -run TestMicroFixtureShadowPhaseSearch -count=1 -v

PUPPT_SHADOW_COMPOSITE_SEARCH_MANIFEST=.../cumulative-shape-0001-7-Freeform-6/manifest.json \
PUPPT_SHADOW_COMPOSITE_SEARCH_OUTPUT=/tmp/slide001-shadow-composite-search.json \
go test ./internal/render -run TestMicroFixtureShadowCompositeSearch -count=1 -v
```

Result against `cumulative-shape-0001-7-Freeform-6`:

```text
phase baseline: 4454 differing alpha pixels, abs alpha delta 21293, signed alpha delta +3155
best phase candidate: shiftX=-0.25 shiftY=-0.25 sampleX=0 sampleY=0.5
best phase result: 4132 differing alpha pixels, abs alpha delta 18685, signed alpha delta -775

visible fixture diff: 6607 / 22448 pixels
current composite baseline: 6602 differing pixels, abs channel delta 65370, signed RGB delta -4062
best composite candidate: shiftX=0.5 shiftY=-0.5 sampleX=0.5 sampleY=0
best composite result: 6435 differing pixels, abs channel delta 58542, signed RGB delta +7762
best direction split: reference-darker 3883, reference-lighter 2545
```

Decision: rejected as a renderer change. The same simple shadow phase family
that only weakly helped slide 007 also fails this largest cumulative target.
It improves the full composite by only `167` pixels and flips the signed RGB
direction positive, while the object-level fixture remains far from passing
and still has the outside-mask contamination noted above.

### WHO HIV Testing Algorithms Slide 015, Picture 4

```text
deck: testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx
slide: 015
object: cNvPr id=1028 name="Picture 4"
kind: pic
z-order: 9
XML part/path: ppt/slides/slide15.xml
geometry pixel bounds: x=677..788 y=360..470
changed-output pixel bounds: x=699..788 y=360..451
source media: ppt/media/object.png, 200x200 PNG, rendered into a 112x111 geometry target with a 90x92 changed-output crop
sampling: fractional target 111.789055x111.789055 pixels; fractional offset -0.192677,-0.374173 from the integer geometry crop; source-to-geometry scale 0.558945x0.558945
micro-fixture: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/fixture.pptx
micro-fixture crop diff: 1200 / 8280 crop pixels differ from reference-crop.png
target scope: 1200 differing pixels inside the current object artifact mask, 0 outside
verifier: `PUPPT_MICRO_FIXTURE_MANIFEST=.../0009-1028-Picture-4/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1` fails with 1200 crop differing pixels
```

This is the current valid small picture target. The target-scope diagnostic
shows the crop is covered by the picture object, so the mismatch is not the
same background/underpaint contamination seen in no-fill or heavily occluded
shape fixtures. Visual inspection and probe output indicate the mismatch is in
image antialiasing/resampling details, not a simple object placement offset.
The geometry crop records the full picture target; the changed-output crop is a
sub-rectangle because the source icon contains white margins.

#### WHO HIV Testing Algorithms Slide 015, Picture 4: opt-in resampling search

Added an opt-in diagnostic that reconstructs only the picture crop from the
fixture's embedded PNG, then searches `x/image/draw` scaler choices and small
target endpoint rounding variants against `reference-crop.png`:

```text
PUPPT_PICTURE_RESAMPLE_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_RESAMPLE_SEARCH_OUTPUT=/tmp/picture-resample-search.json \
go test ./internal/render -run TestMicroFixturePictureResampleSearch -count=1 -v
```

Result against the master `Picture 4` micro-fixture:

```text
current baseline: approx_bilinear/round, target offset x=-22..89 y=0..110, 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
best simple candidate: bilinear/floor_ceil, target offset x=-23..89 y=-1..111, 1156 differing pixels, abs channel delta 108759, signed RGB delta +43725
best direction split: reference-darker 421, reference-lighter 735
catmull_rom/round scored 1203 differing pixels but lower abs channel delta 83250
```

Decision: rejected as a renderer change. The diagnostic reproduced the
fixture's current 1200-pixel mismatch exactly, so it is measuring the right
object path. However, no tested scaler/endpoint variant makes the object
fixture pass, and the lowest pixel-count candidate worsens absolute channel
error and color-direction bias. A future picture fix needs a more specific
source-backed sampling/color explanation than a global scaler swap.

### EPA Generate Slide 007, Rectangle 9

```text
deck: testdata/realworld-ppts/EPA-generate-2021-presentation.pptx
slide: 007
object: cNvPr id=7 name="Rectangle 9"
kind: sp
z-order: 6
XML part/path: ppt/slides/slide7.xml
geometry pixel bounds: x=365..959 y=0..539
fractional pixel bounds: x=365.280000..960 y=0..540
resolved style: rect fill=#C8CACA/FF no_line=true
micro-fixture: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-007/micro-fixtures/shape-0006-7-Rectangle-9/fixture.pptx
occlusion: later Rounded Rectangle 9 and Picture 2 overlap the crop; visible crops mask those bounds with 1px antialias padding
visible micro-fixture crop diff after preserving the source slide background and occlusion padding: 540 / 321300 crop pixels
source-through visible crop diff after the same occlusion mask: 536 / 321300 crop pixels
target scope: 540 differing pixels inside the current object artifact mask, 0 outside; object artifact inspection shows the remaining left-edge column is partial alpha (184/255)
underpaint: earlier z-order Freeform 6 intersects x=365..959 y=19..110; target-scope counts 92 partial-alpha differing pixels over that earlier object mask and 448 partial-alpha differing pixels without earlier object underpaint
verifier: `PUPPT_MICRO_FIXTURE_MANIFEST=.../shape-0006-7-Rectangle-9/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1` fails with 540 visible-crop differing pixels
```

This was the current clean shape-edge target from the ownership summary. After
masking the later occluder antialias fringe, the remaining mismatch is only the
left fractional edge column of the solid rectangle (`different_bounds x=0..0 y=0..539`). The
new underpaint diagnostic shows that only part of that edge is over an earlier
object; most remaining edge differences are over background. Preserving the
source slide background removed a fixture-extraction error where the target
rectangle edge was blending over an inherited layout/master background instead
of the source slide background. A shape fill-edge change must account for both
background and earlier-object blending at the partial-alpha edge, make this
object-level fixture pass, and then survive the full corpus gate before it can
be accepted. The executable fixture intentionally remains target-object-only;
injecting the earlier underpaint object was tested and rejected because it
introduced that object's separate renderer mismatch into the Rectangle 9 target.
The source-deck `source-through-visible-crop.png` confirms the same direction:
including the rendered underpaint path differs by 536 pixels, while the current
source-background-preserving isolated fixture differs by 540 pixels.

### EPA Generate Slide 007, Freeform 6 Master Underpaint

```text
deck: testdata/realworld-ppts/EPA-generate-2021-presentation.pptx
slide: 007
object: cNvPr id=7 name="Freeform 6"
kind: sp
z-order: 1
XML part/path: ppt/slideMasters/slideMaster1.xml
geometry pixel bounds: x=0..232 y=27..98
changed-output pixel bounds: x=0..243 y=19..110
style: custom geometry, white fill, no line, outer shadow
micro-fixture: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/fixture.pptx
visible micro-fixture diff: 2368 / 22448 crop pixels
source-through visible diff: 2368 / 22448 crop pixels
target scope: 2366 differing pixels inside the partial-alpha object mask, 2 outside
verifier: `PUPPT_MICRO_FIXTURE_MANIFEST=.../underpaint-shape-0001-7-Freeform-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1` fails with 2368 visible-crop differing pixels
```

This is now a valid small custom-geometry/shadow target. The executable fixture
and source-through diagnostic match exactly after inherited shape fixtures were
changed to prefer the actual slide background over the source master/layout
background. The remaining mismatch is overwhelmingly inside the object's
partial-alpha mask, so the next renderer probe should focus on custom polygon
shadow/fill edge coverage rather than slide background extraction.

### EPA Generate Slide 007, Freeform 6 Layout Underpaint

```text
deck: testdata/realworld-ppts/EPA-generate-2021-presentation.pptx
slide: 007
object: cNvPr id=7 name="Freeform 6"
kind: sp
z-order: 3
XML part/path: ppt/slideLayouts/slideLayout2.xml
geometry pixel bounds: x=177..959 y=27..98
changed-output pixel bounds: x=173..959 y=19..110
style: custom geometry, white fill, no line, outer shadow
micro-fixture: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0003-7-Freeform-6/fixture.pptx
visible micro-fixture diff: 2572 / 72404 crop pixels
source-through visible diff: 2518 / 72404 crop pixels
target scope: 1642 differing pixels inside this object mask, 930 outside; 988 partial-alpha differences sit over the earlier master Freeform 6 underpaint
```

This is useful as a chained diagnostic but not yet as clean as the z-order 1
master fixture. Fix or characterize the master underpaint first, then re-check
this layout underpaint and finally Rectangle 9.

### EPA Generate Slide 001, Picture 9

```text
deck: testdata/realworld-ppts/EPA-generate-2021-presentation.pptx
slide: 001
object: cNvPr id=10 name="Picture 9"
kind: pic
z-order: 8
XML part/path: ppt/slides/slide1.xml
output pixel bounds: x=587..807 y=307..453
observed diff: 31649 full-slide diff pixels overlap this object's output bounds; object artifact paints 32487 pixels
suspected renderer gap: picture crop, resampling, color management, or media transform
artifact: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-001/object-attribution.json
micro-fixture: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-001/micro-fixtures/0008-10-Picture-9/fixture.pptx
micro-fixture parts: 7 total, including ppt/slides/slide1.xml and ppt/media/object.png
micro-fixture crop diff: 31649 / 32487 crop pixels differ from reference-crop.png
```

Next step: extract this picture object and only its required slide/media/theme
dependencies into a deterministic micro-fixture. Done for this picture object.
Its acceptance target is now the reference crop/object artifact, not the whole
slide. The fixture still fails and should be the next renderer-fix target.

### EPA Generate Slide 001, Title 1

```text
deck: testdata/realworld-ppts/EPA-generate-2021-presentation.pptx
slide: 001
object: cNvPr id=2 name="Title 1"
kind: sp
z-order: 4
XML part/path: ppt/slides/slide1.xml
output pixel bounds: x=127..835 y=165..256
resolved text style: font_family="Calibri Light", font_size=6000, paragraph_font_size=4400, bold=true, text_color=#0070C0/FF
observed diff: 28172 full-slide diff pixels overlap this object's output bounds; object artifact paints 8256 pixels
suspected renderer gap: text shaping, font metrics, paragraph layout, or text anchoring
artifact: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-001/object-attribution.json
micro-fixture: testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/EPA-generate-2021-presentation/slide-001/micro-fixtures/shape-0004-2-Title-1/fixture.pptx
micro-fixture parts: 11 total, including ppt/slides/slide1.xml, stripped ppt/slideLayouts/slideLayout1.xml, stripped ppt/slideMasters/slideMaster1.xml, and ppt/theme/theme1.xml
micro-fixture crop diff: 28370 / 65228 crop pixels differ from reference-crop.png
occlusion: later z-order Picture 3 overlaps x=396..581 y=165..205 in the crop target
visible micro-fixture crop diff: 23709 / 65228 crop pixels differ after masking later-object occlusion
target scope: visible crop has 6617 differing pixels inside the current object artifact mask and 17092 outside it
local font probe: no Calibri Light file was found in the renderer's checked local roots or Office cloud cache on this machine; source inspection indicates the fixture therefore renders via the Carlito substitute path
verifier: `PUPPT_MICRO_FIXTURE_MANIFEST=.../shape-0004-2-Title-1/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1` fails with 23709 visible-crop differing pixels
```

This is now an extracted shape/text micro-fixture. It currently fails its crop
target and is a secondary candidate for a renderer fix. Use the visible crop
target for acceptance because the raw crop includes a later overlapping picture.
Do not start another broad font experiment from this record alone; any
text/layout change must make this object-level fixture pass and must not
regress the full corpus.

The fixture source proves that the inherited master `titleStyle` for this object
has `a:defRPr sz="4400" b="1"` with `+mj-lt`, resolved by the theme to Calibri
Light. The next text-renderer investigation should start from this object-level
bold/major-font path and the executable verifier above.

Rejected object-scoped font probe: forcing `Calibri Light:bold` to
`Carlito-Bold.ttf` with `PUPPT_FONT_MAP` worsened the executable verifier from
`23709` to `27332` visible-crop differing pixels. Do not land or repeat this as
a fallback-font change.

## Rejected Object-Scoped Experiments

### EPA Generate Slide 001, Picture 9: switch PNG scaling to `draw.BiLinear`

Current renderer path for non-YCbCr PNG pictures uses
`golang.org/x/image/draw.ApproxBiLinear`. The extracted `Picture 9` object is
a `676x449` PNG rendered into a `221x147` crop. A narrow candidate switched the
non-YCbCr scaler to `draw.BiLinear`.

Result against the `Picture 9` micro-fixture:

```text
current fixture crop diff: 31649 / 32487 pixels
candidate fixture crop diff: 30791 / 32487 pixels
candidate fixture crop MAE: 0.0187585
```

Decision: rejected and reverted. It improves the attributed object slightly but
does not pass the object-level fixture, so it is not acceptable under the new
rule and must not be treated as renderer progress.

### EPA Generate Slide 001, Picture 9: replace scaler with `imaging`
filters

A temporary probe compared `github.com/disintegration/imaging` filters against
the same extracted object target. The best candidate was Gaussian/B-spline with
the existing Display P3 output transform.

Result against the `Picture 9` micro-fixture:

```text
current fixture crop diff: 31649 / 32487 pixels
best imaging candidate: gaussian p3
best candidate crop diff: 30447 / 32487 pixels
best candidate crop MAE: 4.6279 8-bit channel levels
```

Decision: rejected. A third-party resampler is allowed by policy, but this
probe still does not pass the object-level fixture and therefore is not enough
to add a dependency or change production picture rendering.

### EPA Generate Slide 001, Picture 9: fractional image edge coverage

The object's DrawingML bounds are fractional in output pixels:

```text
x: 586.8000 .. 807.8328
y: 306.6772 .. 453.5172
```

The reference crop's bottom-right pixel is blended toward the slide background,
while the current renderer paints the integer target rectangle fully opaque.
A temporary probe rendered the picture with fractional target coverage over a
white background and the existing Display P3 output transform.

Result against the `Picture 9` micro-fixture:

```text
current fixture crop diff: 31649 / 32487 pixels
candidate fixture crop diff: 31477 / 32487 pixels
candidate fixture crop MAE: 3.9269 8-bit channel levels
```

Decision: rejected for now. Fractional image edge coverage is a real observed
gap, but this candidate does not pass the object-level fixture. Do not land it
alone; revisit only as part of a fixture-passing picture reconstruction change.

### WHO HIV Testing Algorithms Slide 015, Picture 4: external resize
filters

The extracted source image is a `200x200` PNG rendered into a `90x92` crop. A
probe resized the source with ImageMagick filters and compared those crops to
the Apple Notes reference crop.

Result against the `Picture 4` micro-fixture:

```text
current fixture crop diff: 1200 / 8280 pixels
point: 2837 differing pixels
box: 2914 differing pixels
catrom: 2975 differing pixels
mitchell: 3070 differing pixels
triangle: 3116 differing pixels
cubic/spline: 3230 differing pixels
```

Decision: rejected. The tested filters are worse than the current renderer and
do not pass the object-level fixture.

### WHO HIV Testing Algorithms Slide 015, Picture 4: integer placement
shift

A narrow probe composited the current `got-crop.png` over a white `90x92`
canvas at integer offsets from `-3..3` in both axes and compared each result to
the reference crop.

Best shifted result:

```text
dx=-1 dy=3: 1529 differing pixels
current fixture crop diff: 1200 / 8280 pixels
```

Decision: rejected. Integer offset does not explain this object failure; the
best shifted crop is worse than the current unshifted renderer crop.

### WHO HIV Testing Algorithms Slide 015, Picture 4: fractional affine
placement

The object's authored EMU bounds land on fractional output pixels:

```text
x0=676.807322835 y0=359.625826772
x1=788.596377953 y1=471.414881890
width=111.789055118 height=111.789055118
```

A narrow probe rendered the `200x200` source into the `112x111` geometry crop
using ImageMagick `AffineProjection` with those fractional offsets and compared
both the full geometry crop and the changed-output crop to the Apple Notes
reference.

Best changed-output crop result:

```text
current fixture crop diff: 1200 / 8280 pixels
fractional affine triangle: 1184 / 8280 pixels
fractional affine box: 1178 / 8280 pixels
```

Decision: rejected. Fractional placement appears relevant and reduced the
Picture 4 crop mismatch slightly, but it still does not pass the object-level
fixture. Do not land a production image-placement change from this probe alone.

### WHO HIV Testing Algorithms Slide 015, Picture 4: source model, transfer, kernel, and area searches

Ran the focused picture source-model, transfer/gamma, kernel, and area
diagnostics against the selected `Picture 4` fixture and the neighboring EPA
`Google Shape;11;p15` picture fixture:

```text
PUPPT_PICTURE_GAMMA_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_GAMMA_SEARCH_OUTPUT=/Users/artpar/workspace/code/puppt/.../0009-1028-Picture-4/picture-gamma-search.json \
go test ./internal/render -run TestMicroFixturePictureGammaSearch -count=1 -v

PUPPT_PICTURE_KERNEL_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_KERNEL_SEARCH_OUTPUT=/Users/artpar/workspace/code/puppt/.../0009-1028-Picture-4/picture-kernel-search.json \
go test ./internal/render -run TestMicroFixturePictureKernelSearch -count=1 -v

PUPPT_PICTURE_AREA_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_AREA_SEARCH_OUTPUT=/Users/artpar/workspace/code/puppt/.../0009-1028-Picture-4/picture-area-search.json \
go test ./internal/render -run TestMicroFixturePictureAreaSearch -count=1 -v

PUPPT_PICTURE_SOURCE_MODEL_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_SOURCE_MODEL_SEARCH_OUTPUT=/Users/artpar/workspace/code/puppt/.../0009-1028-Picture-4/picture-source-model-search.json \
go test ./internal/render -run TestMicroFixturePictureSourceModelSearch -count=1 -v
```

All four commands passed when rerun with absolute output paths. The same four
diagnostics also passed for:

```text
.../EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json
```

Result against `Picture 4`:

```text
baseline source model: converted_icc/approx_bilinear/round = 1200 differing pixels
best source model: converted_icc/bilinear/floor_ceil = 1156 differing pixels
best transfer/gamma: converted_icc/srgb_byte/bilinear/floor_ceil = 1156 differing pixels
best kernel: converted_icc/cubic_sharp/floor_ceil = 1119 differing pixels
best area: converted_icc/area_gamma_20/floor_ceil = 1202 differing pixels
source variants: converted ICC and raw PNG Paletted/RGBA/NRGBA all preserve 39 unique colors
```

Result against neighboring EPA `Google Shape;11;p15`:

```text
baseline source model: converted_icc/approx_bilinear/round = 2127 differing pixels
best source model: converted_icc/approx_bilinear/round = 2127 differing pixels
best transfer/gamma: converted_icc/gamma_24/approx_bilinear/round = 2119 differing pixels
best kernel: converted_icc/box_0_5/round = 2121 differing pixels
best area: converted_icc/area_linear_srgb/round = 2138 differing pixels
source variants: converted ICC and raw PNG Paletted/RGBA/NRGBA all preserve 70 unique colors
```

Decision: rejected as a renderer change. The best `Picture 4` kernel candidate
reduces the mismatch from `1200` to `1119` pixels but increases absolute channel
error and still does not pass the object fixture. The neighboring picture
fixture remains over `2100` differing pixels across the best source, transfer,
kernel, and area candidates. There is still no source-backed production picture
change that passes both object fixtures.

### WHO HIV Testing Algorithms Slide 015, Picture 4: fractional DrawingML bounds search

Added and ran an opt-in Go diagnostic that renders the picture crop against the
object's actual fractional DrawingML bounds from the manifest instead of first
rounding the geometry to an integer target rectangle:

```text
PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/picture-fractional-bounds-search.json \
go test ./internal/render -run TestMicroFixturePictureFractionalBoundsSearch -count=1 -v

PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_MANIFEST=.../cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/picture-fractional-bounds-search.json \
go test ./internal/render -run TestMicroFixturePictureFractionalBoundsSearch -count=1 -v
```

Result against `Picture 4`:

```text
baseline current approx_bilinear/round: 1200 differing pixels, abs channel delta 97602
fractional target: x=-22.192677..89.596378 y=-0.374173..111.414882
best candidate: converted_icc/bilinear_4x
best result: 1173 differing pixels, abs channel delta 57741
```

Result against neighboring EPA `Google Shape;11;p15`:

```text
baseline current approx_bilinear/round: 2127 differing pixels, abs channel delta 273264
fractional target: x=0.207323..193.917795 y=0.035591..55.249921
best candidate: converted_icc/bilinear_2x
best result: 2113 differing pixels, abs channel delta 157401
```

Decision: rejected as a production renderer change. Fractional DrawingML bounds
are source-backed and reduce aggregate channel error, but neither object fixture
passes and the pixel-count improvement is too small to accept. Keep this as
diagnostic evidence; do not switch production picture scaling to fractional
supersampling from these results alone.

### WHO HIV Testing Algorithms Slide 015, Picture 4: CatmullRom for all raster images

The current scaler uses `x/image/draw.CatmullRom` for uncropped YCbCr JPEG
sources and `ApproxBiLinear` for other raster sources. A narrow candidate made
all in-bounds raster images use CatmullRom to test whether Picture 4's PNG icon
needed higher-quality reconstruction.

Result against the `Picture 4` micro-fixture:

```text
current fixture crop diff: 1200 / 8280 pixels
CatmullRom all-raster candidate: 1203 / 8280 pixels
```

Decision: rejected and reverted. It worsens the attributed object and also
breaks the existing scaler contract that keeps non-JPEG raster sources on the
default scaler.

### EPA Generate Slide 007, Rectangle 9: include underpaint object in fixture

The Rectangle 9 target-scope diagnostic found one earlier underpaint object:
`Freeform 6` from `ppt/slideLayouts/slideLayout2.xml`, intersecting the target
edge at `x=365..959 y=19..110`. A harness candidate injected that raw shape
before Rectangle 9 in the extracted fixture so the partial-alpha edge would
blend over rendered underpaint.

Result against the Rectangle 9 micro-fixture:

```text
pre-background-preservation target-only fixture visible crop diff: 491 / 321300 pixels
with Freeform 6 injected: 513 / 321300 pixels
max channel delta improved from 19 to 10, but differing pixels increased
```

Decision: rejected and reverted. The underpaint object is itself custom
geometry with shadow, so injecting it makes Rectangle 9 depend on a second
renderer mismatch instead of isolating the rectangle edge. Keep the
`underpainted_by` and target-scope counts as diagnostics only.

### EPA Generate Slide 007, Rectangle 9: floor fractional coverage alpha

The remaining Rectangle 9 mismatch is a single partial-alpha left edge column,
so a narrow candidate changed fractional rectangle coverage alpha from rounded
to floored.

Result against the Rectangle 9 micro-fixture:

```text
pre-background-preservation visible crop diff: 491 / 321300 pixels
floor coverage alpha candidate: 495 / 321300 pixels
```

Decision: rejected and reverted. Lowering the partial-edge alpha does not
explain this object failure and worsens the executable fixture.

### EPA Generate Slide 007, Rectangle 9: floor shape-edge blend channels

After preserving the source slide background, the Rectangle 9 target-only
fixture had a one-column partial-alpha mismatch:

```text
current target-only visible crop diff: 540 / 321300 pixels
all 540 differing pixels inside the partial-alpha object mask
92 partial-alpha object-mask pixels over earlier underpaint
448 partial-alpha object-mask pixels over plain slide background
```

A narrow renderer candidate kept rounded coverage alpha but used floor division
when source-over blending partial-coverage rectangle edges. It changed only the
fractional rectangle edge blend path, not the shared `blendPixel` helper.

Result:

```text
focused shape/blend tests: pass
target-only visible crop diff: 19 / 321300 pixels
remaining 19 pixels: x=0, y=20..109, all over earlier underpaint
source-through visible crop diff: 4 / 321300 pixels
full corpus: 61/61 slides still differ; total 9,320,399 pixels
baseline before probe: 9,321,023 pixels
```

Decision: rejected and reverted. This strongly suggests the plain rectangle
edge channel rounding is closer to Apple Notes, but the executable object
fixture still does not pass exactly. The residual is confined to the
partial-alpha edge over `Freeform 6` underpaint, and the real source-through
path still has a 4-pixel mismatch because that underpaint object has its own
custom-geometry/shadow renderer gap. Do not land this renderer change until an
object fixture can pass exactly.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: vector antialias shadow seed

The clean `underpaint-shape-0001-7-Freeform-6` fixture isolates a custom
polygon with an outer shadow. A narrow candidate changed only custom-path
shadow rendering: instead of seeding the Gaussian blur from a binary
point-in-polygon mask, it seeded from the existing `x/image/vector` antialias
mask already used for custom picture masks.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
vector antialias shadow seed: 2360 / 22448 pixels
vector seed with blur radius -1: 3488 / 22448 pixels
vector seed with blur radius +1: 2932 / 22448 pixels
vector seed with x offset +1: 2357 / 22448 pixels
vector seed with y offset +1: 3483 / 22448 pixels
```

Decision: rejected and reverted. Vector antialiasing and a one-pixel x offset
are weakly directionally relevant, but neither comes close to passing the
object fixture. Do not land a custom shadow-mask change from this probe alone.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: custom shadow offset probes

The master `Freeform 6` diff row profile suggested a possible vertical shadow
phase issue: row `80` was too dark while rows `81..91` were too light. A narrow
candidate shifted only custom-path shadow bounds by one pixel vertically.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow y offset +1: 3484 / 22448 pixels
custom shadow y offset -1: 3505 / 22448 pixels
```

Decision: rejected and reverted. The row-level signal was not caused by a
simple vertical phase shift.

Additional narrow custom-path shadow offset probes after adding target-scope
direction buckets:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow x offset +2: 2419 / 22448 pixels
custom shadow x offset +1 and y offset -1: 3500 / 22448 pixels
custom shadow x offset +1 and y offset +1: 3446 / 22448 pixels
```

Decision: rejected and reverted. The earlier `x +1` improvement is a local
minimum, not an object-level pass, and combining it with vertical movement
destroys the fixture.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: shadow blur kernel probes

The master `Freeform 6` target-scope buckets show `2354 / 2368` differing
pixels inside low-alpha object-mask pixels, so the next probes tested whether
the shadow blur falloff shape was wrong.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
box blur instead of Gaussian blur: 3508 / 22448 pixels
Gaussian sigma radius/3 instead of radius/2: 3498 / 22448 pixels
Gaussian sigma radius instead of radius/2: 3503 / 22448 pixels
```

Decision: rejected and reverted. The mismatch is not explained by a simple
box-vs-Gaussian or broader/narrower Gaussian falloff change.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: shadow alpha quantization probes

The blur kernel writes 8-bit alpha by rounding the floating blur result. A
narrow probe tested the two adjacent quantization choices.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
ceil blurred alpha: 2793 / 22448 pixels
floor blurred alpha: 2856 / 22448 pixels
```

Decision: rejected and reverted. Low-alpha fringe mismatch is not fixed by
simple alpha quantization.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: fractional shadow bounds

The master `Freeform 6` object has fractional geometry bounds
`x=0..233.307638 y=26.548583..99.240866`, while the custom-path shadow path
currently receives the rounded integer shape target. A narrow candidate used
the fractional geometry paint bounds for custom-path shadows only.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow using fractional geometry bounds: 3499 / 22448 pixels
```

Decision: rejected and reverted. The shadow fringe mismatch is not fixed by
using the broader fractional paint bounds for the shadow seed.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: shadow blend rounding

Most master `Freeform 6` differences are low-alpha black shadow pixels over a
light background, so a narrow candidate changed only blurred-shadow source-over
channel blending from rounded division to floor division.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
floor shadow source-over blend: 2368 / 22448 pixels
```

Decision: rejected and reverted. Shadow blend rounding alone does not move the
fixture.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: target-scope direction buckets

The target-scope diagnostic now records whether the reference crop is darker or
lighter than the current render for each differing pixel, split by the current
object alpha mask buckets.

Refreshed master `Freeform 6` target-scope counts:

```text
different_pixels: 2368 / 22448
different_bounds: x=0..171 y=0..91
reference RGB delta sum, 8-bit channel sum: -5409
reference RGB absolute delta sum, 8-bit channel sum: 31755
reference darker: 1989
reference darker bounds: x=0..171 y=1..91
reference lighter: 379
reference lighter bounds: x=0..171 y=0..80
inside object mask: 2366
inside partial-alpha object mask: 2366
inside low-alpha object mask: 2354
inside low-alpha reference darker: 1987
inside low-alpha reference lighter: 367
inside mid-alpha object mask: 1
inside mid-alpha reference lighter: 1
inside high-alpha object mask: 11
inside high-alpha reference lighter: 11
outside object mask: 2
```

Hotspot rows/columns:

```text
top different rows: y=7:172, y=80:172, y=85:172, y=84:171, y=86:171, y=82:170, y=83:170, y=88:170
top darker rows: y=85:172, y=84:171, y=86:171, y=82:170, y=83:170, y=88:170, y=81:169, y=87:169
top lighter rows: y=80:172, y=7:161, y=6:7, y=4:5, y=2:3, y=5:3, y=1:2, y=3:2
top different columns: x=170:28, x=168:27, x=171:27, x=169:26, x=167:25, x=164:24, x=165:24, x=166:24
top signed RGB delta sums: -9:523, -15:477, -3:426, -12:344, -6:219, +33:164, +39:148, +3:23, +36:6, +27:4, +30:4, +48:4
```

Interpretation: the mismatch is still concentrated in the low-alpha shadow
fringe, but it is not uniformly one-sided. The top edge and row `80` are
already darker than the reference, while the bottom-edge band below row `80`
needs a darker reference result. The strongest columns sit near the slanted
right edge. A valid renderer fix needs to explain that edge/phase shape, not
just a global shadow opacity multiplier. The signed-delta histogram confirms
the dominant error is small per-pixel darkening, with a smaller but larger
positive tail at the rows that are already too dark.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: custom shadow opacity probes

The direction buckets suggested testing whether the custom-path shadow was
globally too weak. Two narrow candidates changed only the custom-path shadow
source alpha before blur.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow alpha +5%: 2784 / 22448 pixels
custom shadow alpha -5%: 2683 / 22448 pixels
```

Decision: rejected and reverted. The direction split is not explained by a
global custom-shadow opacity multiplier.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: pixel-center shadow seed probe

The row/column hotspots pointed at an edge-phase issue near the custom
polygon's slanted/right edge and bottom shadow band. A narrow candidate changed
only `drawSoftPolygon`'s shadow seed membership test from integer pixel corners
to pixel centers.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
pixel-center shadow seed: 2355 / 22448 pixels
```

Decision: rejected and reverted. The direction is slightly better, but the
object fixture still does not pass; do not land this seed-position change on
its own.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: direction column buckets

The target-scope diagnostic now records darker/lighter column hotspots in
addition to row hotspots. A debug rerender of the master `Freeform 6` fixture
kept the same `2368 / 22448` visible-crop diff and reported:

```text
reference-darker rows: 85:172, 84:171, 86:171, 82:170, 83:170, 88:170, 81:169, 87:169
reference-lighter rows: 80:172, 7:161, 6:7, 4:5, 2:3, 5:3, 1:2, 3:2
reference-darker columns: 170:25, 168:24, 171:24, 169:23, 167:22, 164:21, 165:21, 166:21
reference-lighter columns: 4:6, 1:5, 2:5, 3:4, 6:4, 8:4, 9:4, 10:4
```

This supports a phase/offset hypothesis: the right slanted/bottom shadow area
is too light, while the left/top shadow area is too dark. It does not support a
simple global opacity change.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: shadow-alpha diagnostic

Shape micro-fixture manifests for shadowed objects now include
`shadow-alpha-scope.json`. This estimates black shadow alpha over the
source-before-object crop, so the direction buckets describe shadow alpha
directly instead of only visible RGB brightness.

Refreshed master `Freeform 6` results:

```text
visible fixture diff: 2368 / 22448 pixels
partial-alpha shadow mask pixels: 5594
analyzed non-zero alpha-delta pixels: 2366
reference alpha greater: 1987 pixels
reference alpha less: 379 pixels
reference alpha delta sum: +1801
reference alpha absolute delta sum: 10583
top positive alpha deltas: +3:523, +5:477, +1:424, +4:344, +2:219
top negative alpha deltas: -11:164, -13:148, -1:23, -12:6
reference-alpha-greater rows: 85:172, 84:171, 86:171, 82:170, 83:170, 88:170, 81:169, 87:169
reference-alpha-less rows: 80:172, 7:161, 6:7, 4:5, 2:3, 5:3, 1:2, 3:2
reference-alpha-greater columns: 170:25, 168:24, 171:24, 169:23, 167:22, 164:21, 165:21, 166:21
reference-alpha-less columns: 4:6, 1:5, 2:5, 3:4, 6:4, 8:4, 9:4, 10:4
```

This confirms the mismatch is not a uniform opacity miss. Apple Notes wants
more shadow alpha across the bottom/right slanted-edge band and less alpha
along the top/left band. Future Freeform probes should target custom-path
shadow mask placement/kernel shape rather than global color, alpha, or blend
rounding.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: partial-alpha object tone split

The target-scope diagnostic now splits partial-alpha object artifact pixels by
the artifact's own unpremultiplied RGB tone. This distinguishes white fill-edge
pixels from black shadow pixels.

Debug target-scope result for the master `Freeform 6` fixture:

```text
visible fixture diff: 2368 / 22448 pixels
partial-alpha object artifact pixels: 5522 dark, 36 light, 36 other
differing partial-alpha pixels: 2354 dark, 12 light, 0 other
dark partial-alpha direction: 1987 reference-darker, 367 reference-lighter
light partial-alpha direction: 0 reference-darker, 12 reference-lighter
```

This confirms that the Freeform target is almost entirely a black shadow-mask
problem, not the white fill edge.

The object attribution style summary now records the source `outerShdw`
parameters so the manifest preserves the renderer inputs directly:

```text
geometry: customPath
custom_path_points: 4
custom_path_commands: 4
custom_path_bounds: x=0..1 y=0..1
shadow_color: #000000/66
shadow_blur_emu: 127000
shadow_distance_emu: 63500
shadow_direction: 1800000
shadow_alignment: tl
source_object_xml_path: source-object.xml
```

`source-object.xml` preserves the raw `p:sp` from
`ppt/slideMasters/slideMaster1.xml`, including the four-point `a:path`, the
`a:solidFill` scheme color, `a:ln/a:noFill`, and the exact `a:outerShdw`
attributes. Future Freeform probes should start from this source object
artifact rather than re-reading the full slide master manually.

The manifest also records `fixture_parts` with size and SHA-256 for each ZIP
part in the extracted micro-fixture. For this fixture the package has 11 parts:

```text
[Content_Types].xml
_rels/.rels
ppt/_rels/presentation.xml.rels
ppt/presentation.xml
ppt/slideLayouts/_rels/slideLayout2.xml.rels
ppt/slideLayouts/slideLayout2.xml
ppt/slideMasters/_rels/slideMaster1.xml.rels
ppt/slideMasters/slideMaster1.xml
ppt/slides/_rels/slide1.xml.rels
ppt/slides/slide1.xml
ppt/theme/theme1.xml
```

Each `fixture_parts` entry also records a `reason`, such as "stripped source
slide master dependency" for `ppt/slideMasters/slideMaster1.xml` and "theme
dependency for scheme colors and inherited styles" for `ppt/theme/theme1.xml`.
This keeps the fixture dependency surface reviewable from the manifest.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: x-shift plus pixel-center seed

A combined custom-path shadow probe shifted only custom-path shadow bounds one
pixel right and sampled the shadow seed at pixel centers.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
x+1 plus pixel-center shadow seed: 2364 / 22448 pixels
```

Decision: rejected and reverted. The combined phase adjustment is only four
pixels better and still fails the object fixture.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: custom shadow right-edge scale probes

The dark shadow tone split showed under-dark pixels concentrated on the
right/slanted edge, so three narrow custom-path shadow bounds probes changed
only the right/bottom seed bounds:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow maxX +1: 2314 / 22448 pixels
custom shadow maxX +2: 2360 / 22448 pixels
custom shadow maxX +1 plus pixel-center seed: 2333 / 22448 pixels
custom shadow minY +1: 3472 / 22448 pixels
custom shadow maxY +1: 2393 / 22448 pixels
```

The `maxX +1` probe is directionally useful but still far from passing. Its
target-scope shifts the remaining dark partial-alpha split to `1874`
reference-darker and `424` reference-lighter, so it trades some right-edge
under-dark error for more over-dark error at the far edge.

Decision: rejected and reverted. Do not land any one-pixel custom shadow bounds
change until the Freeform object fixture passes.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: right/bottom shadow bounds expansion

The shadow-alpha diagnostic showed the largest under-dark pixels on the
bottom/right slanted-edge band, so a narrow probe expanded only custom-path
shadow bounds by one pixel on the right and bottom edges before rasterizing the
shadow mask.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow bounds max x/y +1: 2367 / 22448 pixels
```

Decision: rejected and reverted. The one-pixel right/bottom expansion barely
improves the object fixture and does not address the top/left over-dark band.

### EPA Generate Slide 007, Freeform 6 Master Underpaint: transparent blur boundary

Because the object sits near the slide edge, a narrow probe changed the
Gaussian alpha blur to treat samples outside the clipped mask as transparent
instead of clamping to the clipped mask edge.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
transparent outside blur samples: 2368 / 22448 pixels
delta profile: unchanged
```

Decision: rejected and reverted. The Freeform mismatch is not caused by
clamped blur samples at the clipped slide edge.

### EPA Generate Slide 007, Rectangle 9: rerendered fixture debug output and blend-floor retest

The micro-fixture verifier can now write its rerendered `got.png`, target crop,
`micro-fixture-diff.json`, `micro-fixture-diff.png`, and `target-scope.json`
when `PUPPT_MICRO_FIXTURE_DEBUG_DIR` is set. This was used to inspect the
Rectangle 9 blend-floor retest without changing the fixture acceptance rule.

Shape micro-fixture manifests now also include diagnostic underpaint-chain
artifacts when earlier intersecting shape underpaints exist. The chain fixture
renders the earlier shape underpaints plus the target shape in z-order, but it
is not an acceptance target; the object fixture still gates renderer changes.
The manifest also records an `underpaint_chain_summary` so the object-only and
chain scopes can be compared without manually diffing JSON files.

Result against the Rectangle 9 micro-fixture:

```text
current visible fixture diff: 540 / 321300 pixels
current target scope: 92 partial-alpha differing pixels over underpaint, 448 without underpaint
underpaint-chain visible fixture diff: 536 / 321300 pixels
underpaint-chain target scope: 88 partial-alpha differing pixels over underpaint, 448 without underpaint
underpaint-chain summary delta: -4 total pixels; -4 underpainted partial-alpha pixels; 0 plain partial-alpha pixels
underpaint-chain signed RGB delta sum: -1082
floor source-over blend: 19 / 321300 pixels
remaining floor-blend diff bounds: x=0 y=20..109
remaining floor-blend target scope: 19 partial-alpha differing pixels over underpaint, 0 without underpaint
remaining floor-blend signed RGB delta sum: -350
```

Including the earlier Freeform underpaint in a chain fixture improves only
`4` pixels (`540 -> 536`), because that underpaint is itself still mismatched.
The floor-blend probe fixes the plain-background rectangle edge, and its
remaining 19 pixels align with rows where the reference visible crop contains
the earlier Freeform underpaint/shadow on the left edge.

Decision: rejected and reverted. Do not land the blend-floor change from
Rectangle 9 until its visible fixture passes with a faithful underpaint chain,
or until the upstream Freeform underpaint fixture passes and the Rectangle
fixture can be regenerated against that corrected renderer behavior.

### EPA Generate Slide 007, Rectangle 9: non-underpaint target diagnostic

The Rectangle 9 manifest now writes diagnostic crops with earlier underpaint
mask pixels made transparent:

```text
non-underpaint-got-crop.png
non-underpaint-reference-crop.png
non-underpaint-diff.json
```

This isolates the target object's edge from rows/columns where the earlier
`Freeform 6` object artifact contributes non-zero alpha. The refreshed
Rectangle 9 manifest reports:

```text
object-only visible fixture diff: 540 / 321300 pixels
object-only target scope: 92 partial-alpha differing pixels over underpaint, 448 without underpaint
non-underpaint diff: 448 / 321300 pixels
non-underpaint bounds: x=0 y=0..539
non-underpaint signed RGB delta sum: -896
non-underpaint signed RGB delta histogram: -2:448
```

This confirms that the Rectangle 9 target has two separable failures: `448`
plain-background partial-alpha edge pixels, and `92` pixels whose visible
result depends on the earlier Freeform underpaint. The plain-background part is
the part fixed by the rejected floor source-over blend probe; the remaining
underpainted part still depends on making the upstream Freeform underpaint
fixture faithful first.

### EPA Generate Slide 007, Freeform 6: source object summary in manifest

Micro-fixture manifests now include a compact `source_object_summary` parsed
from the extracted `source-object.xml`. This keeps the authored geometry and
shadow inputs next to the rendered fixture artifacts, without relying on the
production renderer's resolved-style summary alone.

The refreshed Freeform 6 underpaint manifest records:

```text
kind: sp
cNvPr: id=7 name="Freeform 6"
transform EMU: x=0 y=337167 cx=2963007 cy=923192
custom path: w=2963007 h=923192
points: moveTo(0,0), lnTo(2039815,0), lnTo(2963007,923192), lnTo(0,923192)
outer shadow: blurRad=127000 dist=63500 dir=1800000 algn=tl rotWithShape=0
```

Verification:

```text
Freeform fixture comparison: still fails at 2368 differing pixels
61-slide Apple Notes gate: still fails at 61/61 slides, total differing pixels=9321023, worst slide 001=308113
```

Decision: diagnostic metadata only. This confirms the current target is still
the Freeform shadow/rasterization behavior, not a missing manifest provenance
field or a drift between the raw XML and the object debug summary.

### EPA Generate Slide 007, Freeform 6: shadow pixel geometry summary

Shape micro-fixture manifests now also include `shadow_render_summary`, which
records the renderer-derived pixel geometry before the shadow mask is
rasterized. This makes the EMU-to-pixel shadow math reviewable from the
manifest instead of requiring code inspection for each probe.

The object style summary also records transformed custom-path coordinates, and
`shadow_render_summary` projects those coordinates into
`target_custom_path_pixel_points` and `shadow_custom_path_pixel_points`. This
keeps future custom-path shadow probes tied to the exact vertices used by the
renderer's integer shadow mask.

The refreshed Freeform 6 underpaint manifest records:

```text
canvas: 960x540
target bounds: x=0..232 y=27..98
shadow offset: x=4 y=2
shadow blur: 10px
shadow bounds before blur: x=4..236 y=29..100
shadow paint bounds after blur/canvas clip: x=0..246 y=19..110
```

Verification:

```text
Freeform fixture comparison: still fails at 2368 differing pixels
61-slide Apple Notes gate: still fails at 61/61 slides, total differing pixels=9321023, worst slide 001=308113
```

Decision: diagnostic metadata only. This narrows future shadow probes to the
mask rasterization/blur shape inside the recorded paint bounds; it does not
change production renderer behavior.

### EPA Generate Slide 007, Freeform 6: unclipped blur mask bounds probe and alpha bounds

A narrow candidate changed only blurred shadow mask construction: the Gaussian
mask was built over the full `shapeBounds.Inset(-blur)` area and clipped only
when compositing to the slide canvas, instead of clipping the mask bounds before
the blur. This tested whether slide-edge clipping/clamped blur samples caused
the top/left over-dark fringe.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
unclipped shadow blur mask bounds: 2368 / 22448 pixels
```

Decision: rejected and reverted. The mismatch is not explained by clipping the
blur mask to the slide canvas before the blur pass.

The shadow-alpha diagnostic now records bounds for both alpha directions:

```text
reference-alpha-greater pixels: 1987
reference-alpha-greater bounds: x=0..171 y=1..91
reference-alpha-less pixels: 379
reference-alpha-less bounds: x=0..171 y=0..80
```

This keeps the existing interpretation intact: Apple Notes wants more alpha in
the lower/right slanted-edge band and less alpha around the top/left fringe.

### EPA Generate Slide 007, Freeform 6: shadow alpha correction centroids

The shadow-alpha diagnostic now records per-direction delta sums and centroids
so future probes can be evaluated against the shape of the required correction,
not just total differing pixels or row hotspots.

The refreshed Freeform 6 underpaint manifest reports:

```text
reference-alpha-greater pixels: 1987
reference-alpha-greater delta sum: +6192
reference-alpha-greater centroid: x=91.51 y=81.17
reference-alpha-greater bounds: x=0..171 y=1..91
reference-alpha-less pixels: 379
reference-alpha-less delta sum: -4391
reference-alpha-less centroid: x=83.17 y=40.32
reference-alpha-less bounds: x=0..171 y=0..80
net alpha delta sum: +1801
absolute alpha delta sum: 10583
```

This reinforces the edge/phase interpretation: the largest positive correction
mass is much lower in the crop than the negative correction mass. A useful
custom-path shadow probe should move alpha downward/rightward without simply
raising global opacity.

### EPA Generate Slide 007, Freeform 6: shadow alpha correction heatmap

Shadowed shape micro-fixture manifests now include a
`shadow_alpha_correction_heatmap_path` artifact. The PNG is transparent where
the estimated black-shadow alpha already matches; red marks pixels where the
Apple Notes reference needs more shadow alpha; blue marks pixels where it needs
less. This provides a visual target for narrow custom-path shadow probes.

The refreshed Freeform 6 underpaint manifest points to:

```text
shadow-alpha-correction-heatmap.png
size: 244x92 RGBA
reference-alpha-greater: 1987 pixels, centroid x=91.51 y=81.17
reference-alpha-less: 379 pixels, centroid x=83.17 y=40.32
```

Verification:

```text
Freeform fixture comparison: still fails at 2368 differing pixels
61-slide Apple Notes gate: still fails at 61/61 slides, total differing pixels=9321023, worst slide 001=308113
```

Decision: diagnostic artifact only. Use this heatmap to evaluate whether a
candidate moves the shadow correction mass in the right direction before
considering full-corpus verification.

### EPA Generate Slide 007, Freeform 6: custom shadow vertical stretch probe

The correction heatmap showed the positive correction centroid lower than the
negative correction centroid, so a narrow candidate changed only custom-path
shadow rendering by stretching the normalized custom shadow path downward from
the top edge by `1%` before rasterizing the shadow mask.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow y scale 1.01 from top edge: 2393 / 22448 pixels
```

Decision: rejected and reverted. Bottom-edge expansion alone worsens the
object fixture, matching the earlier `maxY +1` result. Future probes should not
treat the lower correction centroid as evidence for a simple vertical stretch.

### EPA Generate Slide 007, Freeform 6: current-diagnostic maxX +1 retest

The earlier `maxX +1` custom shadow bounds probe was retested after adding the
shadow correction heatmap and target-scope direction fields. The candidate
changed only custom-path shadow rendering by expanding the custom shadow bounds
one pixel to the right before rasterizing the shadow mask.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
custom shadow maxX +1: 2314 / 22448 pixels
reference-darker pixels: 1989 -> 1876
reference-lighter pixels: 379 -> 438
dark partial-alpha reference-darker: 1987 -> 1874
dark partial-alpha reference-lighter: 367 -> 424
signed RGB delta sum: -5409 -> -4323
absolute RGB delta sum: 31755 -> 31503
```

Decision: rejected and reverted. The probe remains directionally useful but
still fails the object fixture, and the new diagnostics show the tradeoff: it
reduces the under-dark right-edge mass while increasing over-dark pixels at the
far-right/top fringe. Do not land it without a companion change that removes
the new over-dark tail and passes the object fixture.

### EPA Generate Slide 007, Freeform 6: four-sample shadow seed coverage

A narrow candidate changed only soft polygon shadow mask seeding: instead of a
binary point-in-polygon mask before Gaussian blur, it seeded the blur with the
same four-sample edge coverage used by filled polygons.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
four-sample custom shadow seed coverage: 2365 / 22448 pixels
reference-darker pixels: 1989 -> 1984
reference-lighter pixels: 379 -> 381
dark partial-alpha reference-darker: 1987 -> 1982
dark partial-alpha reference-lighter: 367 -> 367
signed RGB delta sum: -5409 -> -5502
absolute RGB delta sum: 31755 -> 31812
```

Decision: rejected and reverted. Four-sample edge coverage is only a three
pixel improvement and worsens aggregate channel error, so it does not explain
the Freeform shadow mismatch.

### EPA Generate Slide 007, Freeform 6: bottom-right vertex x probes

The `maxX +1` bounds retest improved the fixture but added an over-dark tail,
so two narrower probes moved only the lower-right custom path vertex to the
right while leaving the top-right vertex and overall shadow bounds unchanged.

Result against the master `Freeform 6` micro-fixture:

```text
current visible fixture diff: 2368 / 22448 pixels
bottom-right custom path vertex x +1px: 2368 / 22448 pixels
bottom-right custom path vertex x +2px: 2368 / 22448 pixels
```

Decision: rejected and reverted. The `maxX +1` improvement is not caused by
only moving the lower-right authored vertex. Its effect comes from changing the
whole custom shadow bounds/rasterization phase.

### EPA Generate Slide 007, Freeform 6: focused verifier shadow geometry output

`TestMicroFixtureManifestComparison` now writes extra debug-only files when
`PUPPT_MICRO_FIXTURE_DEBUG_DIR` is set:

```text
fixture-objects.json
current-object.json
shadow-render-summary.json
```

The verifier renders with normal object-debug bookkeeping in that mode, so
`current-object.json` reflects the current parser and renderer instead of only
the older checked-in manifest. Normal-mode debug pixel parity is covered by
`TestRenderObjectDebugNormalModeDoesNotChangePixels`.

Refreshed master `Freeform 6` debug output:

```text
visible fixture diff: 2368 / 22448 pixels
custom path normalized points: (0,0), (0.6884273307487967,0), (1,1), (0,1)
target pixel points: (0,27), (160,27), (233,99), (0,99)
shadow pixel points: (4,29), (164,29), (237,101), (4,101)
shadow bounds before blur: x=4..236 y=29..100
shadow paint bounds after blur/canvas clip: x=0..246 y=19..110
```

Decision: diagnostic metadata only. The points confirm that the current
custom-path shadow seed is using the expected authored trapezoid after
EMU-to-pixel scaling and offset. Combined with the alpha diagnostic, future
renderer probes should focus on mask phase/coverage or blur sampling around
those vertices, not raw XML extraction, object attribution, or a global opacity
change.

### EPA Generate Slide 007, Freeform 6: opt-in shadow phase search

Added an opt-in diagnostic:

```text
PUPPT_SHADOW_PHASE_SEARCH_MANIFEST=.../underpaint-shape-0001-7-Freeform-6/manifest.json \
PUPPT_SHADOW_PHASE_SEARCH_OUTPUT=/tmp/shadow-phase-search.json \
go test ./internal/render -run TestMicroFixtureShadowPhaseSearch -count=1 -v
```

The search renders the micro-fixture with current object-debug bookkeeping,
uses the current parsed custom path, and compares candidate custom-path shadow
alpha masks only on dark partial-alpha object pixels against the reference
crop/background. This is diagnostic-only; it does not use the 61-slide corpus
as a tuning target.

Result against the master `Freeform 6` micro-fixture:

```text
analyzed dark partial-alpha pixels: 3490
current phase baseline: 2339 differing alpha pixels, abs alpha delta 10374, signed alpha delta +1916
best searched phase: shiftX=0 shiftY=-0.75 sampleX=0 sampleY=0
best result: 2315 differing alpha pixels, abs alpha delta 10361, signed alpha delta +1565
best direction split: reference-alpha-greater 1868, reference-alpha-less 447
```

Decision: rejected as a renderer change. The best simple phase candidate is
only a small diagnostic improvement and does not make the object fixture pass.
It also increases the over-dark side of the split, matching the earlier pattern
where one-direction phase tweaks trade right/bottom under-dark pixels for a
top/left over-dark tail.

### EPA Generate Slide 007, Freeform 6: opt-in shadow composite search

Added a second opt-in diagnostic that scores the same custom-path shadow phase
candidates after compositing the candidate shadow and current custom-path fill
over the source-before crop:

```text
PUPPT_SHADOW_COMPOSITE_SEARCH_MANIFEST=.../underpaint-shape-0001-7-Freeform-6/manifest.json \
PUPPT_SHADOW_COMPOSITE_SEARCH_OUTPUT=/tmp/shadow-composite-search.json \
go test ./internal/render -run TestMicroFixtureShadowCompositeSearch -count=1 -v
```

The diagnostic applies the same later-object occlusion mask used by the visible
micro-fixture crop, then compares the candidate crop to
`reference-visible-crop.png`. It writes `shadow-composite-best.png` in the test
temp directory for inspection.

Result against the master `Freeform 6` micro-fixture:

```text
visible fixture diff: 2368 / 22448 pixels
current composite baseline: 2355 differing pixels, abs channel delta 31653, signed RGB delta -5229
best searched composite phase: shiftX=0.5 shiftY=-0.75 sampleX=0.5 sampleY=0
best composite result: 2333 differing pixels, abs channel delta 31653, signed RGB delta -4137
best direction split: reference-darker 1870, reference-lighter 463
```

Decision: rejected as a renderer change. The full-crop composite search
confirms the phase direction is only a small improvement and still fails the
object fixture. It also matches the previously rejected
`maxX +1 plus pixel-center seed` result (`2333` pixels), so landing this would
repeat a known insufficient fix.

### EPA Generate Slide 007, Rectangle 9: opt-in edge blend search

Added an opt-in diagnostic that renders the micro-fixture in `before` mode at
the target object's z-order, repaints only the rectangle over that crop with a
small set of edge coverage/source-over quantization variants, applies the same
later-object occlusion mask as the visible fixture, and compares against
`reference-visible-crop.png`.

```text
PUPPT_RECT_EDGE_BLEND_SEARCH_MANIFEST=.../shape-0006-7-Rectangle-9/manifest.json \
PUPPT_RECT_EDGE_BLEND_SEARCH_OUTPUT=/tmp/rect-edge-blend.json \
go test ./internal/render -run TestMicroFixtureRectEdgeBlendSearch -count=1 -v
```

Result against the master `Rectangle 9` micro-fixture:

```text
current baseline: 540 differing pixels, bounds x=0 y=0..539, abs channel delta 1430, signed RGB delta -1430
current non-underpaint target: 448 differing pixels, abs channel delta 896
best candidate: blend_floor, coverage=round, blend=floor
best result: 19 differing pixels, bounds x=0 y=20..109, abs channel delta 350, signed RGB delta -350
best non-underpaint target: 0 differing pixels
best direction split: reference-darker 19, reference-lighter 0
coverage_floor_blend_floor also scored 19; coverage_floor alone scored 540; coverage_ceil scored 540
```

Decision: rejected as a renderer change, but keep as source-backed sequencing
evidence. Floor source-over blending makes the non-underpainted target pass
exactly (`448 -> 0`) and leaves only `19` pixels in the known
Freeform-underpaint row range. That is consistent with the earlier direct
probe (`540 -> 19`, source-through visible diff `4`) and is still not enough to
make the full object fixture pass independently of the unresolved
underpaint/shadow gap.

### WHO HIV Slide 010, Picture 2: visible-crop picture resample/color search

The picture resample diagnostic was tightened to compare candidates against the
same acceptance target as `TestMicroFixtureManifestComparison`: when a picture
fixture has later-object occlusions, the search uses
`reference-visible-crop.png` and applies the recorded occlusion mask to each
candidate. The diagnostic also now tests two source color paths (`converted_icc`
and raw PNG pixels) and two output color paths (`display_p3` and no output
transform), so it can reproduce the renderer's actual output path before
ranking alternatives.

Command:

```text
PUPPT_PICTURE_RESAMPLE_SEARCH_MANIFEST=.../WHO-HIV-testing-algorithms-toolkit/slide-010/micro-fixtures/cumulative-picture-0001-3-Picture-2/manifest.json \
PUPPT_PICTURE_RESAMPLE_SEARCH_OUTPUT=.../picture-resample-search-current.json \
go test ./internal/render -run TestMicroFixturePictureResampleSearch -count=1 -v
```

Result against the worst cumulative picture fixture:

```text
visible fixture diff: 64711 pixels
best candidate: converted_icc/approx_bilinear/round/display_p3
best result: 64711 pixels, abs channel delta 6115553, signed RGB delta -404673
next candidates:
  converted_icc/nearest/round/display_p3: 64744 pixels
  converted_icc/nearest/floor_floor/display_p3: 64768 pixels
  converted_icc/nearest/floor_ceil/display_p3: 64782 pixels
  converted_icc/approx_bilinear/floor_floor/display_p3: 64798 pixels
```

Decision: rejected as a renderer change. The current renderer path is already
the best candidate among the tested source-color, output-color, scaler, and
target-endpoint variants. Raw PNG pixels are worse than the current ICC
conversion, and candidates that omit the Display P3 output transform cannot
reproduce the current crop. Treat this picture fixture as still open, but do
not repeat simple scaler, endpoint rounding, or raw-vs-ICC toggles as the next
fix path.

### EPA Metal Coil Slide 001, Rectangle 23: custom-path vector fill probe

The gate-relevant clean shape fixture selected for this probe is
`EPA-metal-coil-NESHAP-2018/slide-001/micro-fixtures/shape-0003-24-Rectangle-23/manifest.json`.
It is a semi-transparent custom path from the slide layout, with the left
slanted edge clipped against the slide right edge. The object fixture fails by
`9` visible-crop pixels, all at crop columns `0..1` and rows `105..110`; the
target-scope diagnostic reports all `9` as outside the current object artifact
alpha mask, with the reference darker in every differing pixel.

A production probe replaced custom-path shape fill with the existing vector
rasterizer used by picture custom masks, leaving preset polygon drawing
unchanged. The fixture result stayed exactly the same:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=.../EPA-metal-coil-NESHAP-2018/slide-001/micro-fixtures/shape-0003-24-Rectangle-23/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

micro-fixture visible crop mismatch: 9 differing pixel(s), bounds x=0..1 y=105..110
```

Decision: rejected and reverted. The 9-pixel edge miss is real, but simply
switching custom-path fill to the vector mask does not change the artifact and
should not be landed as a behavior change.

### Micro-fixture ownership ranking

Added an opt-in harness summary that scans extracted `manifest.json` files and
ranks object fixtures by target-scope ownership:

```text
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31 \
PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=.../micro-fixture-ownership-summary.json \
go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v
```

Current summary from the refreshed 2026-06-01 ownership command:

```text
total manifests: 170
manifests with target scope: 170
clean object-owned failures: 70
contaminated failures with outside-object pixels: 73
partial-alpha-over-underpaint failures: 9
clean failure candidate:
  WHO-HIV-testing-algorithms-toolkit slide 15 Picture 4
  total=1200, inside_object=1200, outside_object=0
  partial_alpha=0, partial_alpha_over_underpaint=0
  non_underpaint=0
```

Decision: use this ownership summary to choose the next object target. The
`Rectangle 23` 9-pixel fixture has a lower raw pixel count, but all 9 pixels are
outside the target object mask, so it is a contaminated attribution case, not a
renderer-fix target. `Picture 4` remains the first picture-contour target
because the refreshed ownership data attributes its residual to the target
object. `Rectangle 9` remains a useful shape-edge target, but its
partial-alpha/underpaint split means it still needs an object-isolated
acceptance target before any source-over rounding change can be accepted.

### 2026-06-01 Phase 2 checklist tightening

Added `docs/RENDERER_COMPLETION_CHECKLIST.md` as the top-to-bottom execution
ledger for `docs/RENDERER_COMPLETION_GOAL.md`, `docs/RENDERING.md`, and
`swe_skill.md`.

The object attribution records were tightened additively: each painted object
can now carry per-object unsupported items, explicit image relationship/crop
and image-effect summaries, and table unsupported summaries. This is a harness
and diagnostic schema change only; it does not alter production rendering
pixels.

Verification:

```text
go test ./internal/render -run 'TestRenderObjectDebug|TestObjectStyleSummaryIncludes|TestPaintedObjectRecordIncludesUnsupportedItems|TestWriteRealWorldDiffArtifactsWritesMetadata|TestMicroFixtureTargetOwnershipSummary|TestCleanMicroFixtureOwnershipFailureExcludesUnderpaintConfoundedEdges|TestRenderMicroFixtureWithObjectDebugWritesFixtureRecords' -count=1
go test ./internal/render -count=1
go test ./...
git diff --check
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31 PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=/Users/artpar/workspace/code/puppt/testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/micro-fixture-ownership-summary-current.json go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v
```

Decision: accepted as a Phase 2 harness fix. The next checklist phase is to use
the refreshed ownership ranking to select and verify the first clean object
target before any production renderer primitive changes.

### 2026-06-01 Fresh Phase 3 artifact run

Regenerated the real-world object artifacts against the checked-in Apple Notes
references:

```text
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 \
PUPPT_REALWORLD_ARTIFACT_DIR=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 \
go test ./internal/render -run TestRealWorldGoldenComparison -count=1
```

Result: expected parity failure, `61/61` slides differed with `9321023` total
differing pixels and no unsupported reports. Worst slide remained
`EPA-generate-2021-presentation.pptx` slide `001` with `308113` differing
pixels.

Generated the fresh ownership summary:

```text
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 \
PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=/Users/artpar/workspace/code/puppt/testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/micro-fixture-ownership-summary.json \
go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v
```

Result: `170` total manifests, `170` scoped manifests, `70` clean failures,
`73` contaminated failures, and `9` partial-underpaint failures. The current
clean picture-contour candidate is still:

```text
WHO-HIV-testing-algorithms-toolkit slide 015 Picture 4
manifest: testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json
different_pixels=1200
inside_object=1200
outside_object=0
partial_alpha_over_underpaint=0
```

The selected fixture preserves raw source `<p:pic>` XML with cNvPr id `1028`,
name `Picture 4`, `r:embed="rId5"`, empty `a:srcRect`, stretch/fillRect,
transform, `prstGeom rect`, `a:noFill`, and `bwMode="auto"`. The fixture
contains deterministic package entries and the media dependency as
`ppt/media/object.png`, sourced from `ppt/media/image17.png` (`200x200` PNG).
The verifier currently fails exactly at the object crop:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

micro-fixture crop mismatch: 1200 differing pixel(s), bounds x=0..67 y=18..91
```

Decision: use `Picture 4` as the first Phase 5.1 source-backed picture contour
coverage target. Do not edit picture rendering until its source XML, media
bytes, current production path, and rejected prior contour/resampling searches
are re-inspected against this fresh fixture.

### 2026-06-01 Picture 4 source and render-path reinspection

Authoritative source for the selected target:

```text
source object: testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/source-object.xml
source object sha256: b6c5375697195a9eb64a697b97c4cba4c978ce6c78eecdda2c01d951e557385e
fixture sha256: 7d5bfa85dd35a2075cd8d40cdd4ea770683374c9a09bf3309efa5349b40bd134
relationship: slide15.xml.rels rId5 -> ../media/image17.png
source media: ppt/media/image17.png, PNG, 200x200
fixture media: ppt/media/object.png, sha256 c60df9328e69b020494c156265fc1c23ca004bf68b0fddc45a656552bae08bd9
```

Relevant OOXML facts:

```text
object kind: p:pic
cNvPr id/name: 1028 / Picture 4
blip: r:embed="rId5", useLocalDpi val="0"
srcRect: empty
stretch: fillRect
transform: x=8595453 y=4567248 cx=1419721 cy=1419721
geometry: prstGeom rect
fill: noFill, hiddenFill white
bwMode: auto
```

Current production path inspected before editing:

```text
renderPicture -> pictureSourceImage -> fallbackPictureSourceImage -> decodePNGImage
renderPicture -> pictureSourceForElement -> scaleImage
scaleImage -> pictureScaler -> xdraw.ApproxBiLinear for this PNG source
writePNGWithDPI path applies Display P3 output transform after rasterization
```

Expected primitive behavior from the source: render the full 200x200 source PNG
into the authored square picture geometry with no source crop, no rotation, no
custom mask, no soft edge, no line, and no shadow. The selected residual is
therefore a picture contour/resampling/source-color primitive, not a layout,
crop, mask, text, shadow, or occlusion primitive.

Refreshed diagnostic profiles for the `object-debug-2026-06-01` fixture:

```text
PUPPT_PICTURE_RESIDUAL_PROFILE_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_RESIDUAL_PROFILE_OUTPUT=.../0009-1028-Picture-4/picture-residual-profile.json \
go test ./internal/render -run TestMicroFixturePictureResidualProfile -count=1 -v

PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_OUTPUT=.../0009-1028-Picture-4/picture-source-correspondence-profile.json \
go test ./internal/render -run TestMicroFixturePictureSourceCorrespondenceProfile -count=1 -v
```

Result:

```text
residual profile: differing=1200 grayscale=1200 edge=1200 pure_bw=0
got antialias differing pixels: 481
got hard differing pixels: 719
reference antialias differing pixels: 1200
source correspondence: source_bounds x=40..159 y=33..164
nearest source hard pixels: 950
nearest source antialias pixels: 250
mixed 3x3 source-neighborhood pixels: 798
solid 3x3 source-neighborhood pixels: 402
reference darker/lighter split: 540 / 660
```

Decision: diagnostic only. The source-backed target is a grayscale icon contour
coverage problem, but previous generic scaler, gamma, phase, area, source-model,
and thresholded-contour searches did not pass the object fixture. Do not change
production picture scaling until the next candidate explains why Apple Notes
has antialias values at all 1200 residual pixels while the current renderer
still emits hard black/white at 719 of them.

### 2026-06-01 Picture contour diagnostics: hard-edge smoothing rejection

Checked the embedded source PNG for the selected `Picture 4` target:

```text
file: PNG image data, 200 x 200, 8-bit colormap, non-interlaced
visible chunks before IDAT: IHDR, PLTE, IDAT
```

No visible pre-IDAT `gAMA`, `sRGB`, `iCCP`, `cICP`, or alpha chunk explains the
residual. The source-backed failure remains the paletted grayscale contour
itself.

Refreshed the thresholded contour supersampling diagnostic against the fresh
`object-debug-2026-06-01` fixture:

```text
PUPPT_PICTURE_CONTOUR_COVERAGE_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_CONTOUR_COVERAGE_SEARCH_OUTPUT=.../0009-1028-Picture-4/picture-contour-coverage-search.json \
go test ./internal/render -run TestMicroFixturePictureContourCoverageSearch -count=1 -v
```

Result:

```text
baseline: 1200 differing pixels, abs channel delta 97602
best contour candidate: ceil_ceil/threshold_128/6x
best result: 1196 differing pixels, abs channel delta 130143, signed RGB delta +10059
```

Added a narrow diagnostic-only edge search variant that smooths only hard
black/white output pixels whose immediate output neighborhood has mixed luma.
This tests the current profile observation that Apple Notes has antialias values
at every residual pixel while Puppt still emits hard black/white at many of
them. A second source-side variant smooths only hard black/white source palette
pixels whose immediate source neighborhood has mixed luma. These are not
production behavior.

Commands:

```text
PUPPT_PICTURE_EDGE_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_EDGE_SEARCH_OUTPUT=.../0009-1028-Picture-4/picture-edge-search.json \
go test ./internal/render -run TestMicroFixturePictureEdgeSearch -count=1 -v

PUPPT_PICTURE_EDGE_SEARCH_MANIFEST=.../EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_EDGE_SEARCH_OUTPUT=.../Google-Shape-11-p15/picture-edge-search.json \
go test ./internal/render -run TestMicroFixturePictureEdgeSearch -count=1 -v
```

Result:

```text
Picture 4 best candidate remained converted_icc/none/bilinear/floor_ceil/none:
  1156 differing pixels, abs channel delta 108759, signed RGB delta +43725

Google Shape slide 004 best candidate remained current approx_bilinear/round:
  2127 differing pixels, abs channel delta 273264, signed RGB delta -10188

best source-hard-edge candidate for Picture 4:
  source_hard_edge_gaussian_1/catmull_rom/round
  1173 differing pixels, abs channel delta 74532, signed RGB delta +10626

best source-hard-edge candidate for Google Shape:
  source_hard_edge_gaussian_1/catmull_rom/round
  2145 differing pixels, abs channel delta 180693, signed RGB delta -6651
```

Refreshed the existing sampling-phase diagnostic against the same fresh
artifacts:

```text
PUPPT_PICTURE_PHASE_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_PHASE_SEARCH_OUTPUT=.../0009-1028-Picture-4/picture-phase-search.json \
go test ./internal/render -run TestMicroFixturePicturePhaseSearch -count=1 -v

PUPPT_PICTURE_PHASE_SEARCH_MANIFEST=.../EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_PHASE_SEARCH_OUTPUT=.../Google-Shape-11-p15/picture-phase-search.json \
go test ./internal/render -run TestMicroFixturePicturePhaseSearch -count=1 -v
```

Result:

```text
Picture 4 best phase candidate:
  converted_icc/floor_ceil/src_+0.00_+0.25/dst_-0.50_+0.00
  1167 differing pixels, abs channel delta 136956, signed RGB delta +47124

Google Shape slide 004 best phase candidate:
  converted_icc/round/src_+0.25_+0.25/dst_-0.25_-0.50
  2113 differing pixels, abs channel delta 251661, signed RGB delta +18219
```

Decision: rejected as a renderer change. Edge-aware hard-pixel smoothing and
sampling-phase shifts do not pass `Picture 4`, do not pass the neighboring
`Google Shape;11;p15` fixture, and should remain diagnostic only. The
production change checkpoint remains open.

### EPA Generate Slide 007, Freeform 6: opt-in shadow parameter search

Added an opt-in diagnostic that scores a narrow set of outer-shadow parameter
variants around the authored `outerShdw` values for the upstream Freeform
underpaint object:

```text
PUPPT_SHADOW_PARAMETER_SEARCH_MANIFEST=.../underpaint-shape-0001-7-Freeform-6/manifest.json \
PUPPT_SHADOW_PARAMETER_SEARCH_OUTPUT=.../shadow-parameter-search-current.json \
go test ./internal/render -run TestMicroFixtureShadowParameterSearch -count=1 -v
```

The search composites candidate shadow masks over `source-before-crop.png`,
then paints the current custom-path fill and applies the same visible-crop
occlusion mask as the object fixture. It varies only shadow blur pixels, shadow
alpha, and one-pixel offsets around the renderer-derived values from the source
XML (`blurRad=127000`, `dist=63500`, `dir=1800000`, alpha `0x66`).

Result against the master `Freeform 6` underpaint fixture:

```text
composite baseline: blur=10 alpha=102 offset=(4,2), 2355 differing pixels, abs channel delta 31653, signed RGB delta -5229
best candidate: blur=10 alpha=102 offset=(4,2)
next candidate: blur=10 alpha=102 offset=(5,2), 2364 differing pixels, abs channel delta 32082
lower alpha / smaller blur candidates start at 2514 differing pixels and much higher abs channel delta
higher alpha / larger blur candidates increase the light-side error and start above 2640 differing pixels
```

Decision: rejected as a renderer change. The authored/current shadow
parameters are already the best candidate in this local blur/alpha/offset
family. Do not repeat simple outer-shadow blur, distance, or alpha tuning for
this Freeform object without a more specific source-backed model of the
remaining alpha distribution.

### EPA Generate Slide 007, Freeform 6: opt-in shadow kernel search

Added an opt-in diagnostic that keeps the authored/current Freeform shadow
alpha, offset, and blur radius fixed, but swaps the shadow blur kernel:

```text
PUPPT_SHADOW_KERNEL_SEARCH_MANIFEST=.../underpaint-shape-0001-7-Freeform-6/manifest.json \
PUPPT_SHADOW_KERNEL_SEARCH_OUTPUT=.../shadow-kernel-search-current.json \
go test ./internal/render -run TestMicroFixtureShadowKernelSearch -count=1 -v
```

Result against the master `Freeform 6` underpaint fixture:

```text
current gaussian sigma=radius/2: 2355 differing pixels, abs channel delta 31653, signed RGB delta -5229
gaussian sigma=radius/4: 3495 differing pixels, abs channel delta 104304
gaussian sigma=radius/3: 3497 differing pixels, abs channel delta 77253
one-pass box blur: 3509 differing pixels, abs channel delta 142341
three-pass box blur: 3511 differing pixels, abs channel delta 306702
```

Decision: rejected as a renderer change. The current Gaussian kernel is already
the best tested blur primitive for this fixture. Do not repeat box blur or
narrower Gaussian-kernel probes for the Freeform shadow mismatch without a
more specific object-level source signal.

### EPA Generate Slide 007, Freeform 6: opt-in fractional shadow geometry search

Added an opt-in diagnostic that keeps the authored/current Freeform shadow
alpha and Gaussian blur fixed, but compares integer pixel bounds against the
object's recorded fractional slide-to-pixel bounds before seeding the custom
path shadow mask:

```text
PUPPT_SHADOW_GEOMETRY_SEARCH_MANIFEST=.../underpaint-shape-0001-7-Freeform-6/manifest.json \
PUPPT_SHADOW_GEOMETRY_SEARCH_OUTPUT=.../shadow-geometry-search-current.json \
go test ./internal/render -run TestMicroFixtureShadowGeometrySearch -count=1 -v
```

Result against the master `Freeform 6` underpaint fixture:

```text
current integer target rect + integer offset: 2355 differing pixels, abs channel delta 31653, signed RGB delta -5229
best candidate: fractional exact target rect + fractional offset, sample=(0,0.5)
best candidate result: 2321 differing pixels, abs channel delta 30351, signed RGB delta +29853
best candidate target rect: x=0..233.30763779527558, y=26.548582677165353..99.24086614173228
best candidate shadow rect: x=4.330127018922194..237.63776481419777, y=29.048582677165353..101.74086614173228
```

Decision: rejected as a renderer change. Fractional geometry is a real signal
and slightly reduces raw differing pixels, but the best variant overcorrects
the Freeform shadow: the reference becomes lighter in `2252` pixels and darker
in only `69` pixels. Do not land fractional custom-path shadow bounds from this
probe alone; it needs a narrower source-backed model that reduces both pixel
count and signed channel direction.

### WHO HIV Slide 015, Picture 4: raw picture XML fixture preservation

The refreshed ownership summary still ranks `EPA Generate` slide 007
`Rectangle 9` as an underpaint-confounded low-diff candidate. That target is
partially blocked by the upstream `Freeform 6` underpaint. The current clean,
non-underpainted picture-contour object is:

```text
WHO-HIV-testing-algorithms-toolkit slide 015 Picture 4
fixture: .../slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json
current fixture diff: 1200 pixels, all inside the full-alpha object mask
```

While inspecting this target, the picture micro-fixture extractor was found to
synthesize a simplified `<p:pic>` instead of preserving the source object XML.
That dropped source-backed fields such as `descr`, `a14:useLocalDpi`, empty
`a:srcRect`, `bwMode`, `hiddenFill`, and the original image relationship id.
The fixture writer now embeds the raw source `<p:pic>` and keeps the source
relationship id (`rId5` for this object).

Verification after regenerating real-world artifacts:

```text
fixture slide XML contains: descr, r:embed, useLocalDpi, srcRect, bwMode, noFill, hiddenFill
fixture relationship: Id="rId5" Target="../media/object.png"
Picture 4 object fixture: still fails at 1200 differing pixels
picture resample search best: converted_icc/bilinear/floor_ceil/display_p3
best result: 1156 differing pixels, abs channel delta 108759, signed RGB delta +43725
current baseline: 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
61-slide Apple Notes gate: still fails at 61/61 slides, total differing pixels=9321023
```

Decision: accepted as a harness fix, rejected as a renderer fix. The object
fixture is now source-faithful for picture XML, but the Picture 4 renderer gap
remains. Do not accept the `bilinear/floor_ceil` candidate: it reduces raw
pixel count by only `44` while worsening absolute channel error and signed
direction.

### WHO HIV Slide 015, Picture 4: residual edge-coverage profile

Added an opt-in residual profiler for picture micro-fixtures. It compares the
current object crop to the Apple Notes reference crop and classifies differing
pixels as grayscale edge coverage, hard black/white placement differences, or
colored residuals:

```text
PUPPT_PICTURE_RESIDUAL_PROFILE_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_RESIDUAL_PROFILE_OUTPUT=.../picture-residual-profile-current.json \
go test ./internal/render -run TestMicroFixturePictureResidualProfile -count=1 -v
```

Result against the refreshed, raw-XML `Picture 4` fixture:

```text
source image: 200x200 PNG, 39 unique opaque grayscale colors
current fixture diff: 1200 pixels
grayscale differing pixels: 1200
edge-coverage differing pixels: 1200
pure black/white differing pixels: 0
colored differing pixels: 0
got antialias differing pixels: 481
reference antialias differing pixels: 1200
got hard differing pixels: 719
reference hard differing pixels: 0
```

Decision: diagnostic only. This rules out a hard placement or color-space
failure for this object: the remaining Picture 4 mismatch is entirely
grayscale edge-coverage disagreement. Future picture probes should focus on
edge reconstruction/coverage for opaque black-white PNG icons, not on global
color conversion, integer object shifts, or hard black/white thresholding.

### WHO HIV Slide 015, Picture 4: opt-in edge smoothing search

Added an opt-in diagnostic that tests small source/output edge-smoothing
variants around the existing attributed picture path. It keeps the target to
the extracted object crop and compares only against the object reference crop:

```text
PUPPT_PICTURE_EDGE_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_EDGE_SEARCH_OUTPUT=.../picture-edge-search-current.json \
go test ./internal/render -run TestMicroFixturePictureEdgeSearch -count=1 -v
```

The search tries the existing scaler/target modes with no smoothing,
one-pixel source box/gaussian smoothing, and one-pixel output box/gaussian
smoothing.

Result against the refreshed `Picture 4` fixture:

```text
baseline: converted_icc/none/approx_bilinear/round/none
baseline result: 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
best candidate: converted_icc/none/bilinear/floor_ceil/none
best result: 1156 differing pixels, abs channel delta 108759, signed RGB delta +43725
best source-smoothing candidate: converted_icc/source_gaussian_1/approx_bilinear/floor_ceil/none
best source-smoothing result: 1182 differing pixels, abs channel delta 116502, signed RGB delta +45288
output-smoothing candidates did not rank in the top 20
```

Decision: rejected as a renderer change. Simple source/output smoothing does
not explain the edge-coverage residual. The same `bilinear/floor_ceil` variant
is still the lowest pixel-count candidate, and it still worsens aggregate
channel error. Future Picture 4 work needs a more specific reconstruction
model than generic preblur/postblur smoothing.

### WHO HIV Slide 015, Picture 4: opt-in transfer-function search

Added an opt-in diagnostic that tests whether the residual comes from scaling
in a different transfer function. It wraps picture resampling with byte-space,
linear-sRGB, and gamma-family working-space candidates, then compares each
candidate against the attributed object crop:

```text
PUPPT_PICTURE_GAMMA_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_GAMMA_SEARCH_OUTPUT=.../picture-gamma-search-current.json \
go test ./internal/render -run TestMicroFixturePictureGammaSearch -count=1 -v
```

Result against the refreshed `Picture 4` fixture:

```text
baseline: converted_icc/srgb_byte/approx_bilinear/round
baseline result: 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
best candidate: converted_icc/srgb_byte/bilinear/floor_ceil
best result: 1156 differing pixels, abs channel delta 108759, signed RGB delta +43725
best gamma candidate: converted_icc/gamma_20/bilinear/floor_ceil
best gamma result: 1164 differing pixels, abs channel delta 103743, signed RGB delta -17055
linear_srgb candidates did not rank in the top 20
focused Picture 4 object fixture: still fails at 1200 differing pixels
```

Decision: rejected as a renderer change. Transfer-function scaling does not
explain the Picture 4 edge-coverage residual. The lowest pixel-count candidate
is still the earlier byte-space `bilinear/floor_ceil` probe, which remains
unacceptable because it worsens channel error and does not pass the object
fixture.

### WHO HIV Slide 015, Picture 4: opt-in scaler-kernel search

Added an opt-in diagnostic that tests scaler kernels against the attributed
picture object crop. This keeps the source picture and target object fixture
unchanged, varies only resampling kernels plus the existing target endpoint
rounding candidates, and compares against the extracted object acceptance crop:

```text
PUPPT_PICTURE_KERNEL_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_KERNEL_SEARCH_OUTPUT=.../picture-kernel-search-current.json \
go test ./internal/render -run TestMicroFixturePictureKernelSearch -count=1 -v
```

The search includes the current `ApproxBiLinear`, `BiLinear`, and `CatmullRom`
paths plus custom box, linear-support, B-spline, Mitchell, sharper cubic,
Lanczos, and Gaussian kernels.

Result against the refreshed `Picture 4` fixture:

```text
baseline: converted_icc/approx_bilinear/round
baseline result: 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
lowest pixel-count candidate: converted_icc/cubic_sharp/floor_ceil
lowest pixel-count result: 1119 differing pixels, abs channel delta 116838, signed RGB delta +45414
bilinear/floor_ceil result: 1156 differing pixels, abs channel delta 108759, signed RGB delta +43725
lanczos2/round result: 1200 differing pixels, abs channel delta 83283, signed RGB delta +12189
focused Picture 4 object fixture: still fails at 1200 differing pixels
```

Decision: rejected as a renderer change. Kernel choice alone does not explain
the Picture 4 residual. Sharper cubic kernels reduce raw pixel count but
worsen aggregate channel error, while Lanczos reduces channel error without
reducing the object-level failure count. The next useful probe should measure
the residual edge geometry/coverage directly rather than swapping generic
resampling kernels.

### WHO HIV Slide 015, Picture 4: residual edge-geometry profile

Added an opt-in geometry profiler for the attributed picture object residual.
It compares the current crop to the reference crop and classifies differing
pixels by crop-edge location, row/column concentration, hard-vs-antialias luma
state, and luma deltas:

```text
PUPPT_PICTURE_EDGE_GEOMETRY_PROFILE_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_EDGE_GEOMETRY_PROFILE_OUTPUT=.../picture-edge-geometry-profile-current.json \
go test ./internal/render -run TestMicroFixturePictureEdgeGeometryProfile -count=1 -v
```

Result against the refreshed `Picture 4` fixture:

```text
current fixture diff: 1200 pixels, bounds x=0..67 y=18..91
crop-left edge pixels: 3
crop-right edge pixels: 0
crop-top edge pixels: 0
crop-bottom edge pixels: 48
near-crop-edge pixels: 105
interior pixels: 1095
got hard pixels: 719
got antialias pixels: 481
reference hard pixels: 0
reference antialias pixels: 1200
top residual row: y=90, 48 pixels
top residual columns: x=65..67, 67 pixels each
relative fractional target bounds: x=-22.1927..89.5964 y=-0.3742..111.4149
relative output crop bounds: x=0..89 y=0..91
```

Decision: diagnostic only. The residual is not primarily a crop-edge clipping
problem: most differing pixels are interior to the visible object crop. The
reference is antialiased at every differing pixel while the current render has
hard black/white values at most of them. Future Picture 4 work should focus on
internal icon contour/coverage reconstruction, not object-crop edge coverage,
generic kernels, or global color transfer.

### WHO HIV Slide 015, Picture 4: opt-in area-resampling search

Added an opt-in diagnostic that renders the attributed picture object with
exact source-area averaging over each destination pixel. This tests whether the
interior icon-contour residual is explained by coverage-preserving downscale
rather than by the interpolation kernels already tried:

```text
PUPPT_PICTURE_AREA_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_AREA_SEARCH_OUTPUT=.../picture-area-search-current.json \
go test ./internal/render -run TestMicroFixturePictureAreaSearch -count=1 -v
```

The search includes byte-space area averaging and area averaging through
linear-sRGB / gamma-2.0 working spaces, across the same target endpoint
rounding candidates.

Result against the refreshed `Picture 4` fixture:

```text
current fixture: 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
best area candidate by pixel count: converted_icc/area_gamma_20/floor_ceil
best area result: 1202 differing pixels, abs channel delta 106914, signed RGB delta -4836
area_srgb_byte/floor_ceil result: 1203 differing pixels, abs channel delta 112407, signed RGB delta +44271
area_srgb_byte/round result: 1226 differing pixels, abs channel delta 81360, signed RGB delta +8118
focused Picture 4 object fixture: still fails at 1200 differing pixels
```

Decision: rejected as a renderer change. Exact area averaging does not pass the
object fixture and increases the exact pixel failure count, even where it
reduces aggregate channel error. The Picture 4 residual is not solved by a
generic area downscale replacement.

### WHO HIV Slide 015, Picture 4: opt-in sampling-phase search

Added an opt-in diagnostic that renders the attributed picture object with a
local bilinear sampler while varying source and destination subpixel sampling
phase. This tests whether the interior icon-contour residual is a simple phase
alignment issue after the generic kernel and area-resampling probes failed:

```text
PUPPT_PICTURE_PHASE_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_PHASE_SEARCH_OUTPUT=.../picture-phase-search-current.json \
go test ./internal/render -run TestMicroFixturePicturePhaseSearch -count=1 -v
```

The search uses the `round` and `floor_ceil` target endpoint candidates, source
phase values `-0.5..0.5`, and destination phase values `-0.5..0.5`.

Result against the refreshed `Picture 4` fixture:

```text
current fixture: 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
custom bilinear zero-phase baseline: 1199 differing pixels, abs channel delta 97215, signed RGB delta +12861
best phase candidate: converted_icc/floor_ceil/src_+0.00_+0.25/dst_-0.50_+0.00
best phase result: 1167 differing pixels, abs channel delta 136956, signed RGB delta +47124
focused Picture 4 object fixture: still fails at 1200 differing pixels
```

Decision: rejected as a renderer change. Sampling phase can reduce raw pixel
count slightly, but only by substantially worsening aggregate channel error,
and it does not pass the object fixture. Do not land a phase shift from this
probe alone.

### WHO HIV Slide 015, Picture 4: opt-in source image model search

The extracted fixture media is a single `200x200` PNG with an 8-bit palette
(`*image.Paletted` in Go). There is no hidden SVG/vector alternate in the
micro-fixture. Added an opt-in diagnostic that converts the decoded paletted
source to `*image.RGBA` and `*image.NRGBA` before scaling, then compares each
source model against the attributed object acceptance crop:

```text
PUPPT_PICTURE_SOURCE_MODEL_SEARCH_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_SOURCE_MODEL_SEARCH_OUTPUT=.../picture-source-model-search-current.json \
go test ./internal/render -run TestMicroFixturePictureSourceModelSearch -count=1 -v
```

Result against the refreshed `Picture 4` fixture:

```text
source models tested: converted_icc, converted_icc_rgba, converted_icc_nrgba,
raw_png, raw_png_rgba, raw_png_nrgba
all source models: 200x200, 39 unique colors
baseline: converted_icc/approx_bilinear/round
baseline result: 1200 differing pixels, abs channel delta 97602, signed RGB delta +12624
best candidate for every source model: bilinear/floor_ceil
best result for every source model: 1156 differing pixels, abs channel delta 108759, signed RGB delta +43725
focused Picture 4 object fixture: still fails at 1200 differing pixels
```

Decision: rejected as a renderer change. The Picture 4 residual is not caused
by Go scaling directly from an `image.Paletted` source. RGBA/NRGBA source
copies produce identical ranked results.

### Micro-Fixture Ownership Summary: clean failure classification

Tightened the ownership summary classification after the current summary showed
`Rectangle 9` as both a clean failure and a partial-underpaint failure. That was
misleading for the object-attributed loop: partial-alpha pixels over earlier
underpaint mean the target is not a clean standalone object failure yet.

Result after the harness change:

```text
PUPPT_MICRO_FIXTURE_ROOT=.../object-debug-2026-05-31 \
PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=.../micro-fixture-ownership-summary-current.json \
go test ./internal/render -run 'TestMicroFixtureTargetOwnershipSummary|TestCleanMicroFixtureOwnershipFailureExcludesUnderpaintConfoundedEdges' -count=1 -v

total manifests: 170
scoped manifests: 170
clean failures: 70
contaminated failures: 73
partial-underpaint failures: 9
clean picture-contour candidate: WHO-HIV-testing-algorithms-toolkit slide 015 Picture 4, 1200 pixels
```

Decision: accepted as a harness fix. A clean ownership failure now requires all
differing pixels to be inside the object mask and zero partial-alpha overlap
with underpaint. `Rectangle 9` remains tracked, but no longer steers the clean
failure queue ahead of the source-faithful `Picture 4` fixture.

### WHO HIV Slide 015, Picture 4: source-coordinate residual profile

Added an opt-in diagnostic that maps each current Picture 4 crop residual back
to the nearest source PNG coordinate under the current `round` target geometry.
This is a source-correspondence profile only; it does not change production
picture rendering.

```text
PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_OUTPUT=.../picture-source-correspondence-profile-current.json \
go test ./internal/render -run TestMicroFixturePictureSourceCorrespondenceProfile -count=1 -v
```

Result against the refreshed `Picture 4` fixture:

```text
current fixture diff: 1200 pixels, bounds x=0..67 y=18..91
target mode: round, relative target bounds x=-22..89 y=0..110
source coordinate bounds for residuals: x=40..159 y=33..164
nearest source hard pixels: 950
nearest source antialias pixels: 250
nearest source black pixels: 530
nearest source white pixels: 420
nearest source gray pixels: 250
mixed 3x3 source-neighborhood pixels: 798
solid 3x3 source-neighborhood pixels: 402
focused Picture 4 object fixture: still fails at 1200 differing pixels
```

Decision: diagnostic only. The residual is strongly associated with source
icon contours, but not with a simple nearest-source antialias value: most
residuals map to hard black or white source pixels while their local 3x3 source
neighborhood is often mixed. This supports a contour-coverage reconstruction
hypothesis and rules against spending more time on palette model conversion,
global transfer functions, or generic scaler swaps for this object.

### EPA Residential Wood Slide 004, Google Shape Picture: next clean failure probe

After the ownership summary fix, the next clean attributed picture failure
after `Picture 4` is:

```text
EPA-residential-wood-MacCarty slide 004
object: cNvPr id=11 name="Google Shape;11;p15"
fixture: .../slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json
source image: 421x120 PNG
visible fixture diff: 2127 pixels, bounds x=0..193 y=0..54
```

Focused fixture check:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=.../cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
micro-fixture visible crop mismatch: 2127 differing pixels
all 2127 residual pixels are grayscale edge-coverage pixels
got hard pixels: 902
got antialias pixels: 1225
reference hard pixels: 1
reference antialias pixels: 2126
source coordinate bounds for residuals: x=1..419 y=1..118
nearest source hard pixels: 1481
nearest source antialias pixels: 646
nearest source black pixels: 578
nearest source white pixels: 903
nearest source gray pixels: 646
mixed 3x3 source-neighborhood pixels: 1709
solid 3x3 source-neighborhood pixels: 418
```

Decision: diagnostic only. This next clean failure is also an opaque grayscale
picture contour-coverage mismatch, not a simple object crop, occlusion, or
global color problem. It reinforces the same source-contour reconstruction
hypothesis as `Picture 4`; it does not provide a separate low-risk renderer
fix to land.

### Picture contour-coverage reconstruction search

Added an opt-in diagnostic to test the source-contour reconstruction hypothesis
directly. The candidate treats an opaque grayscale picture as a luminance mask,
thresholds the source, then supersamples destination coverage. This is a
diagnostic search only; it is not production rendering behavior.

```text
PUPPT_PICTURE_CONTOUR_COVERAGE_SEARCH_MANIFEST=.../manifest.json \
PUPPT_PICTURE_CONTOUR_COVERAGE_SEARCH_OUTPUT=.../picture-contour-coverage-search-current.json \
go test ./internal/render -run TestMicroFixturePictureContourCoverageSearch -count=1 -v
```

Result for `WHO-HIV-testing-algorithms-toolkit` slide 015 `Picture 4`:

```text
current baseline: 1200 differing pixels, abs channel delta 97602
best contour candidate: ceil_ceil/threshold_128/6x
best result: 1196 differing pixels, abs channel delta 130143
focused Picture 4 object fixture: still fails at 1200 differing pixels
```

Result for `EPA-residential-wood-MacCarty` slide 004
`Google Shape;11;p15`:

```text
current baseline: 2127 differing pixels, abs channel delta 273264
best contour candidate: round/threshold_096/8x
best result: 2119 differing pixels, abs channel delta 213345
focused EPA picture object fixture: still fails at 2127 differing pixels
```

Decision: rejected as a renderer change. Thresholded contour supersampling
slightly reduces raw pixel counts on both clean picture fixtures, but neither
fixture passes. On `Picture 4` it worsens aggregate channel error; on the EPA
picture it improves channel error while creating a large darker bias. This is
not strong enough to replace the picture scaler.

### Picture 4 source extension and current fixture rerun

Rechecked the source model for the selected Phase 5.1 picture target against
the maintained local ECMA schema and the Microsoft Office Drawing extension
reference notes:

```text
ECMA dml-picture.xsd:14-21      CT_Picture is nvPicPr, blipFill, spPr
ECMA dml-main.xsd:648-652       CT_RelativeRect crop/fill values default to 0
ECMA dml-main.xsd:1455-1464     stretch fill mode owns optional fillRect
ECMA dml-main.xsd:1502-1509     CT_BlipFillProperties is blip, srcRect, fill mode, dpi/rotWithShape
MS-ODRAWXML 2.3.1.13            useLocalDpi is a BLIP extension for local BLIP compression override
MS-ODRAWXML 5.1 schema          hiddenFill is an Office Drawing 2010 main extension element
```

`Picture 4` has `a14:useLocalDpi val="0"` and `a14:hiddenFill` in the raw
source XML, but the current fixture media is an opaque PNG. Rerun residual
diagnostics confirm the source image has `40000` opaque pixels and `0` alpha
pixels, and the crop residual is still entirely grayscale edge coverage:

```text
PUPPT_PICTURE_RESIDUAL_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_RESIDUAL_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/picture-residual-profile-current-rerun.json \
go test ./internal/render -run TestMicroFixturePictureResidualProfile -count=1 -v

PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/picture-source-correspondence-profile-current-rerun.json \
go test ./internal/render -run TestMicroFixturePictureSourceCorrespondenceProfile -count=1 -v
```

Results:

```text
Picture 4 fixture verifier: 1200 crop differing pixels, bounds x=0..67 y=18..91
Google Shape;11;p15 fixture verifier: 2127 visible-crop differing pixels, bounds x=0..193 y=0..54

Picture 4 residual profile:
  source: 200x200, 39 unique colors, 40000 opaque pixels, 0 alpha pixels
  differing pixels: 1200
  grayscale differing pixels: 1200
  edge-coverage differing pixels: 1200
  got hard differing pixels: 719
  got antialias differing pixels: 481
  reference hard differing pixels: 0
  reference antialias differing pixels: 1200

Picture 4 source correspondence:
  source coordinate bounds for residuals: x=40..159 y=33..164
  nearest source hard pixels: 950
  nearest source antialias pixels: 250
  mixed 3x3 source-neighborhood pixels: 798
  solid 3x3 source-neighborhood pixels: 402
```

Decision: reject an extension-driven production picture change for this target.
`useLocalDpi` is not evidence for changing visible raster scaling, and
`hiddenFill` does not explain an opaque-PNG contour mismatch with no source
alpha pixels. The next Phase 5.1 path still needs a source-backed
coverage/reconstruction model for opaque grayscale icon contours, proven on
both `Picture 4` and the neighboring EPA `Google Shape;11;p15` fixture before
any production picture renderer edit.

### Picture PNG metadata profile

Added an opt-in profile for the raw PNG bytes inside an extracted picture
fixture. The goal is to prove whether ignored PNG chunks such as gamma, ICC,
physical pixel size, or transparency are a credible source-backed renderer lead
before changing image decoding or scaling.

```text
PUPPT_PICTURE_PNG_METADATA_PROFILE_MANIFEST=.../0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_PNG_METADATA_PROFILE_OUTPUT=.../0009-1028-Picture-4/picture-png-metadata-profile.json \
go test ./internal/render -run TestMicroFixturePicturePNGMetadataProfile -count=1 -v

PUPPT_PICTURE_PNG_METADATA_PROFILE_MANIFEST=.../cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_PNG_METADATA_PROFILE_OUTPUT=.../cumulative-picture-0001-11-Google-Shape-11-p15/picture-png-metadata-profile.json \
go test ./internal/render -run TestMicroFixturePicturePNGMetadataProfile -count=1 -v
```

Results:

```text
Picture 4:
  media: ppt/media/object.png
  sha256: c60df9328e69b020494c156265fc1c23ca004bf68b0fddc45a656552bae08bd9
  dimensions: 200x200
  PNG type: 8-bit indexed-color
  chunks: IHDR, PLTE, IDAT, IEND; all CRCs valid
  palette entries: 39
  absent chunks: tRNS, gAMA, sRGB, iCCP, pHYs

EPA Google Shape;11;p15:
  media: ppt/media/object.png
  sha256: e58bde653a8579ceb5839083b0f5c4bc30cc5d8e04aa1e5f8af67c79bda82eb5
  dimensions: 421x120
  PNG type: 8-bit indexed-color
  chunks: IHDR, PLTE, IDAT, IEND; all CRCs valid
  palette entries: 256
  absent chunks: tRNS, gAMA, sRGB, iCCP, pHYs
```

Decision: reject PNG metadata handling as the next production picture change.
The two clean picture-contour fixtures contain no gamma, color-profile,
physical-DPI, or transparency chunks that could explain the reference edge
coverage. The source-backed failure remains indexed-color grayscale contour
reconstruction/resampling, not metadata interpretation.

Phase 5.1 boundary: no production picture renderer change is accepted from the
current evidence packet. The remaining work needs a new source-backed
contour-reconstruction model, not another broad scaler, gamma, phase, metadata,
extension, or global color experiment. Continue the next failure families while
keeping `Picture 4` and `Google Shape;11;p15` as the required picture gates for
any future picture-contour candidate.

### WHO HIV Slide 015, TextBox 7: target-scope color buckets

Inspected the next clean, non-picture object fixture instead of repeating the
picture resampling path:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=.../WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
micro-fixture crop mismatch: 19868 differing pixels
diff bounds: x=0..371 y=0..53
object: cNvPr id=8 name="TextBox 7"
source: rect textbox, accent5 fill with lumMod=20000/lumOff=80000, centered bold text
suspected gap: text shaping, font metrics, paragraph layout, or text anchoring
```

Manual crop histogram check showed the mismatch is not just text layout:
dominant got fill is `#E0EBF6`, dominant reference fill is `#E1EBF5`, and the
reference crop has full-width white rows that the got crop does not. That points
at shape color/height behavior around `spAutoFit` and theme luminance
conversion before any text-layout change is justified.

Harness change: `microFixtureTargetScope` now records `top_got_colors`,
`top_reference_colors`, `top_different_got_colors`, and
`top_different_reference_colors`. This keeps the color/fill evidence inside the
object fixture manifest instead of requiring ad hoc external histogram commands.

Added an opt-in shape object profile:

```text
PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST=.../shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_OBJECT_PROFILE_OUTPUT=.../shape-profile-current.json \
go test ./internal/render -run TestMicroFixtureShapeObjectProfile -count=1 -v
```

Current profile:

```text
diff: 19868 pixels
geometry target: x=94..465 y=116..166
shape-autofit text target: x=94..465 y=116..169
text bounds before fit: x=101..458 y=120..162
text bounds after fit: x=101..458 y=120..165
measured text: 351x46 pixels
current fill: #DEEBF7/FF
dominant got color: #E0EBF6/FF, 15584 pixels
dominant reference color: #E1EBF5/FF, 12976 pixels
second reference color: #FFFFFF/FF, 1860 pixels
top lighter-reference rows: 48, 49, 50, 51 each with 372 pixels
```

Tested a narrow renderer hypothesis: keep geometry fill/outline at the original
shape target and apply `spAutoFit` expansion only to the text target. Result:

```text
TextBox 7 fixture mismatch improved from 19868 to 18752 differing pixels
new bounds: x=0..371 y=0..50
```

Decision: rejected and reverted. It removes the bottom overpaint symptom but the
object fixture still fails by a large margin, so it does not meet the
object-fixture acceptance rule. The remaining mismatch still needs a more
specific attributed explanation, likely combining theme luminance/fill
conversion and text metrics/placement.

Added a second opt-in diagnostic that searches simple shape fill/height
candidates without changing production rendering. It replaces only the current
dominant fill-like pixels and optionally stops painting below candidate shape
heights while preserving current text coverage.

```text
PUPPT_SHAPE_FILL_HEIGHT_SEARCH_MANIFEST=.../shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_FILL_HEIGHT_SEARCH_OUTPUT=.../shape-fill-height-search-current.json \
go test ./internal/render -run TestMicroFixtureShapeFillHeightSearch -count=1 -v
```

Result:

```text
baseline: 19868 differing pixels
best candidate: fill #E1EBF5/FF, height 49px
best result: 7347 differing pixels, bounds x=0..371 y=5..48
next height candidates:
  height 48px: 7347 pixels, slightly worse channel delta
  height 50px: 7719 pixels
  height 51px: 8091 pixels
```

Decision: diagnostic only. Matching the dominant reference fill and reducing the
painted height explains a large part of the TextBox 7 failure, but it still
does not pass the object fixture. A renderer change on fill color or
`spAutoFit` height alone is not acceptable.

Added a residual text/ink profile after applying the best fill/height
normalization candidate:

```text
PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_MANIFEST=.../shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_OUTPUT=.../shape-residual-text-profile-current.json \
go test ./internal/render -run TestMicroFixtureShapeResidualTextProfile -count=1 -v
```

Result:

```text
normalized residual: 7347 differing pixels
either side text-like: 3767 pixels
both sides text-like: 295 pixels
reference text-like: 2227 pixels
got text-like: 1835 pixels
reference fill-like: 2900 pixels
got fill-like: 3266 pixels
top residual row: y=48, 372 pixels
top text-like residual rows: y=37 (209), y=33 (200), y=39 (189), y=40 (181)
top luma deltas: -121 (967), +121 (652), +2 (384)
```

Decision: diagnostic only. After the fill/height normalization, the residual is
not purely text placement: it is a mixed text/fill/antialias problem. A broad
font metrics or text-layout experiment would still violate the attribution rule.

Added an opt-in DrawingML luminance color candidate search for the same source
fill (`accent5`, `lumMod=20000`, `lumOff=80000`):

```text
PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_MANIFEST=.../shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_OUTPUT=.../shape-luminance-color-search-current.json \
go test ./internal/render -run TestMicroFixtureShapeLuminanceColorSearch -count=1 -v
```

Result:

```text
base theme color: #5B9BD5/FF
got dominant output color: #E0EBF6/FF
reference dominant output color: #E1EBF5/FF

current-hsl:       internal #DEEBF7/FF, output #E0EBF6/FF, distance-to-reference 1
encoded-rgb-round: internal #DEEBF7/FF, output #E0EBF6/FF, distance-to-reference 1
encoded-rgb-ceil:  internal #DFEBF7/FF, output #E1EBF6/FF, distance-to-reference 1
encoded-rgb-floor: internal #DEEBF6/FF, output #E0EBF5/FF, distance-to-reference 1
linear-rgb-round:  internal #D1DDEE/FF, output #D3DDEC/FF, distance-to-reference 14
```

Decision: diagnostic only. The current luminance path is already within one
channel of the dominant reference fill after Display P3 conversion. A
color-formula-only renderer change is not justified by this object.

Decision: diagnostic only. Do not land a renderer change from this object until
the TextBox 7 object fixture passes and the full 61-slide corpus has no
regression.

### WHO HIV Slide 015, TextBox 7: source/spec checkpoint rerun

Rechecked the `TextBox 7` source XML against the maintained ECMA-376 schema
bundle before considering any production change:

```text
source object:
  <p:sp> cNvPr id=8 name="TextBox 7"
  <p:cNvSpPr txBox="1"/>
  xfrm x=1191129 y=1468901 cx=4728410 cy=646331
  prstGeom rect
  solidFill schemeClr accent5 lumMod=20000 lumOff=80000
  bodyPr wrap="square" rtlCol="0" with spAutoFit
  two centered bold paragraphs, direct run color srgbClr 0070C0

ECMA anchors:
  dml-main.xsd:667-680    color choice includes schemeClr and srgbClr
  dml-main.xsd:1577-1590  fill properties include solidFill
  dml-main.xsd:2610-2624  text autofit includes spAutoFit
  dml-main.xsd:2625-2659  text body properties and body content model
```

Current commands:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_OBJECT_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/shape-profile-current-rerun.json \
go test ./internal/render -run TestMicroFixtureShapeObjectProfile -count=1 -v

PUPPT_SHAPE_FILL_HEIGHT_SEARCH_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_FILL_HEIGHT_SEARCH_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/shape-fill-height-search-current-rerun.json \
go test ./internal/render -run TestMicroFixtureShapeFillHeightSearch -count=1 -v
```

Results:

```text
focused fixture: 19868 crop differing pixels, bounds x=0..371 y=0..53
geometry target: x=94..465 y=116..166
shape-autofit text target: x=94..465 y=116..169
text bounds before fit: x=101..458 y=120..162
text bounds after fit: x=101..458 y=120..165
measured text: 351x46
resolved fill: #DEEBF7/FF
dominant got fill: #E0EBF6/FF
dominant reference fill: #E1EBF5/FF
second reference color: #FFFFFF/FF
best fill/height diagnostic: #E1EBF5/FF at 49px -> 7347 differing pixels
```

Decision: source/spec checkpoint only. `spAutoFit` and the theme luminance fill
are both involved, but existing diagnostics show neither a shape-height-only
change nor a color-formula-only change passes the fixture. The next TextBox
step must isolate one production primitive that can explain the remaining
mixed fill/text/antialias residual, then prove it on this fixture and a
neighboring text-box fixture before any renderer edit is accepted.

### WHO HIV Slide 012, Rectangle 5: next clean non-picture shape target

Regenerated the ownership summary with an absolute output path:

```text
PUPPT_MICRO_FIXTURE_ROOT=.../object-debug-2026-05-31 \
PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=.../micro-fixture-ownership-summary-current.json \
go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v
```

Result:

```text
total=170 scoped=170 clean_failures=70 contaminated_failures=73 partial_underpaint_failures=9
clean picture-contour candidate remains WHO HIV slide 015 Picture 4 at 1200 pixels
first clean non-picture failure: WHO HIV slide 012 Rectangle 5 at 7423 pixels
```

Focused verifier:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=.../WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
visible crop mismatch: 7423 differing pixels
diff bounds: x=0..959 y=0..78
object: cNvPr id=6 name="Rectangle 5"
source: rect, fill #0070C0, style line #2F528F width=12700 EMU, centered bold 40pt white text
text: "                    Ordering Test Kits "
occluded by: Picture Placeholder 8 over x=788..925 y=18..58, masked in visible crop
```

Shape profile:

```text
PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST=.../shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_OBJECT_PROFILE_OUTPUT=.../shape-profile-current.json \
go test ./internal/render -run TestMicroFixtureShapeObjectProfile -count=1 -v
```

Current profile:

```text
geometry target: x=0..959 y=0..77
text bounds: x=7..952 y=4..73
measured text: 484x50 pixels
fill: #0070C0/FF
dominant got/reference color: #2F6EBA/FF
top differing rows: y=0,77,78 each 960 pixels, then text rows y=31,49,32,34,33
top differing columns: x=0 and x=959 each 79 pixels, then text columns
```

Visual inspection of `got-visible-crop.png` and `reference-visible-crop.png`
shows the background and later-object mask are aligned, while the label is
lower in the current render than in the Apple Notes reference. The full-width
top/bottom rows and side columns also implicate rectangle stroke/fractional
edge handling. Decision: diagnostic only. This is a valid clean shape target,
but no production change is accepted yet; the next experiment should search
stroke placement/edge coverage and centered-text vertical placement together
against this object fixture.

Added an opt-in text/stroke profile for clean shape fixtures:

```text
PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=.../shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=.../shape-text-stroke-profile-current.json \
go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v
```

Result:

```text
baseline: 7423 differing pixels
got text mask: x=190..479 y=26..59, 2927 pixels
reference text mask: x=190..479 y=22..55, 3560 pixels
reference top minus got top: -4 px
reference center minus got center: -4 px
edge-band differences: 3032 pixels
top/bottom edge rows: y=0,77,78 each 960 pixels
text-like differences: 3539 pixels
non-text differences: 3884 pixels
best diagnostic text-mask shift: -4 px, 5536 differing pixels, abs channel delta 720532
oracle edge-band replacement only: 4391 differing pixels, abs channel delta 1213353
best text shift plus oracle edge-band replacement: 2504 differing pixels, abs channel delta 341016
```

Decision: diagnostic only. The `-4 px` text-mask result proves a vertical
placement component, but it still leaves the object fixture far from passing.
The 3032 edge-band pixels mean a renderer fix for this object probably needs a
stroke/fractional edge explanation alongside centered-text vertical placement.
The combined oracle diagnostic still leaves 2504 differing pixels, so even a
perfect two-pixel edge-band correction plus the best simple vertical text shift
does not make the object pass. Do not land a broad centered-text shift or a
stroke-only change from this profile.

Extended the same profile with a fixture-local font candidate diagnostic. It
erases current text-like pixels from the object crop, redraws the real object
text in the real text bounds with candidate font families plus small vertical
shifts, and compares the result to the Apple Notes object crop.

Result:

```text
best font candidate: Calibri/shift-y_-4
resolved font: ~/.cache/puppt/fonts/microsoft-word-16.109.26052523/.../DFonts/Calibrib.ttf
best font candidate diff: 5429 pixels, abs channel delta 666049
candidate text mask: x=189..479 y=22..55, 3243 pixels
reference text mask: x=190..479 y=22..55, 3560 pixels
current simple text-mask shift: 5536 pixels
Arial/Helvetica/Aptos-family candidates: >=9506 pixels and much wider text bounds
```

Decision: diagnostic only. The best candidate is the exact Office Calibri Bold
font already available to the renderer, and it only improves slightly over the
mask-shift diagnostic. The remaining mismatch is not explained by a missing
Calibri substitute or by switching to Arial/Helvetica/Aptos. Do not land a font
family substitution from this object.

## 2026-06-01 - Phase 5.2 Rectangle 5 Source/Path Audit And Rejected Fractional Outline Candidate

Target:

```text
testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json
```

Source object:

```text
deck: testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx
slide: 12
cNvPr: id=6 name="Rectangle 5"
geometry: rect
transform: x=0 y=1 cx=12192000 cy=996758
fractional bounds: x=0..960 y=0.00007874015748031496..78.48496062992126
fill: direct srgbClr 0070C0
line: style lnRef idx=2, accent1 shade 50000, resolved #2F528F width=12700 EMU
text: bodyPr anchor=ctr, bold 40pt "                    Ordering Test Kits "
```

Current production path inspected before editing:

```text
renderShape
  fillShapeRectWithFloatBounds for rect fill fractional coverage
  drawStyledRectOutlineAlignedWithCap for rect stroke using snapped integer target
  drawShapeTextForElement
  drawShapeTextWithDPI
  anchoredTextTop
  textBounds
```

Focused verifier:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=.../shape-0001-6-Rectangle-5/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
visible crop mismatch: 7423 differing pixels
diff bounds: x=0..959 y=0..78
```

Profiles generated:

```text
PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST=.../shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_OBJECT_PROFILE_OUTPUT=.../shape-object-profile.json \
go test ./internal/render -run TestMicroFixtureShapeObjectProfile -count=1 -v

PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=.../shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=.../shape-text-stroke-profile.json \
go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v

PUPPT_SHAPE_FILL_HEIGHT_SEARCH_MANIFEST=.../shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_FILL_HEIGHT_SEARCH_OUTPUT=.../shape-fill-height-search.json \
go test ./internal/render -run TestMicroFixtureShapeFillHeightSearch -count=1 -v

PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_MANIFEST=.../shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_OUTPUT=.../shape-residual-text-profile.json \
go test ./internal/render -run TestMicroFixtureShapeResidualTextProfile -count=1 -v
```

Diagnostic results:

```text
shape object profile: diff=7423, geometry target x=0..959 y=0..77, text bounds x=7..952 y=4..73
text/stroke profile: got text x=190..479 y=26..59, reference text x=190..479 y=22..55
edge-band differences: 3032 pixels
best diagnostic text-mask shift: -4 px -> 5536 differing pixels
best font candidate: Calibri/shift-y_-4 -> 5429 differing pixels
fill/height search best: dominant-fill-replacement/#2F6EBA/FF/79px -> 7397 differing pixels
residual after fill/height normalization: 7397 differing pixels
luminance color search: rejected as not applicable because source fill is direct srgbClr, not schemeClr
```

Rejected production candidate:

```text
candidate: draw non-dashed rectangle outlines from fractional DrawingML bounds
unit evidence: focused geometry/stroke tests passed
object evidence: Rectangle 5 fixture still failed with 7423 differing pixels
secondary signal: total absolute channel delta improved, but pixel acceptance did not
```

Decision: reject and revert the production candidate. The fractional outline
idea is source-plausible and lowered channel error, but it did not pass the
object fixture. Under the renderer completion rule, this is not an acceptable
production renderer change. The remaining Rectangle 5 failure still requires a
combined source-backed explanation for edge/stroke coverage and centered text
placement; do not land stroke-only, fill-height-only, font-substitution, or
generic vertical-shift changes from these diagnostics.

## 2026-06-01 - Phase 5.3 Rectangle 5 Text Anchoring Spec Check And Forward Path

Local spec sources:

```text
docs/specs/ecma-376/part1/Ecma Office Open XML Part 1 - Fundamentals And Markup Language Reference.pdf
docs/specs/ecma-376/part1/schema/strict/dml-main.xsd
docs/specs/ecma-376/SHA256SUMS
```

Relevant schema anchors from the maintained ECMA-376 bundle:

```text
dml-main.xsd:2547-2555  ST_TextAnchoringType allows t, ctr, b, just, dist
dml-main.xsd:2625-2652  CT_TextBodyProperties owns anchor, insets, overflow, wrap, autofit, anchorCtr, compatLnSpc
dml-main.xsd:2653-2659  CT_TextBody is bodyPr, optional lstStyle, and one or more paragraphs
```

Supplemental searchable SDK prose says `bodyPr@anchor` represents the
anchoring position of the `txBody` within the shape, omitted value implies top,
and `anchor="ctr"` vertically aligns the text in the center of the containing
shape. It also documents the ECMA-derived default text-box inset values used by
the current parser: left/right 91440 EMU and top/bottom 45720 EMU.

Rectangle 5 source text body:

```text
<p:txBody>
  <a:bodyPr rtlCol="0" anchor="ctr"/>
  <a:lstStyle/>
  <a:p>
    <a:r>
      <a:rPr lang="en-ES" sz="4000" b="1" dirty="0"/>
      <a:t>                    Ordering Test Kits </a:t>
    </a:r>
  </a:p>
</p:txBody>
```

Current renderer path inspected:

```text
parseBodyProperties -> TextAnchor="ctr"
paragraphTextRunsWithTheme -> trims run collection edges and preserves the run text used for drawing
textBounds -> default DrawingML insets because no lIns/tIns/rIns/bIns are authored
drawShapeTextWithDPI -> textRenderLinesForElement -> measureTextRenderLines
measuredTextAnchorHeight -> visible ascent/descent for ctr/b anchors
anchoredTextTop -> centers that measured height in text bounds
```

Fixture evidence:

```text
focused object fixture: 7423 differing visible-crop pixels
got text mask: x=190..479 y=26..59
reference text mask: x=190..479 y=22..55
reference minus got text top: -4 px
reference minus got text center: -4 px
edge-band residual: 3032 pixels
best glyph-mask-only shift: -4 px -> 5536 pixels, still fails
best font candidate: Calibri/shift-y_-4 -> 5429 pixels, still fails
best text shift plus oracle edge-band replacement: 2504 pixels, still fails
```

Concrete forward path:

1. Add a diagnostic that renders Rectangle 5's parsed text body with
   text-body line-box anchoring, not a post-render glyph-mask shift.
2. The diagnostic must output measured line metrics, current anchor height,
   candidate line-box anchor height, resulting baseline/top, and object diff.
3. Compare against the focused Rectangle 5 fixture and at least one neighboring
   centered text fixture before production edit.
4. Accept a production text-layout change only if it is described as an
   `anchor="ctr"` text-body line-box rule, not as an arbitrary pixel shift, and
   only if the object fixture plus same-family neighbors justify it.
5. If the line-box rule improves but does not pass, keep it as diagnostic and
   continue the combined edge/stroke plus text-body explanation instead of
   landing it alone.

Decision: no production text-layout change yet. The next implementation step is
a source-backed diagnostic for `anchor="ctr"` line-box anchoring. Broad text
shifts, font substitution, and glyph-mask-only repairs remain rejected.

## 2026-06-01 - Phase 5.3 Rectangle 5 Line-Box Anchor Diagnostic

Added a test-only diagnostic to `TestMicroFixtureShapeTextStrokeProfile` that
keeps the parsed DrawingML text body and compares current centered text
placement with a candidate using full text line-box height for
`anchor="ctr"`. This is diagnostic-only; production text layout was not
changed.

Commands:

```text
PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/shape-text-stroke-profile-linebox-anchor.json \
go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v

PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-010/micro-fixtures/shape-0002-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-010/micro-fixtures/shape-0002-6-Rectangle-5/shape-text-stroke-profile-linebox-anchor.json \
go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v

PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/shape-0001-6-Rectangle-5/shape-text-stroke-profile-linebox-anchor.json \
go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v
```

Results:

```text
slide 012 Rectangle 5:
  baseline diff: 7423 pixels
  text anchor: ctr
  text bounds: x=7..952 y=4..73
  line metrics: one line, ascent=39, descent=11, height=50
  current anchor height: 50
  line-box anchor height: 50
  line-box shift: 0 px
  current-visible-anchor candidate: 7421 pixels
  line-box-anchor candidate: 7421 pixels

slide 010 Rectangle 5 neighbor:
  baseline diff: 13320 pixels
  current anchor height: 46
  line-box anchor height: 46
  line-box shift: 0 px
  current-visible-anchor candidate: 13306 pixels
  line-box-anchor candidate: 13306 pixels

slide 009 Rectangle 5 neighbor:
  baseline diff: 18027 pixels
  current anchor height: 78
  line-box anchor height: 78
  line-box shift: 0 px
  current-visible-anchor candidate: 18002 pixels
  line-box-anchor candidate: 18002 pixels
```

Focused fixture gate:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result: still fails with 7,423 visible-crop differing pixels.

Decision: reject a production line-box anchor change for this target. The local
ECMA schema confirms `bodyPr@anchor="ctr"` is authored and valid, but the
candidate line-box calculation is identical to the current calculation for the
focused object and two same-family neighbors. The remaining `-4px` text-mask
offset is therefore not explained by current visible-height versus line-box
anchoring. Continue with a combined source-backed explanation for font metrics,
glyph rendering, and rectangle edge/stroke coverage before any production
renderer edit.

## 2026-06-01 - Phase 5.4 TextBox 7 Parsed Text Over Normalized Fill/Height

Local spec sources:

```text
docs/specs/ecma-376/part1/Ecma Office Open XML Part 1 - Fundamentals And Markup Language Reference.pdf
docs/specs/ecma-376/part1/schema/strict/dml-main.xsd
docs/specs/ecma-376/SHA256SUMS
```

Relevant schema anchors from the maintained ECMA-376 bundle:

```text
dml-main.xsd:667-680    CT_Color color choices include schemeClr and srgbClr
dml-main.xsd:1577-1590  EG_FillProperties includes solidFill
dml-main.xsd:2610-2624  CT_TextAutofit includes spAutoFit
dml-main.xsd:2625-2652  CT_TextBodyProperties owns wrap, insets, overflow, and autofit
dml-main.xsd:2653-2659  CT_TextBody is bodyPr, optional lstStyle, and one or more paragraphs
```

Focused source object:

```text
WHO HIV slide 015, cNvPr id=8, name="TextBox 7"
p:cNvSpPr txBox="1"
xfrm x=1191129 y=1468901 cx=4728410 cy=646331
solidFill schemeClr accent5 lumMod=20000 lumOff=80000
bodyPr wrap="square" rtlCol="0" with a:spAutoFit
two centered bold paragraphs with direct srgbClr 0070C0 text
```

Current renderer path inspected:

```text
parseBodyProperties -> parses spAutoFit
renderShape -> computes the shape target and fill
shapeAutofitTarget -> measures text and expands target height from y=166 to y=169
fillShapeRectWithFloatBounds -> paints the expanded fill
textBounds -> applies default DrawingML insets
drawShapeTextWithDPI -> renders parsed paragraphs and runs
```

Focused diagnostics:

```text
PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/shape-residual-text-profile-current-rerun.json \
go test ./internal/render -run TestMicroFixtureShapeResidualTextProfile -count=1 -v

PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/shape-luminance-color-search-current-rerun.json \
go test ./internal/render -run TestMicroFixtureShapeLuminanceColorSearch -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Focused results:

```text
fixture gate: 19868 differing pixels, bounds x=0..371 y=0..53
luminance color search: current output #E0EBF6 is one channel from reference #E1EBF5
fill/height normalization: #E1EBF5 at 49 px improves to 7347 differing pixels but still fails
residual after normalization: either-side text-like 3767 pixels, both-side text-like 295 pixels
parsed text over normalized fill/height:
  shape-autofit-text-bounds x=101..458 y=120..165 -> 7692 differing pixels
  source-geometry-text-bounds x=101..458 y=120..162 -> 7692 differing pixels
```

Neighboring TextBox evidence:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json \
go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json \
PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/shape-residual-text-profile-current-rerun.json \
go test ./internal/render -run TestMicroFixtureShapeResidualTextProfile -count=1 -v
```

Neighbor results:

```text
slide 003 TextBox 7 fixture gate: 132995 visible-crop differing pixels
neighbor residual after fill/height normalization: 37520 differing pixels
neighbor parsed text over normalized fill/height:
  shape-autofit-text-bounds -> 38022 differing pixels
  source-geometry-text-bounds -> 38022 differing pixels
```

Decision: reject a production parsed-source-text redraw, geometry text-bounds,
or `spAutoFit` text-bounds change for the current TextBox target. The source
and ECMA schema justify testing text body bounds and `spAutoFit`, but the
diagnostic makes the focused object worse and has the same rejection signal on a
neighboring `txBox`/`spAutoFit`/accent5 TextBox. The remaining residual is still
mixed fill-height, glyph rasterization, and antialias/coverage behavior; no
production TextBox renderer change is accepted from this checkpoint.

Validation after adding the test-only diagnostic and project-log evidence:

```text
git diff --check
go test ./internal/render -count=1
go test ./...
shasum -a 256 -c docs/specs/ecma-376/SHA256SUMS
```

All passed on 2026-06-01.

## 2026-06-01 - Production Backend Path And Failure Scoreboard

Added a reusable scoreboard test that summarizes the current object-debug
artifact tree into primitive-level production priorities:

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

Clean isolated object-fixture groups:

```text
picture clean failures: 46 fixtures, 1,499,584 differing pixels
shape clean failures: 24 fixtures, 550,448 differing pixels
```

Decision: the production path is no longer more broad pixel tuning. The
evidence supports a backend split: Puppt keeps Open XML/package interpretation
and introduces bounded primitive backends for vector geometry, text shaping, and
picture sampling/color. The maintained path is now recorded in
`docs/RENDERER_PRODUCTION_PATH.md`, with dependency candidates and the first
backend checkpoints. No renderer production behavior was changed by this
scoreboard step.

## 2026-06-01 - draw2d Rectangle Backend Diagnostic

Added a controlled primitive dependency and a test-only vector backend profile:

```text
go get github.com/llgcode/draw2d@v0.0.0-20260422081035-c4331ac66734
go mod tidy
```

The diagnostic adapter:

- consumes Puppt-parsed DrawingML shape geometry only
- renders rectangle fill and stroke with `draw2d`
- converts shape/text colors into Puppt's existing Display P3 output space
  because fixture crops are captured after production output conversion
- redraws parsed text using Puppt's current text renderer
- does not read, write, render, mutate, validate, or interpret `.pptx` packages

Focused and same-family commands:

```text
PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/shape-vector-backend-profile.json \
go test ./internal/render -run TestMicroFixtureShapeVectorBackendProfile -count=1 -v

PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-010/micro-fixtures/shape-0002-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-010/micro-fixtures/shape-0002-6-Rectangle-5/shape-vector-backend-profile.json \
go test ./internal/render -run TestMicroFixtureShapeVectorBackendProfile -count=1 -v

PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/shape-0001-6-Rectangle-5/shape-vector-backend-profile.json \
go test ./internal/render -run TestMicroFixtureShapeVectorBackendProfile -count=1 -v

PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-013/micro-fixtures/shape-0005-4-TextBox-3/manifest.json \
PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-013/micro-fixtures/shape-0005-4-TextBox-3/shape-vector-backend-profile.json \
go test ./internal/render -run TestMicroFixtureShapeVectorBackendProfile -count=1 -v
```

Results:

```text
slide 012 Rectangle 5: baseline 7423, draw2d source-geometry target 7421
slide 010 Rectangle 5: baseline 13320, draw2d source-geometry target 13306
slide 009 Rectangle 5: baseline 18027, draw2d source-geometry target 18000
slide 013 TextBox 3: baseline 25347, draw2d shape-autofit target 25233
```

Decision: keep as a diagnostic backend only. The adapter is now wired and gives
a consistent same-family improvement, but it does not pass any focused object
fixture. Do not replace production rectangle rendering with draw2d yet. The
next vector-backend step is to isolate whether the remaining residual is stroke
placement, antialias coverage, or text metrics by adding candidate image output
and separating fill/stroke/text layers.

Follow-up layer split:

```text
slide 012 Rectangle 5: baseline 7423, best fill/stroke-only candidate 7368
slide 010 Rectangle 5: baseline 13320, best fill/stroke-only candidate 12823
slide 009 Rectangle 5: baseline 18027, best fill/stroke-only candidate 15266
slide 013 TextBox 3: baseline 25347, best fill/stroke-only candidate 16467
```

Decision update: draw2d fill/stroke rasterization remains a useful diagnostic
candidate, but adding the current text renderer back into the composite erases
most of the gain. The production fix should move to text placement/metrics and
raster composition before replacing rectangle painting wholesale.

## 2026-06-01 - go-text Text Shaping Diagnostic

Added a controlled primitive dependency and a test-only text shaping profile:

```text
go get github.com/go-text/typesetting@v0.3.4
go mod tidy
```

The diagnostic adapter:

- consumes Puppt-resolved text runs, font family, bold/italic, character
  spacing, tab stops, and effective font size
- resolves the same local font bytes as the production font cache
- applies the same `1800` default font size used by `openFontFaceWithDPI` when
  a run and element omit size
- compares current x/image segment widths with HarfBuzz-shaped advances
- does not read, write, render, mutate, validate, or interpret `.pptx` packages

Focused commands:

```text
PUPPT_SHAPE_TEXT_SHAPING_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json \
PUPPT_SHAPE_TEXT_SHAPING_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/shape-text-shaping-profile.json \
go test ./internal/render -run TestMicroFixtureShapeTextShapingProfile -count=1 -v

PUPPT_SHAPE_TEXT_SHAPING_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json \
PUPPT_SHAPE_TEXT_SHAPING_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/shape-text-shaping-profile.json \
go test ./internal/render -run TestMicroFixtureShapeTextShapingProfile -count=1 -v

PUPPT_SHAPE_TEXT_SHAPING_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-013/micro-fixtures/shape-0005-4-TextBox-3/manifest.json \
PUPPT_SHAPE_TEXT_SHAPING_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-013/micro-fixtures/shape-0005-4-TextBox-3/shape-text-shaping-profile.json \
go test ./internal/render -run TestMicroFixtureShapeTextShapingProfile -count=1 -v
```

Results:

```text
slide 012 Rectangle 5: 1 line, 1 segment, max advance delta 1 px
slide 015 TextBox 7: 2 lines, 2 segments, max advance delta 2 px
slide 013 TextBox 3: 8 lines, 8 segments, max advance delta 5 px
```

Decision: keep go-text as a diagnostic backend only for now. Missing HarfBuzz
advance shaping does not explain these residuals. The next text work should
profile vertical placement, line box metrics, baseline placement, and how text
coverage is composited into the Display P3 output.

## 2026-06-01 - Picture Pipeline Split Diagnostic

Added `TestMicroFixturePicturePipelineProfile` as the source-backed split
required before any more picture-rendering experiments. The diagnostic opens an
attributed picture micro-fixture and records these stages from the current
production path:

- source decode from the fixture image relationship
- source color as decoded sRGB bytes and the final Display P3 conversion
- DrawingML `srcRect` crop, whose absent values default to zero under
  `CT_RelativeRect`
- flip and `alphaModFix` transform stage
- integer EMU target sampling with the current `pictureScaler`
- final output crop, including visible-crop occlusion masking when present

Focused commands:

```text
PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v
```

Results:

```text
WHO slide 015 Picture 4:
  source: ppt/media/object.png, image/png, *image.Paletted, 200x200, 39 colors, 40000 opaque pixels
  crop: full source; no flip; no alphaModFix; scaler approx_bilinear
  Display P3 source-color delta: 0 pixels
  staged output vs current got: 0 differing pixels
  staged output vs reference: 1200 differing pixels

EPA slide 004 Google Shape;11;p15:
  source: ppt/media/object.png, image/png, *image.Paletted, 421x120, 70 colors, 50520 opaque pixels
  crop: full source; no flip; alphaModFix authored at 100000 but not applied; scaler approx_bilinear
  Display P3 source-color delta: 3029 pixels, max channel delta 34
  staged output vs current visible got: 0 differing pixels
  staged output vs visible reference: 2127 differing pixels
```

Decision: keep this as the picture pipeline ledger and do not accept a
production picture change yet. The diagnostic now proves the fixture residuals
are reproduced by the current documented stages, so the next picture change has
to target a named stage and pass both fixture references rather than rerunning
ungrounded scaler/color searches.

## 2026-06-01 - Renderer Scene Boundary Start

Architecture decision: the way to finish the renderer is to stop treating
`slideElement` as both the resolved OOXML model and the backend paint contract.
Puppt must own OOXML interpretation, then lower resolved objects into stable
render primitives consumed by replaceable primitive backends.

Added the first code boundary:

```text
internal/render/render_scene.go
internal/render/render_scene_test.go
```

The initial primitive is `renderPicturePrimitive`. It preserves:

- object id/name and source part
- image relationship id, resolved media part, and content type
- integer and fractional target bounds
- DrawingML `srcRect` crop percentages
- flip, `alphaModFix`, rotation, and `rotWithShape`
- soft edge, custom mask, and line metadata

Focused validation:

```text
go test ./internal/render -run 'TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestRenderSceneFromElementsKeepsPictureZOrderAndErrors' -count=1 -v
```

Result: passed.

Decision: this is the production route of record. The next implementation step
is not another picture residual search; it is to move the existing picture
painting code behind a `renderPicturePrimitive` backend with zero pixel change,
then replace the backend stage by stage under object-fixture gates.

## 2026-06-01 - Picture Backend Zero-Diff Migration

Implemented the next route step: `renderPicture` now lowers the resolved
picture object into `renderPicturePrimitive` and calls `currentPictureBackend`
through a `pictureBackend` interface before painting.

This is intentionally a parity-preserving migration. The backend input still
carries the legacy resolved `slideElement` while the remaining paint fields are
promoted into `renderPicturePrimitive`; that follow-up is tracked in the
checklist. The important boundary is now in production code: picture rendering
has an explicit backend call site fed by a scene primitive.

Compatibility note: production `renderPicture` also paints shape-level blip
fills, so `renderPicturePrimitiveFromElement` now accepts `pic`, `sp`, and
`cxnSp` picture-backed elements and records the original object kind.

Focused validation:

```text
go test ./internal/render -run 'TestRenderPicture|TestPicture|TestDrawPictureRaster|TestRenderPaintsEmbeddedPNGPicture' -count=1

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v
```

Results:

```text
focused picture and shape blip-fill tests: passed
WHO Picture 4: got_delta=0, reference_delta=1200
EPA Google Shape;11;p15: got_delta=0, reference_delta=2127
```

Decision: the zero-diff backend migration is accepted as structural progress,
not as a visual parity fix. No picture residual is closed yet. The next
implementation step is to remove the backend's legacy `slideElement` dependency
and make `renderPicturePrimitive` the complete picture paint contract.

## 2026-06-01 - Picture Backend Primitive Contract Completion

Removed the transitional `*slideElement` from `pictureBackendInput`. The current
picture backend now paints from `renderPicturePrimitive` plus decoded image
data, canvas, target part, and slide size.

Promoted primitive-owned paint fields:

- object kind and SVG fallback relationship id
- crop, flip, `alphaModFix`, rotation, and `rotWithShape`
- soft edge radius
- custom mask path, commands, and unsupported messages
- line color, width, dash, alignment, and cap
- shadow color, blur, distance, direction, scale/skew flags, and
  rotate-with-shape flag
- 3-D unsupported feature metadata

Focused validation:

```text
go test ./internal/render -run 'TestRenderPicturePrimitive|TestPictureSourceForElement|TestRenderPicture|TestPicture|TestDrawPictureRaster|TestRenderPaintsEmbeddedPNGPicture|TestRenderElementsPaintsShape.*BlipFill' -count=1

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v
```

Results:

```text
focused primitive and picture/blip-fill tests: passed
WHO Picture 4: got_delta=0, reference_delta=1200
EPA Google Shape;11;p15: got_delta=0, reference_delta=2127
```

Decision: accepted as structural progress. The picture backend now has a real
primitive contract. The next implementation step is to replace one backend
stage, starting with sampling/color, only when the replacement passes named
picture fixtures.

## 2026-06-01 - Picture Sampling Stage Boundary

Extracted picture sampling into a backend stage:

```text
pictureSamplingStage
pictureSamplingInput
currentPictureSamplingStage
```

`currentPictureBackend` now passes the primitive, target rectangle, decoded
source image, source bounds, canvas, slide size, and output width into the
sampling stage. The default stage preserves the existing behavior for normal
sampling, soft edge, custom mask, rotation, and rotated-line composition. Tests
can now inject an alternate sampling stage without bypassing the production
backend call path.

Focused validation:

```text
go test ./internal/render -run 'TestCurrentPictureBackendUsesSamplingStage|TestRenderPicturePrimitive|TestRenderPicture|TestPicture|TestDrawPictureRaster|TestRenderPaintsEmbeddedPNGPicture|TestRenderElementsPaintsShape.*BlipFill' -count=1

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json \
PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/picture-pipeline-profile.json \
go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v
```

Results:

```text
focused sampling-stage and picture/blip-fill tests: passed
WHO Picture 4: got_delta=0, reference_delta=1200
EPA Google Shape;11;p15: got_delta=0, reference_delta=2127
```

Decision: accepted as structural progress only. No visual residual is closed by
this step. The next accepted production change must replace the sampling stage
with an implementation that passes the named picture fixtures.

## 2026-06-01 - Picture Sampling Stage Acceptance Gate

Added the opt-in fixture gate for promoting a replacement sampling stage:

```text
PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE=1 go test ./internal/render -run TestCurrentPictureSamplingStageAcceptanceGate -count=1 -v
```

The test renders the two current picture gates through the `pictureSamplingStage`
backend hook rather than through a separate diagnostic-only path:

- WHO slide 015 `Picture 4`
- EPA slide 004 `Google Shape;11;p15`

Results:

```text
WHO Picture 4 current sampling residual: 1200 pixels on crop bounds={MinX:699 MinY:360 MaxX:788 MaxY:451}
EPA Google Shape;11;p15 current sampling residual: 2127 pixels on visible crop bounds={MinX:28 MinY:474 MaxX:221 MaxY:528}
```

Spec/source basis checked in this step:

- `swe_skill.md` Definition of Done requires source docs, current code-path
  inspection, a coherent source-backed implementation step, tests/fixtures,
  validation, and honest output; M12 keeps failing supported-scope gates in the
  implementation queue unless source evidence proves a static-renderer
  impossibility.
- `docs/specs/ecma-376/README.md` anchors DrawingML picture content,
  `srcRect`, stretch/fill, and blip-fill schema paths.
- `docs/specs/ms-odrawxml/README.md` remains the source for Microsoft Drawing
  extensions such as `useLocalDpi`, but no extension-derived behavior is
  accepted by this gate.

Decision: accepted as an implementation gate, not as visual renderer progress.
The renderer still has the same picture residuals. A sampling-stage replacement
is promotable only when this gate reports zero residual for both named fixtures
or when the checklist is updated with a source-backed fixture target change.

## 2026-06-01 M04 Coordinates, Transforms, And Clipping Closeout

Milestone: `docs/renderer-milestones/04-coordinates-transforms-and-clipping.md`.

Source anchors:

- `dml-main.xsd:613 CT_Transform2D`
- `dml-main.xsd:622 CT_GroupTransform2D`
- `pml.xsd:1209 CT_Shape`
- `pml.xsd:1228 CT_Connector`
- `pml.xsd:1245 CT_Picture`
- `pml.xsd:1263 CT_GraphicalObjectFrame`
- `pml.xsd:1282 CT_GroupShape`

Coordinate model:

- Parse-time group composition remains the OOXML source boundary: nested
  `grpSpPr/a:xfrm` `off`, `ext`, `chOff`, and `chExt` are composed into child
  element EMUs before painting.
- `renderElementTransformFor` is the shared EMU-to-pixel boundary for primitive
  lowering, legacy paint targets, text transform targets, table targets, line
  endpoints, object-debug pixel bounds, and object-debug fractional bounds.
- Integer bounds use the renderer's existing rounded EMU scale, while
  fractional bounds preserve the pre-rounded source position for diagnostics and
  future sampling/layout work.
- Raw targets preserve directed extents instead of normalizing through
  `image.Rect`; non-positive extents therefore produce empty object masks while
  zero-width or zero-height connector endpoints can still be drawn through the
  line endpoint path.
- Clipped bounds are the canvas intersection used for object masks and visible
  target diagnostics.
- Rotation is normalized to degrees for primitive/debug metadata. M04 records
  rotation/flip semantics in the shared model and keeps actual geometry/path
  rotation behavior deferred.

Accepted renderer choices where OOXML leaves current behavior underspecified:

- Fractional coordinates are not anti-aliased or resampled in M04; the legacy
  rounded-pixel target remains the production paint target.
- Clipping is rectangle/canvas clipping only. Shape path masks, text overflow
  semantics, picture crop sampling, and soft-edge masks remain later primitive
  work.
- Group transforms are flattened into child EMU coordinates and group container
  primitives remain provenance/debug records, not separate compositing layers.
- Negative extents are treated as non-renderable object masks for non-line
  objects; connector endpoints keep their directed source start/end values.

Changed code/docs:

- `internal/render/render_transform.go`
- `internal/render/render_transform_test.go`
- `internal/render/render_paint.go`
- `internal/render/render_tables.go`
- `internal/render/render_text_layout.go`
- `internal/render/render_object_debug.go`
- `tools/generate_ooxml_drawingml_audit.py`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/renderer-coverage-summary.json`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`
- `docs/RENDERER_EXPERIMENT_LOG.md`

Validation:

```text
python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -run 'Test.*Transform|Test.*Bounds|Test.*Group|TestRenderObjectDebug' -count=1

go test ./internal/render -count=1

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m04-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

git diff --check
```

Results:

```text
coverage summary: passed, queue totals unchanged
focused M04 tests: passed
full internal/render suite: passed
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950955502 mean_channel_rms_similarity=0.829145432 total_diff=9321023
git diff --check: passed
```

Decision: M04 is accepted as a structural coordinate-model milestone. It does
not close geometry path, text shaping, image sampling, effects, or real-world
visual parity residuals. Next checkpoint: M05 theme, color, fill, and style
resolution.

## 2026-06-01 M05 Theme, Color, Fill, And Style Resolution Closeout

Milestone:
`docs/renderer-milestones/05-theme-color-fill-and-style-resolution.md`.

Source anchors:

- `dml-main.xsd:85 CT_ColorScheme`
- `dml-main.xsd:275 EG_ColorTransform`
- `dml-main.xsd:667 EG_ColorChoice`
- `dml-main.xsd:1391 CT_SolidColorFillProperties`
- `dml-main.xsd:1438 CT_GradientFillProperties`
- `dml-main.xsd:1502 CT_BlipFillProperties`
- `dml-main.xsd:1569 CT_PatternFillProperties`
- `dml-main.xsd:1576 CT_GroupFillProperties`
- `dml-main.xsd:1579 EG_FillProperties`
- `dml-main.xsd:2246 CT_StyleMatrixReference`
- `dml-main.xsd:2252 CT_FontReference`
- `dml-main.xsd:2258 CT_ShapeStyle`
- `dml-main.xsd:2284 CT_ColorMapping`
- `dml-main.xsd:2301 CT_ColorMappingOverride`
- `pml.xsd:1209 CT_Shape`
- `pml.xsd:1245 CT_Picture`
- `pml.xsd:1282 CT_GroupShape`
- `pml.xsd:1314 CT_BackgroundProperties`
- `pml.xsd:1328 CT_Background`

Implemented semantics:

- Paint resolution is now a shared source boundary for shape/background/style
  fill consumers instead of backend-specific theme parsing.
- Color choices resolve sRGB, scRGB, HSL, system colors, scheme colors,
  preset colors, and placeholder colors through theme/color-map state.
- Color modifiers are applied in source order, including alpha, hue,
  saturation, luminance, RGB channel, grayscale, inverse, complement, gamma,
  and inverse-gamma transforms.
- Direct fill, style-derived fill, background `bgPr`/`bgRef`, and group
  `grpFill` resolve into stable paint primitives before painting.
- Pattern fills are implemented for the renderer's existing filled shape paths
  and slide backgrounds. Pattern paint is also lowered into scene primitives
  and object-debug summaries.
- Direct shape fill takes precedence over style fill refs, matching the source
  shape-property boundary.

Remaining true partials:

- Blip fill image sampling/tile details remain tied to the later picture
  sampling milestone.
- Advanced gradient clauses and effect rendering beyond style-ref resolution
  remain partial.
- Geometry path completeness, text shaping, and final visual parity remain
  later milestones.

Changed code/docs:

- `internal/render/render_color.go`
- `internal/render/render_paint_style.go`
- `internal/render/render_m05_test.go`
- `internal/render/render_types.go`
- `internal/render/render_parse.go`
- `internal/render/render_shape_parse.go`
- `internal/render/render_background.go`
- `internal/render/render_inheritance_theme.go`
- `internal/render/render.go`
- `internal/render/render_paint.go`
- `internal/render/render_scene.go`
- `internal/render/render_object_debug.go`
- `tools/generate_ooxml_drawingml_audit.py`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/renderer-coverage-summary.json`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`
- `docs/RENDERER_EXPERIMENT_LOG.md`

Validation:

```text
python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -run 'TestM05|Test.*Color|Test.*Theme|Test.*Fill|Test.*Background|Test.*Style' -count=1

go test ./internal/render -count=1

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m05-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

git diff --check
```

Results:

```text
coverage summary: passed; queue totals core-static=16 common-partial=94 hard-rendering=379 unsupported-preserve=444 out-of-scope=74
focused M05/color/theme/fill/background/style tests: passed
full internal/render suite: passed
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950955502 mean_channel_rms_similarity=0.829145432 total_diff=9321023
git diff --check: passed
```

Decision: M05 is accepted. The renderer now implements the feasible
color/fill/style semantics in this milestone rather than treating unsupported
status as a substitute for implementation. Next checkpoint: M06 geometry,
stroke, and connectors.

## 2026-06-01 M06 Geometry, Stroke, And Connectors Closeout

Milestone:
`docs/renderer-milestones/06-geometry-stroke-and-connectors.md`.

Source anchors:

- `dml-main.xsd:1984 CT_ConnectionSite`
- `dml-main.xsd:1996 CT_ConnectionSiteList`
- `dml-main.xsd:2001 CT_Connection`
- `dml-main.xsd:2005 CT_Path2DMoveTo`
- `dml-main.xsd:2010 CT_Path2DLineTo`
- `dml-main.xsd:2015 CT_Path2DArcTo`
- `dml-main.xsd:2021 CT_Path2DQuadBezierTo`
- `dml-main.xsd:2026 CT_Path2DCubicBezierTo`
- `dml-main.xsd:2031 CT_Path2DClose`
- `dml-main.xsd:2042 CT_Path2D`
- `dml-main.xsd:2057 CT_Path2DList`
- `dml-main.xsd:2062 CT_PresetGeometry2D`
- `dml-main.xsd:2074 CT_CustomGeometry2D`
- `dml-main.xsd:2084 EG_Geometry`
- `dml-main.xsd:2096 ST_LineEndType`
- `dml-main.xsd:2120 CT_LineEndProperties`
- `dml-main.xsd:2138 EG_LineJoinProperties`
- `dml-main.xsd:2172 EG_LineDashProperties`
- `dml-main.xsd:2178 ST_LineCap`
- `dml-main.xsd:2191 ST_PenAlignment`
- `dml-main.xsd:2197 ST_CompoundLine`
- `dml-main.xsd:2206 CT_LineProperties`
- `dml-main.xsd:2223 CT_ShapeProperties`
- `pml.xsd:1209 CT_Shape`
- `pml.xsd:1228 CT_Connector`

Implemented semantics:

- Custom geometry parsing now supports multiple `a:path` entries plus
  `moveTo`, `lnTo`, `quadBezTo`, `cubicBezTo`, `arcTo`, and `close`. Unsupported
  unknown commands still report the exact command name.
- Path primitives preserve subpaths, commands, fill flags, stroke flags, and
  unsupported records independent of raw source XML.
- Shape fill, gradient fill, pattern fill, outline, and shadow paths consume
  the custom subpath list instead of a single first path where the current
  backend can paint multiple paths.
- Custom picture masks now rasterize `quadBezTo` and the arc polyline generated
  from DrawingML `arcTo`.
- Stroke parsing/rendering now carries custom dash stops, joins, compound line
  variants, and all schema-defined line-end marker types: triangle, stealth,
  diamond, oval, and arrow.
- Straight connectors use the shared M04 endpoint model plus M06 stroke
  semantics for width, dash, cap, compound line, and head/tail markers.

Remaining true partials:

- The full DrawingML preset geometry catalog and guide formula system are not
  complete; M06 covers the common preset subset already represented in tests
  and current real-world decks.
- Routed connector paths and connection-site routing remain incomplete.
- Gradient/pattern stroke fills remain partial; solid line fills and no-line
  semantics are implemented.
- Bevel/miter joins use the stable segment stroke model; round joins are
  explicitly rendered for supported path outlines.
- Text layout, picture sampling, effects/shadow quality, and final visual
  parity remain later milestones.

Focused real-world fixture evidence:

```text
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result: failed with the known current residual, 7,423 visible-crop differing
pixels for slide 012 object 6 `Rectangle 5`.

Same-family checks:

```text
slide 009 object 6 Rectangle 5: 18,027 visible-crop differing pixels
slide 010 object 6 Rectangle 5: 13,320 visible-crop differing pixels
```

These match the pre-M06 Rectangle 5 residuals recorded in the production backend
path notes. M06 therefore does not regress this same-family shape fixture group,
but it does not close those text/edge residuals.

Changed code/docs:

- `internal/render/render_geometry.go`
- `internal/render/render_paint.go`
- `internal/render/render_pictures.go`
- `internal/render/render_scene.go`
- `internal/render/render_shape_parse.go`
- `internal/render/render_tables.go`
- `internal/render/render_test.go`
- `internal/render/render_types.go`
- `internal/render/render_m06_test.go`
- `tools/generate_ooxml_drawingml_audit.py`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/renderer-coverage-summary.json`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`
- `docs/RENDERER_EXPERIMENT_LOG.md`

Validation:

```text
python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -run 'TestM06|TestRenderShape|Test.*Geometry|Test.*Connector|Test.*Line|Test.*Stroke|Test.*Marker' -count=1

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

go test ./internal/render -count=1

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m06-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

git diff --check
```

Results:

```text
coverage summary: passed; queue totals core-static=16 common-partial=102 hard-rendering=371 unsupported-preserve=444 out-of-scope=74
focused M06 geometry/stroke/connector tests: passed
focused Rectangle 5 micro-fixture: expected documented failure, 7,423 visible-crop differing pixels
full internal/render suite: passed
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950961349 mean_channel_rms_similarity=0.829162565 total_diff=9321380
git diff --check: passed
```

Decision: M06 is accepted. Unsupported status is limited to true remaining gaps,
and unknown custom commands/marker values remain explicitly reported. Next
checkpoint: M07 pictures, media, and image pipeline.

## 2026-06-01 M07 Pictures, Media, And Image Pipeline

Schema anchors:

- `dml-picture.xsd:14 CT_Picture`
- `dml-main.xsd:687 ST_BlackWhiteMode`
- `dml-main.xsd:702 AG_Blob`
- `dml-main.xsd:1242 CT_AlphaBiLevelEffect`
- `dml-main.xsd:1252 CT_AlphaModulateFixedEffect`
- `dml-main.xsd:1258 CT_AlphaReplaceEffect`
- `dml-main.xsd:1261 CT_BiLevelEffect`
- `dml-main.xsd:1268 CT_ColorChangeEffect`
- `dml-main.xsd:1275 CT_ColorReplaceEffect`
- `dml-main.xsd:1280 CT_DuotoneEffect`
- `dml-main.xsd:1291 CT_GrayscaleEffect`
- `dml-main.xsd:1305 CT_LuminanceEffect`
- `dml-main.xsd:1447 CT_TileInfoProperties`
- `dml-main.xsd:1455 CT_StretchInfoProperties`
- `dml-main.xsd:1475 CT_Blip`
- `dml-main.xsd:1502 CT_BlipFillProperties`

Implemented semantics:

- Picture primitives now preserve embedded and linked image relationship ids,
  SVG fallback ids, signed `srcRect` crop/padding, `rotWithShape`, stretch/tile
  fill mode, and source-space blip effects.
- Internal linked image relationships can render through the same package image
  policy as embedded images. External linked targets are not fetched and are
  reported as unsupported image data.
- The source transform stage now renders alphaModFix, alphaBiLevel,
  alphaCeiling, alphaFloor, alphaInv, alphaRepl, biLevel, clrChange, clrRepl,
  duotone, grayscl, lum, hsl, tint, simple blur, fillOverlay, and
  scalar-container alphaMod effects before sampling.
- Default tiled blip fills render from the decoded/cropped/effected source with
  scale, offset, alignment, and x/y/xy alternate flipping. Tiled fills combined
  with custom masks or soft edges remain partial and report explicitly.
- Unsupported visible blip effects are not silent: non-scalar alphaMod
  containers emit per-object partial diagnostics.
- Object-debug image summaries now expose linked relationship ids, fill mode,
  supported effect metadata, and unsupported blip effect records.

Accepted residuals:

- WHO slide 015 `Picture 4`: still 1,200 differing pixels. The source-backed
  fixture evidence shows a full-source 200x200 opaque PNG, no crop, no mask,
  no soft edge, no line/shadow, and the documented current sampling/output
  pipeline. The remaining residual is picture contour antialiasing/sampling.
- EPA slide 004 `Google Shape;11;p15`: still 2,127 differing pixels under the
  current backend. Relationship, crop, transform, and visible target are
  reproduced; previous broad source/kernal/phase/fractional searches remain
  rejected because they did not pass the picture family.
- The sampling-stage gate now permits a true zero-diff replacement directly and
  permits these current residuals only with explicit source-backed reasons. It
  is no longer a bare residual-lock test.

Remaining true partials:

- Full SVG rendering, video/audio playback, and non-static media behavior stay
  out of scope for M07.
- The exact PowerPoint/Apple Notes picture contour sampling model remains open;
  no broad tuning change was accepted.
- Complex unimplemented visible blip effects are reported instead of marked
  supported: non-scalar alphaMod containers.
- Tiled blip fills under custom geometry masks or soft edges are rendered as
  stretched fallback with a partial report.

Changed code/docs:

- `internal/render/render_parse.go`
- `internal/render/render_pictures.go`
- `internal/render/render_scene.go`
- `internal/render/render_types.go`
- `internal/render/render_object_debug.go`
- `internal/render/render_picture_stage_acceptance_test.go`
- `internal/render/render_m07_test.go`
- `tools/generate_ooxml_drawingml_audit.py`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/renderer-coverage-summary.json`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`
- `docs/RENDERER_EXPERIMENT_LOG.md`

Validation:

```text
python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -run 'TestRenderPicture|TestPicture|Test.*Blip|Test.*Image|Test.*Sampling' -count=1

PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE=1 go test ./internal/render -run TestCurrentPictureSamplingStageAcceptanceGate -count=1

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

go test ./internal/render -count=1

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m07-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

git diff --check
```

Results:

```text
coverage summary: passed; queue totals core-static=16 common-partial=118 hard-rendering=367 unsupported-preserve=432 out-of-scope=74
focused M07 picture/blip/image/sampling tests: passed
picture sampling gate: passed with accepted residuals Picture 4=1200 and Google Shape;11;p15=2127
focused Picture 4 micro-fixture: expected documented failure, 1,200 crop differing pixels
full internal/render suite: passed
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950961349 mean_channel_rms_similarity=0.829162565 total_diff=9321380
git diff --check: passed
```

Decision: M07 is accepted for source-backed picture/media semantics. The
production renderer now implements feasible blip relationship, fill, crop, tile,
and source-effect behavior and only marks the remaining visible image effects as
partial where no current source-backed static implementation has been adopted.

## 2026-06-01 M08 Text Shaping, Layout, And Fonts

Schema anchors:

```text
dml-main.xsd:2540 CT_TextParagraph
dml-main.xsd:2543 EG_TextRun
dml-main.xsd:2592 CT_TextListStyle
dml-main.xsd:2625 CT_TextBodyProperties
dml-main.xsd:2653 CT_TextBody
dml-main.xsd:2873 CT_TextCharacterProperties
dml-main.xsd:2994 CT_TextParagraphProperties
dml-main.xsd:3035 CT_RegularTextRun
```

Implemented source-backed behavior:

- Added a production `textShapingBackend` with a HarfBuzz/go-text
  implementation for supported horizontal LTR runs. Text wrapping, segmented
  line measurement, alignment, highlight widths, underline/strike widths, and
  following segment positions now use shaped advances when the backend accepts
  the run.
- Cached resolved shaping font sources and parsed go-text faces so production
  layout does not reparse fonts for every measurement.
- Extended text primitives with `FontResolution` and static text
  `Unsupported` reports. The primitive now carries paragraph source data plus
  font fallback and text-layout partial diagnostics independent of whether the
  source object was a shape or table cell.
- Added explicit bidi/RTL reporting with schema anchors. RTL is not silently
  labeled supported; it falls back to the existing LTR renderer and emits a
  partial diagnostic.
- Regenerated the schema matrix so text body, paragraph/run, list style,
  bullet, spacing, autofit, and font scheme rows say M08 partial only where
  current code/tests prove parsing, layout, fallback, or reporting.

Accepted residuals:

- WHO slide 012 `Rectangle 5`: still 7,423 visible-crop differing pixels. This
  remains the previously documented text/stroke residual; M08 did not accept a
  y-shift or font tuning workaround.
- WHO slide 015 `TextBox 7`: crop residual is now 19,939 differing pixels,
  improved from the previously recorded 132,995-pixel text-box residual after
  shaped advances entered production layout. The remaining residual is accepted
  only as source-backed M08 partial evidence because glyph drawing, exact Office
  font metrics, and paragraph vertical parity are still incomplete.

Remaining true partials:

- Vertical text modes, text body rotation, multi-column text, bidi/RTL
  reordering, WordArt/text warp, complex-script shaping, and exact Office text
  metrics remain partial/reported.
- The renderer still draws glyph outlines through the existing `font.Drawer`;
  M08 changed layout advances, not glyph rasterization.
- Real-world perceptual metrics are slightly worse than M07, but the accepted
  change is source-backed text semantics rather than broad image tuning.

Validation:

```text
go test ./internal/render -run 'TestM08|TestMeasureStyledSegmentsIncludesCharacterSpacing|TestRenderShapeReportsSpecificUnsupportedTextLayoutFeatures' -count=1

go test ./internal/render -run 'Test.*Text|Test.*Font|Test.*Bullet|Test.*Autofit|Test.*Paragraph' -count=1

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m08-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

git diff --check
```

Results:

```text
focused M08 synthetic tests: passed
focused text/font/bullet/autofit/paragraph tests: passed
Rectangle 5 micro-fixture: expected documented failure, 7,423 visible-crop differing pixels
TextBox 7 micro-fixture: expected documented failure, 19,939 crop differing pixels
coverage summary: passed; queue totals core-static=16 common-partial=139 hard-rendering=346 unsupported-preserve=432 out-of-scope=74
full internal/render suite: passed
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950452042 mean_channel_rms_similarity=0.827985604 total_diff=9337907
git diff --check: passed
```

Decision: M08 is accepted for source-backed horizontal LTR text layout
semantics. Unsupported reporting is limited to concrete unimplemented text
modes rather than being used as a way around feasible implementation.

## 2026-06-01 M09 Tables

Schema anchors:

```text
pml.xsd:1263 CT_GraphicalObjectFrame
dml-main.xsd:842 CT_GraphicalObjectData
dml-main.xsd:2347 CT_TableCellProperties
dml-main.xsd:2381 CT_TableGrid
dml-main.xsd:2386 CT_TableCell
dml-main.xsd:2398 CT_TableRow
dml-main.xsd:2405 CT_TableProperties
dml-main.xsd:2423 CT_Table
dml-main.xsd:2430 tbl
dml-main.xsd:2480 CT_TableCellBorderStyle
dml-main.xsd:2512 CT_TableStyle
```

Implemented source-backed behavior:

- Added DrawingML table diagonal border support for direct cell properties:
  `lnTlToBr` and `lnBlToTr`.
- Added table-style diagonal border support for `tcBdr/tl2br` and
  `tcBdr/tr2bl`.
- Diagonal borders preserve line width, solid color, preset/custom dash,
  supported caps, and single/double compound rendering. Double compound
  diagonals use offsets perpendicular to the diagonal stroke rather than the
  horizontal/vertical table-border offsets.
- Unsupported table line decorations on diagonal borders are now detected by
  the same table unsupported-reporting path as normal cell edges.
- Real-world table fixture manifest generation now selects table graphic frames
  and records table schema anchors plus expected table semantic/primitive
  descriptions.

Generated table fixture:

```text
testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-008/micro-fixtures/table-0005-146-Google-Shape-146-p6/manifest.json
```

Accepted residual:

- EPA slide 008 object 146 `Google Shape;146;p6` remains a large table residual:
  222,465 crop differing pixels. The fixture is now table-specific and records
  the source graphic frame/table schema anchors, source XML path, expected table
  semantic model, and expected table primitive. This residual is not a pass; it
  is the focused M09 real-world table open item for later table layout/text
  parity.

Remaining true partials:

- Full Office table layout/text parity remains incomplete, especially row/cell
  text metrics, table style precedence details, and large real-world table
  layout residuals.
- Cell 3-D and advanced table effects remain partial/reported.
- Image/group cell fills, advanced cell effects, and unknown line decorations
  remain reported where they are not rendered. Direct solid, gradient, and
  pattern cell fills now share the table paint renderer.
- The broad real-world golden artifact pass was stopped after the focused table
  fixture was generated; full corpus golden parity is still a later milestone,
  not M09 completion evidence.

Validation:

```text
go test ./internal/render -run 'TestM09|Test.*Table|TestRenderGraphicFrame' -count=1

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 PUPPT_REALWORLD_ARTIFACT_DIR=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-008/micro-fixtures/table-0005-146-Google-Shape-146-p6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -run 'Test.*Table|TestRenderGraphicFrame' -count=1

go test ./internal/render -count=1

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m09-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

git diff --check
```

Results:

```text
focused M09 table tests: passed
artifact pass: stopped after generating the focused table fixture; full golden mismatch remains known
focused table micro-fixture: expected documented failure, 222,465 crop differing pixels
coverage summary: passed; queue totals core-static=16 common-partial=140 hard-rendering=345 unsupported-preserve=432 out-of-scope=74
focused table/render graphic frame tests: passed
full internal/render suite: passed
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950452042 mean_channel_rms_similarity=0.827985604 total_diff=9337907
git diff --check: passed
```

Decision: M09 accepts source-backed table diagonal border support and
table-specific fixture generation. Unsupported reporting remains explicit and
limited to concrete table gaps rather than replacing feasible implementation.

## 2026-06-01 M10 Effects, Shadows, And Compositing

Schema anchors:

```text
dml-main.xsd:129 CT_EffectProperties
dml-main.xsd:1285 CT_GlowEffect
dml-main.xsd:1309 CT_OuterShadowEffect
dml-main.xsd:1323 ST_PresetShadowVal
dml-main.xsd:1347 CT_PresetShadowEffect
dml-main.xsd:1375 CT_SoftEdgesEffect
dml-main.xsd:1655 CT_EffectContainer
dml-main.xsd:1671 CT_EffectList
dml-main.xsd:1689 CT_EffectProperties
```

Implemented source-backed behavior:

- Shape soft edges now render through an offscreen object layer, DrawingML
  radius alpha-mask blur, and source-over compositing. Unsupported is emitted
  only if the underlying shape layer cannot render.
- Shape and picture glow effects now render from authored radius and color by
  blurring the object alpha mask before the object is painted.
- `prstShdw` now maps authored color, distance, and direction into the existing
  static shadow renderer, with an explicit simplified-preset diagnostic.
- Visible unimplemented effects are no longer silent. Later M12 passes
  implemented source-backed `blur`, `fillOverlay`, `innerShdw`, and
  `reflection` subsets; remaining effectDag and complex effect combinations
  report per object.
- Render effect and picture primitives preserve the new glow and unsupported
  effect metadata for diagnostics/backends.

Focused real-world residual:

- EPA slide 007 master `Freeform 6` underpaint remains a known custom-path
  shadow residual: 2,368 visible-crop differing pixels in
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/manifest.json`.
  This is not accepted as visually correct; it remains source-backed evidence
  that custom-path shadow mask/kernel parity is still incomplete.

Remaining true partials:

- Inner shadow, reflection, object blur, fill overlay, full effectDag ordering,
  3-D effects, and host shadow/glow kernel parity remain partial or reported.
- The existing Freeform shadow diagnostics remain the boundary for future
  custom-path shadow changes; do not replace them with broad pixel tuning.

Validation:

```text
go test ./internal/render -run 'TestM10|TestRenderShapePaintsSoftEdgeEffect|TestRenderShapeReportsSoftEdgeOnlyWhenShapeLayerCannotRender|Test.*Shadow|Test.*Effect|Test.*SoftEdge|Test.*Composite|Test.*3D' -count=1

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m10-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

git diff --check
```

Results:

```text
focused M10 effect tests: passed
focused Freeform shadow micro-fixture: expected documented failure, 2,368 visible-crop differing pixels
coverage summary: passed; queue totals core-static=16 common-partial=144 hard-rendering=341 unsupported-preserve=432 out-of-scope=74
full internal/render suite: passed
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950452042 mean_channel_rms_similarity=0.827985604 total_diff=9337907
git diff --check: passed
```

Decision: M10 accepts the feasible static effect subset implemented from
DrawingML source semantics. Remaining visible effects are explicit partials or
unsupported reports, not silent drops.

## 2026-06-01 M12 Top Picture Failure Refresh

Purpose: refresh current M12 gate evidence and inspect a high-impact,
gate-relevant clean picture failure from source XML before accepting any
picture-pipeline change.

Refreshed gates:

```text
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v

PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-m12-current.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v
```

Results:

```text
clean micro-fixture suite: expected-failure accounting passed; total=70 passed=0 failed=70
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9337907, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported gaps=none
production scoreboard: slides=61 total_slide_diff=9323908 object_groups=8 clean_failures=70
```

Current scoreboard keeps the highest-impact object-overlap queues as shape
geometry/fill/line/clipping/antialiasing, text shaping/font metrics/paragraph
layout/anchoring, and picture crop/resampling/color/media transform. Clean
fixture families are still led by pictures: `Picture 2` has 5 failures and
382,408 differing pixels.

Gate-relevant clean picture target:

```text
manifest: testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json
source object: p:pic cNvPr id=3 name="Picture 2"
blip: r:embed="rId4"
fill mode: a:stretch/a:fillRect
transform: x=0 y=1335505 cx=12192000 cy=5233737
geometry: rect
fixture residual: 154741 differing pixels
```

Diagnostics:

```text
PUPPT_PICTURE_PNG_METADATA_PROFILE_MANIFEST=.../0003-3-Picture-2/manifest.json PUPPT_PICTURE_PNG_METADATA_PROFILE_OUTPUT=.../0003-3-Picture-2/picture-png-metadata.json go test ./internal/render -run TestMicroFixturePicturePNGMetadataProfile -count=1 -v

PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=.../0003-3-Picture-2/manifest.json PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=.../0003-3-Picture-2/picture-pipeline-profile.json go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v

PUPPT_PICTURE_SOURCE_MODEL_SEARCH_MANIFEST=.../0003-3-Picture-2/manifest.json PUPPT_PICTURE_SOURCE_MODEL_SEARCH_OUTPUT=.../0003-3-Picture-2/picture-source-model-search.json go test ./internal/render -run TestMicroFixturePictureSourceModelSearch -count=1 -v

PUPPT_PICTURE_RESIDUAL_PROFILE_MANIFEST=.../0003-3-Picture-2/manifest.json PUPPT_PICTURE_RESIDUAL_PROFILE_OUTPUT=.../0003-3-Picture-2/picture-residual-profile.json go test ./internal/render -run TestMicroFixturePictureResidualProfile -count=1 -v

PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_MANIFEST=.../0003-3-Picture-2/manifest.json PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_OUTPUT=.../0003-3-Picture-2/picture-source-correspondence-profile.json go test ./internal/render -run TestMicroFixturePictureSourceCorrespondenceProfile -count=1 -v
```

Results:

```text
PNG metadata: 2830x820 truecolor-alpha, iCCP=true profile="ICC Profile", pHYs=5669x5669 pixels/meter
pipeline profile: got_delta=0, reference_delta=154741, source=2830x820, unique_colors=1157, opaque_pixels=2320600, alpha_pixels=0, scaler=approx_bilinear
Display P3 output conversion: changed_source_pixels=446036 absolute_delta=2200568
source model search: best=converted_icc/approx_bilinear/floor_floor, different_pixels=154741
residual profile: grayscale_different=112436, edge_coverage_different=112436, pure_black_white=0
source correspondence: source_bounds=x=1..2828 y=0..819, mixed_3x3=107787, nearest_source_antialias=48275
```

Decision: no production picture change accepted. The source object is a simple
full-source stretched PNG and the media metadata is not being silently ignored:
the current ICC-aware source path is already the best source-model candidate.
The remaining `Picture 2` blocker is source/sampling correspondence across a
large ICC-profiled raster, not a missing relationship, crop, mask, or metadata
branch. The next picture implementation step needs a fixture-proven sampling
model that improves this target without regressing the existing Picture
4/Google Shape sampling acceptance gate.

## 2026-06-01 M12 Shape Candidate Rejections

Purpose: test source-backed shape hypotheses from current clean failures without
accepting a renderer change that does not improve the object fixture.

### WHO slide 003 `TextBox 7`: preserve fractional `xfrm` after `spAutoFit`

Source evidence:

```text
source object: p:sp cNvPr id=8 name="TextBox 7"
geometry: a:prstGeom prst="rect"
fill: a:schemeClr val="accent5" with lumMod=20000 lumOff=80000
bodyPr: wrap="square" with a:spAutoFit
```

Candidate: keep the source-derived fractional shape bounds after applying the
`spAutoFit` height adjustment instead of converting the autofit target back to
an integer rectangle before solid-fill rendering.

Verification:

```text
go test ./internal/render -run 'TestShapeAutofitTarget|TestShapeAutofitTargetsPreservesFractionalSourceEdges' -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
synthetic autofit tests: passed
candidate TextBox 7 object fixture: failed, 133021 differing pixels
current rerun after revert: failed, 133022 differing pixels
```

Decision: rejected and reverted. The hypothesis was source-backed, but it did
not pass the object fixture. The TextBox 7 residual remains fill edge, text
placement, and antialiasing evidence; preserving fractional bounds through the
current integer `spAutoFit` target is not an accepted production fix.

### WHO slide 002 `Rectangle 11`: omitted `a:ln/@algn` as centered pen

Source evidence:

```text
source object: p:sp cNvPr id=12 name="Rectangle 11"
geometry: a:prstGeom prst="rect"
line: a:ln w=22225 with a:prstDash val="sysDash"; no algn attribute
bodyPr: wrap="square" with a:spAutoFit
```

Candidate: treat omitted shape line alignment as centered pen alignment in the
shape parser.

Verification:

```text
go test ./internal/render -run 'TestCollectSlideElements.*Line|TestLineDashPatternPixelsUsesDrawingMLPresetRuns' -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
synthetic line parsing tests: passed
candidate Rectangle 11 object fixture: failed, 71272 differing pixels
current rerun after revert: failed, 71272 differing pixels
```

Decision: rejected and reverted. The candidate did not pass the clean fixture,
so no default pen-alignment renderer change was accepted in M12.

### WHO slide 007 `Rectangle 7`: source color versus dominant reference fill

Source evidence:

```text
source object: p:sp cNvPr id=8 name="Rectangle 7"
geometry: a:prstGeom prst="rect"
fill: a:schemeClr val="accent5" with lumMod=20000 lumOff=80000
line: a:ln w=19050, srgbClr val="0070C0", prstDash val="solid"
text: six auto-numbered paragraphs, bodyPr anchor="ctr"
```

Diagnostics:

```text
PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json PUPPT_SHAPE_OBJECT_PROFILE_OUTPUT=.../shape-object-profile-m12.json go test ./internal/render -run TestMicroFixtureShapeObjectProfile -count=1 -v

PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_OUTPUT=.../shape-luminance-color-search-m12.json go test ./internal/render -run TestMicroFixtureShapeLuminanceColorSearch -count=1 -v

PUPPT_SHAPE_FILL_HEIGHT_SEARCH_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json PUPPT_SHAPE_FILL_HEIGHT_SEARCH_OUTPUT=.../shape-fill-height-search-m12.json go test ./internal/render -run TestMicroFixtureShapeFillHeightSearch -count=1 -v

PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=.../shape-text-stroke-profile-m12.json go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v
```

Results:

```text
current object fixture: failed, 56812 differing pixels
dominant got fill: #E0EBF6/FF
dominant reference fill: #E1EBF5/FF
source color search best: current-hsl, internal #DEEBF7/FF, output #E0EBF6/FF, distance_to_reference=1
dominant-fill replacement search: #E1EBF5/FF reduces the fixture to 17197 differing pixels
text/stroke profile: got_text x=8..282 y=10..185, ref_text x=9..284 y=9..184, best text-mask shift only reduces to 56688 pixels
```

Decision: no production color change accepted. Replacing the parsed source fill
with the dominant reference color would reduce this fixture, but the
source-backed luminance candidates do not produce `#E1EBF5/FF` as an exact
DrawingML result. The remaining Rectangle 7 blocker is a combined vector color
rounding, edge antialiasing, and text rasterization gap; a future fix must be
derived from the color/geometry/text model rather than from the reference crop
bucket.

### EPA Residential Wood slide 005 `Google Shape;108;p4`: visible occlusion source bounds

Purpose: audit a high-impact picture fixture before treating its visible label
residual as a picture renderer bug.

Source evidence:

```text
target object: p:pic cNvPr id=108 name="Google Shape;108;p4"
target schema anchors: pml.xsd:1245 CT_Picture, dml-picture.xsd:14 CT_Picture, dml-main.xsd:1502 CT_BlipFillProperties, dml-main.xsd:2223 CT_ShapeProperties
source image: ppt/media/image7.jpg, jpeg, 1632x1056
picture crop: l=0 t=0 r=0 b=39865
later occluding labels: p:sp objects 116, 114, 113, 112, 115, 111, 110, 109, and 117 in ppt/slides/slide5.xml
occluder schema anchors: pml.xsd:1209 CT_Shape, dml-main.xsd:2223 CT_ShapeProperties, dml-main.xsd:2653 CT_TextBody
```

Finding: the reference labels over this picture are not part of the source
JPEG. They are later z-order text-box shapes. The micro-fixture manifest already
recorded later-object occlusion, but those mask bounds came from current
renderer output/ink bounds. For text labels, current renderer ink can be
narrower than the source-authored object rectangle, leaving reference label
pixels inside the visible picture comparison.

Accepted harness change:

```text
microFixtureOcclusions now uses a later object's source-authored pixel_bounds
when present, falling back to output_pixel_bounds only when source bounds are
unavailable.
```

Validation:

```text
go test ./internal/render -run 'TestMicroFixtureOcclusions|TestRenderObjectDebug|TestCleanMicroFixtureOwnershipFailureExcludesUnderpaintConfoundedEdges|TestRenderMicroFixtureWithObjectDebugWritesFixtureRecords' -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 PUPPT_REALWORLD_ARTIFACT_DIR=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-005/micro-fixtures/0005-108-Google-Shape-108-p4/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/micro-fixture-ownership-summary.json go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-m12-current.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v
```

Results:

```text
focused occlusion/object-debug tests: passed
exact Apple Notes gate after artifact refresh: failed; 61/61 slides differ, total_diff=9337907, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported gaps=none
Google Shape;108;p4 targeted fixture: failed at 86813 visible differing pixels, bounds x=0..950 y=28..359
ownership summary: total=179 scoped=179 clean_failures=59 contaminated_failures=74 partial_underpaint_failures=10
clean micro-fixture suite: expected-failure accounting passed; total=59 passed=0 failed=59
production scoreboard: slides=61 total_slide_diff=9337907 object_groups=8 clean_failures=59
```

Decision: accepted as an M12 fixture attribution correction, not as renderer
visual completion. It removes stale label leakage from visible fixture crops and
keeps the residual on the actual picture comparison. The coverage matrix is
unchanged because no schema support status changed. The exact Apple Notes gate
and the clean fixture suite still fail, so M12 remains incomplete.

### WHO slide 012 `Table 3`: package table-style dependency in micro-fixtures

Purpose: audit the top clean table fixture from source XML before treating its
unstyled isolated fixture output as a table renderer failure.

Source evidence:

```text
target object: p:graphicFrame cNvPr id=2 name="Table 3"
schema anchors: pml.xsd:1263 CT_GraphicalObjectFrame, dml-main.xsd:842 CT_GraphicalObjectData, dml-main.xsd:2423 CT_Table, dml-main.xsd:2405 CT_TableProperties, dml-main.xsd:2533 CT_TableStyleList
table flags: firstRow=1 bandRow=1
table style id: {5C22544A-7EE6-4342-B048-85BDC9FD1C3A}
source style part: ppt/tableStyles.xml
style name: Medium Style 2 - Accent 1
style semantics: first-row accent fill/text, banded row fills, and white table borders
```

Finding: the full-deck object artifact already rendered the table style, but
the isolated `fixture.pptx` omitted `ppt/tableStyles.xml`. The fixture therefore
rendered with the default unstyled grid and over-reported the table renderer
gap. The source `a:tblPr/a:tableStyleId` is not self-contained; it resolves
through the package-level DrawingML table-style list.

Accepted harness change:

```text
writeShapeObjectsFixture now copies ppt/tableStyles.xml when the extracted
object set includes a table graphic frame, and the fixture content types include
the table-style part override.
```

Validation:

```text
go test ./internal/render -run 'TestShapeObjectFixtureCopiesTableStylesForTableGraphicFrames|TestShapeObjectFixtureBackgroundXMLPrefersActualSlideBackground|TestRenderGraphicFramePaintsParsedTableStyle|TestResolvedTableCellStyleAppliesGenericRegionPrecedence' -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 PUPPT_REALWORLD_ARTIFACT_DIR=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/micro-fixture-ownership-summary.json go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-m12-current.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v
```

Results:

```text
synthetic table-style dependency/render tests: passed
exact Apple Notes gate after artifact refresh: failed; 61/61 slides differ, total_diff=9337907, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported gaps=none
fixture package check: ppt/tableStyles.xml present, size=3803
Table 3 targeted fixture: failed at 284470 pixels, down from 300067 before tableStyles.xml was copied
ownership summary: total=179 scoped=179 clean_failures=59 contaminated_failures=74 partial_underpaint_failures=10
clean micro-fixture suite: expected-failure accounting passed; total=59 passed=0 failed=59
production scoreboard: slides=61 total_slide_diff=9337907 object_groups=8 clean_failures=59
graphicFrame/table clean failures: 5 fixtures, 582424 differing pixels
```

Follow-up color profile:

```text
PUPPT_TABLE_STYLE_COLOR_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json PUPPT_TABLE_STYLE_COLOR_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/table-style-color-profile-m12.json go test ./internal/render -run TestMicroFixtureTableStyleColorProfile -count=1 -v
```

Profile result:

```text
table style id: {5C22544A-7EE6-4342-B048-85BDC9FD1C3A}
style name: Medium Style 2 - Accent 1
first-row source fill: #4472C4/FF; Display P3 sample: #4F71BE/FF; top got/reference: #4F71BE/FF
band1-row source fill: #CFD5EA/FF; Display P3 sample: #D0D5E8/FF; top got/reference: #D0D5E8/FF vs #CFD4E8/FF
band2-row source fill: #E9EBF5/FF; Display P3 sample: #E9EBF4/FF; top got/reference: #E9EBF4/FF vs #E8EBF4/FF
target diff: 284470 pixels
reference RGB delta sum: -1043634
reference RGB absolute delta sum: 9736410
```

Decision: accepted as an M12 fixture package-dependency correction, not as table
renderer completion. The isolated fixture now preserves the source table style
dependency and exercises the actual styled table renderer. The remaining
`Table 3` residual is table color-management/text/border parity under
`CT_TableStyleList`/`CT_TableStyle`/`CT_TableCellProperties`, so M12 still
requires source-backed renderer work before completion. No renderer color
override was accepted from the reference bucket alone: the source-resolved
table-style fills already match the current Display P3 output for the sampled
header and differ by only one channel on sampled band fills, so the next
checkpoint must prove the color-management/tint rule from source behavior.

### WHO slide 009 `Picture 2`: area-resampling rejection

Purpose: test one source-backed picture sampling candidate for the current
largest non-table clean object fixture without changing production rendering.

Source boundary:

```text
schema anchors: pml.xsd:1245 CT_Picture, dml-picture.xsd:14 CT_Picture, dml-main.xsd:1502 CT_BlipFillProperties, dml-main.xsd:2223 CT_ShapeProperties
source object: p:pic cNvPr id=3 name="Picture 2"
blip: r:embed="rId4"
fill mode: a:stretch/a:fillRect
transform: x=0 y=1335505 cx=12192000 cy=5233737
source media: 2830x820 ICC-profiled PNG
current fixture residual: 154741 differing pixels
```

Validation:

```text
PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE=1 go test ./internal/render -run TestCurrentPictureSamplingStageAcceptanceGate -count=1 -v

PUPPT_PICTURE_AREA_SEARCH_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json PUPPT_PICTURE_AREA_SEARCH_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/picture-area-search-m12.json go test ./internal/render -run TestMicroFixturePictureAreaSearch -count=1 -v
```

Results:

```text
picture stage acceptance gate: passed in current residual-lock mode for WHO slide 015 Picture 4 and EPA slide 004 Google Shape;11;p15
best area candidate: converted_icc/area_srgb_byte/floor_floor
current Picture 2 residual: 154741 differing pixels
area candidate residual: 155545 differing pixels
area candidate total absolute channel delta: 14963497, improved from the current 22065889 but still not an object-fixture pass
```

Decision: rejected. Area resampling is a plausible source-backed model for
minifying a full-source stretched raster, but this candidate worsens the exact
`Picture 2` object fixture and still leaves a large residual. Exact pixel
counts are diagnostic for real-world gates, but a production picture sampling
replacement still must pass the attributed object fixture and preserve the
same-family picture acceptance gate. No production picture change was accepted.

## 2026-06-01 M11 Diagrams, Charts, And Embedded Content

Schema anchors:

```text
pml.xsd:827 CT_OleObjectEmbed
pml.xsd:834 CT_OleObjectLink
pml.xsd:840 CT_OleObject
pml.xsd:851 oleObj
pml.xsd:852 CT_Control
pml.xsd:859 CT_ControlList
pml.xsd:1263 CT_GraphicalObjectFrame
pml.xsd:1297 CT_Rel
dml-main.xsd:842 CT_GraphicalObjectData
dml-diagram.xsd:147 CT_DataModel
dml-diagram.xsd:387 CT_RelIds
dml-diagram.xsd:393 relIds
dml-chart.xsd:26 CT_RelId
dml-chart.xsd:1437 chart
dml-chartDrawing.xsd:139 CT_Drawing
```

Implemented source-backed behavior:

- Graphic-frame payloads now preserve source `graphicData` URI and relationship
  metadata in `slideElement` and render primitives.
- Chart graphic frames are detected from `c:chart/@r:id`, preserved, and
  reported with relationship target details instead of falling through to a
  generic unrendered graphic-frame message.
- OLE objects, controls, content parts, audio, and video payloads are detected
  from source XML and relationships and reported with family-specific messages.
- OLE/control preview pictures remain renderable because nested `p:pic`
  fallback content is still collected and sent through the normal picture path.
- Diagram policy is explicit: only related diagram drawing fallbacks that lower
  into shape/text primitives render; missing drawing fallback and non-shape
  diagram content are reported.
- Package writing preservation is covered for unsupported chart, OLE, ActiveX,
  media, and relationship parts.

Decisions:

- No chart engine was added in M11. Charts are preserve/report until a future
  chart-renderer milestone changes scope.
- No SmartArt layout engine was added. M11 keeps the current drawing-fallback
  path and reports unavailable fallback drawings.
- No OLE/control/media execution or playback was added. Embedded application
  content, ActiveX controls, arbitrary content parts, audio, and video are
  preserved and reported.

Validation:

```text
go test ./internal/render -run 'TestM11|Test.*Diagram|Test.*GraphicFrame|Test.*Chart|Test.*Unsupported' -count=1

go test ./internal/... -run 'Test.*Preserve|Test.*Unsupported|Test.*Validate' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

git diff --check
```

Results:

```text
focused M11 render/report tests: passed
internal preserve/unsupported/validate selector: passed
coverage summary: passed; queue totals core-static=16 common-partial=149 hard-rendering=337 unsupported-preserve=431 out-of-scope=74
git diff --check: passed
```

Decision: M11 completes the non-basic graphic payload boundary. Unsupported is
not used as a shortcut for feasible static rendering; embedded apps, controls,
arbitrary content parts, and media playback use explicit preserve/report
boundaries, while later M12 accounting keeps chart and SmartArt static
rendering as Partial implementation work.

## 2026-06-01 M12 Final Conformance And Release Audit

Purpose: audit the completed supported-scope renderer evidence packet without
using unsupported for feasible static rendering work.

Coverage and supported-row reconciliation:

- `python3 tools/generate_ooxml_drawingml_audit.py` passed.
- Current generated totals: core-static=16, common-partial=362,
  hard-rendering=87, unsupported-preserve=402, out-of-scope=140.
- The coverage matrix now contains 0 `Unimplemented / no evidence` rows. M12
  added source-backed reporting for `embeddedFontLst` package declarations and
  Partial/report evidence for `a:cell3D` table-cell properties instead of
  promoting feasible static gaps to Unsupported.
- The only Supported rows are core package, presentation, slide-order,
  slide-size, and low-level geometry/unit declarations. Renderer object
  families remain Partial, preserve/report, or out of scope instead of being
  over-promoted.

Validation:

```text
go test ./...

git diff --check

go test ./internal/cli -run 'TestRenderJSON|TestRenderJSONHonorsDPIFlag' -count=1 -v

go run ./cmd/puppt render testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/fixture.pptx --slide 1 --out /tmp/puppt-m12-supportedish.png --json

go run ./cmd/puppt render testdata/realworld-ppts/EPA-generate-2021-presentation.pptx --slide 1 --out /tmp/puppt-m12-realworld-slide001.png --json

go test ./internal/render -run TestRendererImplementationHasNoTargetDeckHardcodesOrExternalRendererCalls -count=1 -v

go list -deps ./cmd/puppt | rg -i 'libreoffice|powerpoint|keynote|soffice|chrom(e|ium)|playwright|puppeteer|selenium|unoconv|cloudconvert|magick|slides'

PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m12-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-m12-current.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v
```

Results:

```text
coverage regeneration: passed
focused embedded-font/table-cell-3D/timing renderer tests: passed
go test ./...: passed after M12 documentation closeout edits
git diff --check: passed after M12 documentation closeout edits
CLI JSON tests: passed
direct render JSON checks: passed with stable puppt.v1 render envelope
production dependency audit: passed; no office/browser/SaaS/image-conversion renderer dependency hits
real-world perceptual metrics: slides=61 different=61 mean_luma_similarity=0.950452042 mean_channel_rms_similarity=0.827985604 total_diff=9337907
real-world exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9337907, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 differing pixels, top unsupported rendering gaps=none
clean micro-fixture suite: expected-failure accounting mode passed; total=70 passed=0 failed=70
production scoreboard: slides=61 total_slide_diff=9323908 object_groups=8 clean_failures=70
```

Blocker detail:

```text
coverage: 0 rows remain Unimplemented / no evidence
fixture ownership: total_manifests=170 target_scoped=170 clean=70 contaminated=73 partial_underpaint=9
clean fixture groups: pic failures=46 diff=1499584; sp failures=24 diff=550448
highest-impact object-overlap queues:
- sp shape geometry, fill, line, clipping, or antialiasing: objects=176 overlap_diff=7564185
- sp text shaping, font metrics, paragraph layout, or text anchoring: objects=288 overlap_diff=3465463
- pic picture crop, resampling, color management, or media transform: objects=168 overlap_diff=2356361
- cxnSp shape geometry, fill, line, clipping, or antialiasing: objects=34 overlap_diff=1250121
- graphicFrame table layout or inherited table text styling: objects=9 overlap_diff=814596
gate-relevant clean fixture failures:
- WHO-HIV-testing-algorithms-toolkit.pptx slide 009 pic Picture 2 diff=154741
- WHO-HIV-testing-algorithms-toolkit.pptx slide 003 sp TextBox 7 diff=132995
- EPA-generate-2021-presentation.pptx slide 007 pic Picture 2 diff=95960
- WHO-HIV-testing-algorithms-toolkit.pptx slide 002 pic Picture 7 diff=92497
- EPA-generate-2021-presentation.pptx slide 012 pic Picture 19 diff=91082
```

Decision: M12 is not complete. The renderer has an explicit, source-backed
support boundary and passes the non-visual release-audit checks, but final
conformance remains blocked by exact real-world pixel parity and the 70 tracked
clean micro-fixture failures. The next checkpoint is fixture-family reduction
of those object failures from source XML and schema anchors, then rerunning the
exact Apple Notes gate. No content should be marked unsupported merely to close
these failures.

## 2026-06-01 - M12 EPA Table Row Text Minimum Reflow

Source object:

- Deck: `testdata/realworld-ppts/EPA-residential-wood-MacCarty.pptx`
- Slide: 13
- Object: `graphicFrame` id `179`, name `Google Shape;179;p9`
- Schema anchors: `pml.xsd:1263 CT_GraphicalObjectFrame`,
  `dml-main.xsd:2423 CT_Table`, `dml-main.xsd:2398 CT_TableRow`,
  `dml-main.xsd:2386 CT_TableCell`

Implementation:

- Table rendering now measures rendered text for each source table cell and
  treats that as a row minimum.
- Rows that need more space grow inside the authored graphic-frame table bounds;
  rows with spare capacity shrink proportionally.
- The row reflow is additive and deterministic. It does not mark table text or
  ordinary table rows unsupported.

Validation:

```text
go test ./internal/render -run 'TestAdjustTableRowOffsetsForMinimumHeights|TestTableRowOffsets|TestRenderGraphicFrame' -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-013/micro-fixtures/table-0005-179-Google-Shape-179-p9/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
focused table tests: passed
EPA slide 013 Google Shape;179;p9 fixture: failed at 127315 differing pixels
```

Decision: keep the row text-minimum implementation as source-backed table
layout coverage, but do not claim the EPA table object is fixed. The unchanged
fixture diff shows additional table text wrapping/font metrics/border parity
work remains inside the supported M12 rendering gap.

## 2026-06-01 - M12 Authored Hyphen Text Wrap Point

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 3
- Object: `sp` id `8`, name `TextBox 7`
- Schema anchors: `pml.xsd:1209 CT_Shape`,
  `dml-main.xsd:2540 CT_TextParagraph`,
  `dml-main.xsd:3035 CT_RegularTextRun`

Source semantics:

- The second bullet contains the run text `treatment-adjusted prevalence`.
- The reference wraps this at the authored hyphen:
  `treatment-` then `adjusted prevalence`.
- The previous renderer tokenized only whitespace, so it treated
  `treatment-adjusted` as one unbreakable word and wrapped before the bold run.

Implementation:

- Plain and styled text tokenization now exposes a wrap point after authored
  ASCII hyphen and Unicode hyphen while preserving the hyphen on the previous
  line.
- This does not change font metrics, measured advances, colors, or line widths.

Validation:

```text
go test ./internal/render -run 'TestWrapTextWithPrefixes|TestStyledWordTokens|TestTextRenderLinesForElementPreservesAuthoredSpacesWhenStyledTextWraps' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-hyphen PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-013/micro-fixtures/shape-0005-4-TextBox-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
synthetic hyphen/text token tests: passed
slide 003 TextBox 7: still failed, 130250 visible-crop differing pixels
slide 015 TextBox 7: still failed, 19939 crop differing pixels
slide 013 TextBox 3: still failed, 25347 visible-crop differing pixels
```

Decision: accept the source-backed hyphen wrap implementation, but do not claim
the slide 003 `TextBox 7` object is fixed. The fixture pixel count worsened
slightly from the prior 130,103 diagnostic count even though the hyphenated line
pattern now matches the reference; remaining work is still text antialiasing,
fill/color, and line placement parity inside the supported text renderer.

## 2026-06-01 - M12 Generated Bullet-Prefix Spacer Font Candidate

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 3
- Object: `sp` id `8`, name `TextBox 7`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json`
- Schema anchors: `pml.xsd:1209 CT_Shape`,
  `dml-main.xsd:2653 CT_TextBody`,
  `dml-main.xsd:2540 CT_TextParagraph`,
  `dml-main.xsd:2994 CT_TextParagraphProperties`, and
  `dml-main.xsd:3035 CT_RegularTextRun`.

Source semantics:

- The bullet paragraphs use `a:buFont typeface="Arial"`.
- Text runs use `a:rPr/a:latin typeface="Arial"`.
- The renderer-generated prefix spacer between the bullet and first text run
  had been falling back to paragraph/default font metrics.

Candidate:

- Make the generated bullet-prefix spacer inherit the first text segment's
  font family, size, style, character spacing, and kerning metadata.
- Seed wrapped first lines with the first token when constructing the prefix so
  the spacer can see those first-token metrics.

Validation:

```text
go test ./internal/render -run 'TestBulletPrefixSpacerInheritsFirstTextSegmentMetrics|TestWrappedBulletPrefixSpacerInheritsFirstTokenMetrics|TestTextParagraphsFromNodeDetectsBulletsAndLevels|TestTextParagraphsFromNodeCapturesBulletSizeFollowText' -count=1 -v

PUPPT_SHAPE_TEXT_SHAPING_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json PUPPT_SHAPE_TEXT_SHAPING_PROFILE_OUTPUT=/tmp/puppt-textbox7-shaping-spacer2.json go test ./internal/render -run TestMicroFixtureShapeTextShapingProfile -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-spacer2 PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
focused synthetic tests: passed
text shaping profile: passed; lines=14 segments=64 max_delta=1
profile confirmation: bullet, spacer, and first text segment all resolved Arial
slide 003 TextBox 7 fixture: still failed, moved from 130250 to 130252 visible-crop differing pixels
```

Decision: reject and revert. The source observation is valid, but this local
model worsened the object fixture, so it is not acceptable production renderer
behavior. Bullet separator metrics, tab behavior, hanging geometry, and line
placement remain supported-scope text-renderer work; no Unsupported
classification is used.

## 2026-06-01 - M12 Wingdings Symbol Bullet Mapping

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 2
- Object: `sp` id `12`, name `Rectangle 11`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json`
- Schema anchors: `pml.xsd:1209 CT_Shape`,
  `dml-main.xsd:2741 EG_TextBulletTypeface`,
  `dml-main.xsd:2751 CT_TextCharBullet`,
  `dml-main.xsd:2994 CT_TextParagraphProperties`

Source semantics:

- The nested bullet paragraphs use `a:buFont typeface="Wingdings"` and
  `a:buChar char="§"`.
- The reference crop renders these bullets as solid square bullets.
- The previous local-font path mapped this through a Wingdings private-use code
  when an exact local font was available, which produced visible missing-glyph
  boxes in this renderer path.

Implementation:

- Known Office symbol-font bullet encodings now normalize to deterministic
  Unicode static equivalents before any local Wingdings private-use path.
- Wingdings `§` maps to Unicode `▪`; Wingdings `Ø` keeps the existing Unicode
  `¬` fallback.
- Unicode-mapped symbol bullets use the paragraph/generic font selection for
  rendering so they do not depend on private-use Wingdings glyph availability.

Validation:

```text
go test ./internal/render -run 'TestTextParagraphsFromNodeDetectsBulletsAndLevels|TestTextParagraphsFromNodeMapsWingdingsNotSignBullet|TestTextRenderLinesForElementAppliesBulletFontFamily|TestTextRenderLinesForElementUsesParagraphFontForBulletFontTx|TestRenderShape.*Symbol|TestTextRenderLinesForElement.*Bullet' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect11-wingdings PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v
```

Result:

```text
focused bullet parsing/layout tests: passed
slide 002 Rectangle 11: still failed, 71260 visible-crop differing pixels
```

Decision: accept the source-backed symbol bullet mapping. The debug crop now
uses square bullets instead of missing-glyph boxes, and the object diff improves
from the tracked 71,272 to 71,260 pixels. Do not claim the object is fixed:
remaining supported-scope work includes bullet indentation, text placement,
shape fill/stroke, and antialiasing parity.

## 2026-06-01 - M12 Non-Placeholder OtherStyle Paragraph Defaults

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 2
- Object: `sp` id `12`, name `Rectangle 11`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json`
- Schema anchors: `pml.xsd:1412 CT_SlideMasterTextStyles`,
  `dml-main.xsd:2592 CT_TextListStyle`,
  `dml-main.xsd:2994 CT_TextParagraphProperties`

Source semantics:

- `Rectangle 11` is a non-placeholder text shape.
- Its first paragraph has local `marL="285750"` and `indent="-285750"`.
- Its level-1 paragraphs have local `lvl="1"` plus bullet color/font/char but
  omit `marL`, `indent`, and `defTabSz`.
- The fixture master provides `p:txStyles/p:otherStyle/a:lvl2pPr` with
  `marL="457200"` and `defTabSz="914400"`.
- The previous renderer parsed `otherStyle` as `default` but applied it only to
  table cell text, so non-placeholder shape bullets fell back to literal
  space-prefix indentation.

Implementation:

- Non-placeholder shape text now receives inherited `otherStyle` paragraph
  defaults through the existing paragraph-style merge path.
- Local paragraph geometry still wins over inherited defaults.
- No fixture-specific pixel offsets, font overrides, color overrides, or
  screenshot tuning were added.

Validation:

```text
go test ./internal/render -run 'TestApplyInheritedTextStylesAppliesDefaultParagraphStyleToNonPlaceholderShapes|TestApplyInheritedTextStylesAppliesTitleButSkipsBodyPlaceholders|TestInheritedTextStylesUsePresentationDefaultAsBase|TestApplyInheritedTextStylesAppliesBodyParagraphMargins|TestTextRenderLinesForElementUsesHangingBulletTabStop|TestTextParagraphsFromNodeDetectsBulletsAndLevels' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect11-otherstyle PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-013/micro-fixtures/shape-0005-4-TextBox-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

go test ./internal/render -count=1
```

Result:

```text
focused inheritance/text tests: passed
slide 002 Rectangle 11: still failed, 71231 visible-crop differing pixels
slide 003 TextBox 7: still failed, 130250 visible-crop differing pixels
slide 015 TextBox 7: still failed, 19939 crop differing pixels
slide 013 TextBox 3: still failed, 25347 visible-crop differing pixels
go test ./internal/render -count=1: passed
```

Decision: accept the source-backed `otherStyle` inheritance change. The nested
`Rectangle 11` square bullets now use inherited paragraph offsets and align
horizontally with the reference, improving the object diff from 71,260 to
71,231 pixels. Do not claim the object is fixed; remaining residual is still
inside supported text/shape rendering parity.

## 2026-06-01 - M12 Default-Cap Dashed Stroke Antialiasing

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 2
- Object: `sp` id `12`, name `Rectangle 11`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json`
- Schema anchors: `dml-main.xsd:2160 CT_PresetLineDashProperties`,
  `dml-main.xsd:2172 EG_LineDashProperties`,
  `dml-main.xsd:2206 CT_LineProperties`

Source semantics:

- The shape stroke is `a:ln w="22225"` with no authored `cap` attribute.
- The stroke contains `a:prstDash val="sysDash"`.
- The existing renderer preserved the dash pattern but routed omitted/default
  square-cap dashed lines through the legacy point-plotting path instead of the
  antialiased dash renderer already used for explicit cap modes.

Implementation:

- Default or explicit square-cap dashed strokes now call the existing
  antialiased dashed-line renderer.
- Solid square-cap line rendering is unchanged.
- No fixture-specific stroke color, width, alignment, or dash values were
  introduced.

Validation:

```text
go test ./internal/render -run 'TestRenderShapePaintsDashedRectOutline|TestRenderShapeUsesStrokeWidthForSystemDotRectOutline|TestRenderShapeHonorsExplicitFlatCapForSystemDotRectOutline|TestRenderShapeHonorsFlatLineCapOnDashedLine|TestLineDashPatternPixelsUsesDrawingMLPresetRuns|TestM06RendersCompoundConnectorAndCustomDash' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect11-dashaa PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=/tmp/puppt-rect11-text-stroke-profile.json go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v

go test ./internal/render -count=1

git diff --check
```

Result:

```text
focused dashed-stroke tests: passed
slide 002 Rectangle 11: still failed, 71231 visible-crop differing pixels
shape text/stroke profile: passed; got text mask x=8..581 y=13..108, reference text mask x=6..576 y=6..109, edge residual=2860, text-like residual=8555, best diagnostic text-mask shift y=-6 at 71039 pixels
go test ./internal/render -count=1: passed
git diff --check: passed
```

Decision: accept the default-cap dashed-stroke change as source-backed
`CT_LineProperties` coverage, but do not claim the object is fixed. The
diagnostic text/stroke profile points at text metrics and paint residuals. The
source runs have `a:ea typeface="Arial"` without `a:latin`; using that East
Asian slot to force Latin text into Arial is rejected because it contradicts the
current source model and M08 tests. The next fix must start from stronger
source-backed text metrics or paint semantics rather than a y-shift or font-slot
shortcut.

## 2026-06-01 - M12 Zero-Height Table Row Text Proportions

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 15
- Object: `graphicFrame` id `3`, name `Table 2`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/table-0010-3-Table-2/manifest.json`
- Schema anchors: `pml.xsd:1263 CT_GraphicalObjectFrame`,
  `dml-main.xsd:842 CT_GraphicalObjectData`,
  `dml-main.xsd:2423 CT_Table`, `dml-main.xsd:2386 CT_TableCell`,
  `dml-main.xsd:2398 CT_TableRow`, and
  `dml-main.xsd:2347 CT_TableCellProperties`.

Source semantics:

- The source table has six `a:tr h="0"` rows.
- The first row is a table header (`a:tblPr firstRow="1" firstCol="1" bandRow="1"`).
- Several first-row cells contain multiple source paragraphs, such as `Assay`
  and `1`, with `a:tcPr ... anchor="ctr"`.
- Because all row heights are zero, there is no usable authored row-height
  proportion to preserve; the source text body is the only row-height signal
  inside the fixed graphic-frame extent.

Implementation:

- `tableRowOffsetsWithTextMinimums` now detects the all-zero authored-height
  table case and derives row proportions from measured source table-cell text
  heights instead of keeping an equal-row fallback.
- Non-zero or mixed authored row heights continue through the existing
  source-height/text-minimum reflow path.
- No reference color, pixel offset, font, or table-specific object override was
  added.

Validation:

```text
go test ./internal/render -run 'TestTableRowOffsets|TestAdjustTableRowOffsets|TestTableTextMinimum|TestRenderGraphicFramePaintsParsedTableStyle|TestResolvedTableCellStyleAppliesGenericRegionPrecedence|TestTableRowOffsetsWithZeroAuthoredHeightsGrowMultiParagraphHeader' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table2-zeroheight PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/table-0010-3-Table-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-after-zeroheight PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

go test ./internal/render -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-m12-current.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v
```

Result:

```text
focused table tests: passed
Table 2 fixture: still failed, improved from 63031 to 55832 differing pixels
Table 3 fixture: still failed, unchanged at 284470 differing pixels
go test ./internal/render -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59; graphicFrame/table diff total now 564875
production scoreboard: passed; slides=61 total_slide_diff=9337907 object_groups=8 clean_failures=59
```

Decision: accept the source-backed zero-height table-row layout change as
`CT_TableRow`/`CT_TableCell` progress. `Table 2` now keeps the two-line header
inside the header row and moves body text closer to the reference. Do not claim
table parity or M12 completion: `Table 2`, `Table 3`, the clean fixture suite,
and the exact Apple Notes gate still fail.

## 2026-06-01 - M12 Spanning Table Cell Text Minimum Width

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 8
- Object: `graphicFrame` id `15`, name `Table 15`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/table-0003-15-Table-15/manifest.json`
- Schema anchors: `pml.xsd:1263 CT_GraphicalObjectFrame`,
  `dml-main.xsd:842 CT_GraphicalObjectData`,
  `dml-main.xsd:2423 CT_Table`, `dml-main.xsd:2386 CT_TableCell`,
  `dml-main.xsd:2398 CT_TableRow`, and
  `dml-main.xsd:2347 CT_TableCellProperties`.

Source semantics:

- The first row contains a header `a:tc gridSpan="3"` with two source
  paragraphs: `Number of quality-assured products eligible for` and
  `procurement through WHO and Global Fund`.
- The next two cells in that row are `a:tc hMerge="1"`.
- Row-height reflow must measure the header text against the full spanned
  source cell width, not only the first physical grid column.

Implementation:

- `tableTextMinimumRowHeights` now computes the text-measurement rectangle from
  `CT_TableCell/@gridSpan` before measuring row text minimums.
- Existing paint-time `tableCellRect` already used `gridSpan`; this aligns the
  reflow measurement path with the render primitive.
- No reference-driven row offsets, font changes, color changes, or fixture
  overrides were added.

Validation:

```text
go test ./internal/render -run 'TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth|TestTableRowOffsetsWithZeroAuthoredHeightsGrowMultiParagraphHeader|TestAdjustTableRowOffsetsForMinimumHeights' -count=1 -v

go test ./internal/render -run 'TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth|TestTableRowOffsetsWithZeroAuthoredHeightsGrowMultiParagraphHeader|TestTableRowOffsets|TestAdjustTableRowOffsets|TestRenderGraphicFramePaintsParsedTableStyle|TestResolvedTableCellStyleAppliesGenericRegionPrecedence' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table15-gridspan PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/table-0003-15-Table-15/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table2-gridspan PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/table-0010-3-Table-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-gridspan PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

go test ./internal/render -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v

temporary A/B check with only the over-capacity branch disabled:
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result:

```text
focused table tests: passed
Table 15 fixture: still failed, unchanged at 79708 differing pixels
Table 2 fixture: still failed, unchanged at 55832 differing pixels
Table 3 fixture: still failed, unchanged at 284470 differing pixels
go test ./internal/render -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
```

Decision: accept the `gridSpan` measurement correction as source-backed
`CT_TableCell` coverage. It fixes a semantic mismatch between table paint and
table reflow measurement, but it does not close a tracked table fixture. Table
layout/text/color parity remains open, and M12 remains blocked by the exact
Apple Notes and clean-fixture gates.

## 2026-06-01 - M12 Over-Capacity Spanning Table Row Reflow

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 8
- Object: `graphicFrame` id `15`, name `Table 15`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/table-0003-15-Table-15/manifest.json`
- Schema anchors: `pml.xsd:1263 CT_GraphicalObjectFrame`,
  `dml-main.xsd:842 CT_GraphicalObjectData`,
  `dml-main.xsd:2423 CT_Table`, `dml-main.xsd:2386 CT_TableCell`,
  `dml-main.xsd:2398 CT_TableRow`, and
  `dml-main.xsd:2347 CT_TableCellProperties`.

Source semantics:

- The table has five equal authored `a:tr h="370840"` rows inside a fixed
  `p:graphicFrame` transform.
- The first row contains a centered `a:tc gridSpan="3"` header with two source
  paragraphs and two following `a:tc hMerge="1"` cells.
- The measured source row text minimums exceed the available fixed table frame
  height. The prior allocator treated that as no-op and kept equal row offsets,
  leaving the two-line spanning header compressed.

Implementation:

- `tableRowOffsetsWithTextMinimums` now detects the first-row spanning/header
  over-capacity case and derives row offsets from measured source
  text-minimum proportions while keeping the table inside the authored frame.
- The fallback is gated by `tableFirstRowHasSpanningCells` plus measured
  minimums exceeding the fixed frame. It does not apply to ordinary tables
  whose row minima can be satisfied by the existing capacity-aware reflow.
- Added
  `TestTableRowOffsetsWithTextMinimumsUsesMinimumProportionsWhenFrameIsOverCapacity`.
- No screenshot-derived constants, font/color changes, sampling changes, or
  unsupported/out-of-scope classifications were added.

Validation:

```text
go test ./internal/render -run 'TestTableRowOffsetsWithTextMinimumsUsesMinimumProportionsWhenFrameIsOverCapacity|TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth|TestTableRowOffsetsWithZeroAuthoredHeightsGrowMultiParagraphHeader|TestAdjustTableRowOffsetsForMinimumHeights|TestTableRowOffsets' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table15-after PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/table-0003-15-Table-15/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table2-after PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/table-0010-3-Table-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-after PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

go test ./internal/render -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v
```

Result:

```text
focused table tests: passed
Table 15 fixture: still failed, improved from 79708 to 72605 differing pixels
Table 2 fixture: still failed, unchanged at 55832 differing pixels
Table 3 fixture: still failed, unchanged at 284470 differing pixels
go test ./internal/render -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59; graphicFrame/table diff total now 557772
exact Apple Notes gate with branch enabled: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 differing pixels, top unsupported rendering gaps=none
exact Apple Notes gate with branch disabled: failed; 61/61 slides differ, total_diff=9348120, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 differing pixels, top unsupported rendering gaps=none
```

Decision: accept the over-capacity first-row spanning table reflow as
source-backed `CT_TableRow`/`CT_TableCell` progress. It visibly moves the
two-line `Table 15` header toward the reference, reduces the clean table
bucket, and improves the current exact gate by 7,103 pixels versus an A/B run
with the branch disabled. M12 remains incomplete: `Table 15`, `Table 2`,
`Table 3`, the clean fixture suite, and the exact Apple Notes gate still fail.

## 2026-06-01 - M12 Character-Bullet Hanging Tab Stops

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 2
- Object: `sp` id `12`, name `Rectangle 11`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json`
- Schema anchors: `pml.xsd:1209 CT_Shape`,
  `dml-main.xsd:2751 CT_TextCharBullet`, and
  `dml-main.xsd:2994 CT_TextParagraphProperties`

Source semantics:

- The first paragraph has a character bullet plus local source geometry
  `marL="285750"` and `indent="-285750"`.
- The previous renderer only converted margin/indent hanging geometry into a
  tab stop for auto-number bullets.
- Character bullets therefore used a literal bullet-plus-space prefix instead
  of moving following text to the authored hanging margin.

Candidate:

- `hangingBulletTabStop` was temporarily changed to apply to any paragraph with
  a bullet and source margin/indent geometry.
- The change reuses the existing tab-stop renderer and does not add
  object-specific x offsets, font substitutions, color overrides, or screenshot
  tuning.
- Added `TestHangingBulletTabStopAppliesToCharacterBullets`.

Validation:

```text
go test ./internal/render -run 'TestHangingBulletTabStopAppliesToCharacterBullets|TestTextParagraphsFromNodeDetectsBulletsAndLevels|TestTextParagraphsFromNodeLocalBulletChoiceBlocksStyledAutoNumber' -count=1

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect11-hanging-bullet PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-hanging-bullet PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox4-hanging-bullet PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/shape-0002-5-TextBox-4/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect7-hanging-bullet PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-007/micro-fixtures/shape-0005-8-Rectangle-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1
```

Result:

```text
focused character-bullet tests: passed
slide 002 Rectangle 11: still failed, moved from 71231 to 71244 differing pixels
slide 003 TextBox 7: still failed, moved from 130250 to 130392 differing pixels
slide 008 TextBox 4: still failed, unchanged at 26639 differing pixels
slide 007 Rectangle 7: still failed, unchanged at 56812 differing pixels
coverage summary: passed; queue totals unchanged at core-static=16 common-partial=362 hard-rendering=87 unsupported-preserve=402 out-of-scope=140
go test ./internal/render -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59; shape bucket now 571613 pixels
exact Apple Notes gate with candidate: failed; 61/61 slides differ, total_diff=9356836, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
post-revert coverage summary: passed; queue totals unchanged
post-revert go test ./internal/render -count=1: passed
post-revert clean micro-fixture suite: passed in expected-failure accounting mode; total=59 passed=0 failed=59; shape bucket restored to 571458 pixels
post-revert go test ./...: passed
post-revert git diff --check: passed
post-revert exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: rejected and reverted. The hypothesis was source-backed, but the
simple auto-number tab-stop generalization worsened the targeted fixture,
neighbor `TextBox 7`, clean-suite shape pixels, and the exact real-world gate.
Character-bullet hanging geometry remains supported-scope work; the next
attempt needs a stronger source model for text body insets, tab-stop positions,
literal leading spaces, and shaped text metrics before changing production
layout.

## 2026-06-01 - M12 Non-Spanning First-Row Table Reflow

Source object:

- Deck: `testdata/realworld-ppts/EPA-residential-wood-MacCarty.pptx`
- Slide: 13
- Object: `graphicFrame` id `179`, name `Google Shape;179;p9`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-013/micro-fixtures/table-0005-179-Google-Shape-179-p9/manifest.json`
- Schema anchors: `pml.xsd:1263 CT_GraphicalObjectFrame`,
  `dml-main.xsd:842 CT_GraphicalObjectData`,
  `dml-main.xsd:2423 CT_Table`,
  `dml-main.xsd:2386 CT_TableCell`, and
  `dml-main.xsd:2347 CT_TableCellProperties`

Source semantics:

- The table has `a:tblPr firstRow="1" bandRow="1"`.
- The first row is not a spanning header, but its source cells contain wrapping
  header text such as `Emission Rate PM2.5 (g/hr)` and `Firepower(W)`.
- The existing over-capacity fallback used measured source text-minimum
  proportions only when the first row had spanning cells. This left
  non-spanning first-row tables outside that source-backed row allocator.

Change:

- `tableRowOffsetsWithTextMinimums` now applies the first-row over-capacity
  proportional fallback to any authored `firstRow` when the first row is the
  only row over its measured text minimum and total measured minimums exceed the
  fixed graphic-frame height.
- Added
  `TestTableRowOffsetsWithTextMinimumsReflowsNonSpanningFirstRowWhenFrameIsOverCapacity`.
- No reference-derived row offsets, colors, fonts, or object-specific table
  names were added.

Validation:

```text
go test ./internal/render -run 'TestTableRowOffsetsWithTextMinimums' -count=1

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-google179-candidate PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-013/micro-fixtures/table-0005-179-Google-Shape-179-p9/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-candidate PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table15-candidate PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/table-0003-15-Table-15/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

go test ./internal/render -run 'TestTableRowOffsetsWithTextMinimums|TestTableTextMinimum|TestRenderGraphicFrame|TestMicroFixtureTableStyleColorProfile' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

git diff --check

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

go test ./... -count=1

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result:

```text
focused first-row/table tests: passed
EPA slide 013 Google Shape;179;p9: still failed, unchanged at 127315 differing pixels
WHO slide 012 Table 3: still failed, unchanged at 284470 differing pixels
WHO slide 008 Table 15: still failed, unchanged at 72605 differing pixels
coverage summary: passed; queue totals unchanged at core-static=16 common-partial=362 hard-rendering=87 unsupported-preserve=402 out-of-scope=140
go test ./internal/render -count=1: passed
git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
go test ./... -count=1: passed
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as source-backed `CT_TableRow`/`CT_TableCell` coverage with
synthetic fixture proof, but not as a table fixture completion. The current top
table failures remain supported-scope work. `Unsupported` was not used for any
visible table residual.

## 2026-06-01 - M12 DrawingML Shape/Picture Blur Effect Rendering

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1264`
  `CT_BlurEffect`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1266`
  `a:blur/@grow`, optional with default `true`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` row
  `dml-main.xsd:1264 CT_BlurEffect`

Change:

- `parseShapeEffects` now parses visible `a:blur` into shape effect state
  instead of reporting it as unrendered.
- Shape and picture rendering now paint the supported static object into an
  isolated layer, apply an alpha-weighted RGBA Gaussian blur using the source
  `rad`, and composite either the grown bounds (`grow=true`) or authored bounds
  (`grow=false`) back to the slide.
- The coverage generator now classifies `CT_BlurEffect` as Partial in
  `hard-rendering`: supported static shape/picture blur is implemented. Later
  M12 work added simple blip blur rendering; combined blip blur with
  higher-order object effects remains an explicit partial report.

Validation:

```text
go test ./internal/render -run 'TestM10.*Blur|TestM10.*Effect|TestM10.*Glow|TestM10.*SoftEdge' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

rg -n "<a:blur|:blur" testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 -g 'source-object.xml' -g '*.xml'

go test ./internal/render -count=1

git diff --check

go test ./... -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result:

```text
focused blur/effect tests: passed
coverage summary: passed; queue totals core-static=16 common-partial=362 hard-rendering=88 unsupported-preserve=401 out-of-scope=140
current clean object fixture corpus blur search: no a:blur objects found
go test ./internal/render -count=1: passed
git diff --check: passed
go test ./... -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as source-backed implementation of static shape/picture
blur. This does not close any current clean object fixture because the tracked
corpus has no `a:blur` source objects; remaining exact-gate and clean-fixture
failures stay supported-scope implementation work, not Unsupported.

## 2026-06-01 - M12 DrawingML Fill Overlay Effect Rendering

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1606`
  `CT_FillOverlayEffect`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1597`
  `ST_BlendMode`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` row
  `dml-main.xsd:1606 CT_FillOverlayEffect`

Change:

- `parseShapeEffects` now parses visible `a:fillOverlay` instead of reporting
  it as unrendered.
- Shape and picture primitives now carry the resolved overlay fill and required
  source blend mode.
- Shape and picture rendering now paint the supported static object into an
  isolated layer, apply `over`, `mult`, `screen`, `darken`, or `lighten`
  fill-overlay blending to object pixels, and composite the result back to the
  slide.
- The coverage generator now classifies `CT_FillOverlayEffect` as Partial in
  `hard-rendering`; unsupported no longer covers this statically renderable
  effect.

Validation:

```text
go test ./internal/render -run 'TestM10.*FillOverlay|TestM10.*Blur|TestM10.*Effect|TestM10.*Glow|TestM10.*SoftEdge' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

rg -n 'fillOverlay|alphaOutset|innerShdw|reflection' testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 -g 'source-object.xml' -g '*.xml'

go test ./... -count=1

git diff --check

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result:

```text
focused fillOverlay/effect tests: passed
coverage summary: passed; queue totals core-static=16 common-partial=362 hard-rendering=89 unsupported-preserve=400 out-of-scope=140
go test ./internal/render -count=1: passed
current clean object fixture corpus effect search: no fillOverlay, alphaOutset, innerShdw, or reflection objects found
go test ./... -count=1: passed
git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as source-backed implementation of static shape/picture fill
overlay. This does not close any current clean object fixture because the
tracked corpus has no `a:fillOverlay` source objects; remaining exact-gate and
clean-fixture failures stay supported-scope implementation work, not
Unsupported.

## 2026-06-01 - M12 DrawingML Reflection Effect Rendering

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1355`
  `CT_ReflectionEffect`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` row
  `dml-main.xsd:1355 CT_ReflectionEffect`

Change:

- `parseShapeEffects` now parses visible `a:reflection` instead of reporting it
  as unrendered.
- Shape, picture, theme, scene, and object-debug effect state now carries the
  authored reflection blur, alpha ramp, start/end positions, distance,
  direction, fade direction, scale/skew, alignment, and rotate-with-shape
  values.
- Shape and picture rendering now paint the supported static object into an
  isolated layer, mirror the object below its authored bounds, apply the
  authored alpha ramp and optional blur, and composite the reflection plus
  object back to the slide.
- Non-bottom reflection transform variants are still explicit simplified
  partials; they are not silently dropped.
- The coverage generator now classifies `CT_ReflectionEffect` as Partial in
  `hard-rendering`; unsupported no longer covers this statically renderable
  effect.

Validation:

```text
go test ./internal/render -run 'TestM10.*Reflection|TestM10.*Effect|TestM10.*InnerShadow|TestM10.*FillOverlay|TestM10.*Blur|TestM10.*Glow|TestM10.*SoftEdge' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

go test ./... -count=1

git diff --check

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result:

```text
focused reflection/effect tests: passed
coverage summary: passed; queue totals core-static=16 common-partial=362 hard-rendering=91 unsupported-preserve=398 out-of-scope=140
go test ./internal/render -count=1: passed
current clean object fixture corpus precise effect tag search: no alphaOutset or reflection objects found
go test ./... -count=1: passed
git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as source-backed implementation of the static
shape/picture bottom reflection subset. This does not close any current clean
object fixture because the tracked corpus has no `a:reflection` source objects;
remaining exact-gate and clean-fixture failures stay supported-scope
implementation work, not Unsupported.

## 2026-06-01 - M12 DrawingML Inner Shadow Effect Rendering

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1297`
  `CT_InnerShadowEffect`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` row
  `dml-main.xsd:1297 CT_InnerShadowEffect`

Change:

- `parseShapeEffects` now parses visible `a:innerShdw` instead of reporting it
  as unrendered when the required source color resolves.
- Shape, picture, theme, scene, and object-debug effect state now carries the
  resolved inner-shadow color, blur radius, distance, and direction.
- Shape and picture rendering now paint the supported static object into an
  isolated layer, builds an object alpha mask, offsets and blurs it, applies the
  inward shadow inside the source alpha, and composites the result back to the
  slide.
- The coverage generator now classifies `CT_InnerShadowEffect` as Partial in
  `hard-rendering`; unsupported no longer covers this statically renderable
  effect.

Validation:

```text
go test ./internal/render -run 'TestM10.*InnerShadow|TestM10.*Effect|TestM10.*FillOverlay|TestM10.*Blur|TestM10.*Glow|TestM10.*SoftEdge' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

rg -n '<a:(alphaOutset|innerShdw|reflection)\b' testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 -g 'source-object.xml' -g '*.xml'

go test ./... -count=1

git diff --check

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v
```

Result:

```text
focused innerShadow/effect tests: passed
coverage summary: passed; queue totals core-static=16 common-partial=362 hard-rendering=90 unsupported-preserve=399 out-of-scope=140
go test ./internal/render -count=1: passed
current clean object fixture corpus precise effect tag search: no alphaOutset, innerShdw, or reflection objects found
go test ./... -count=1: passed
git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as source-backed implementation of static shape/picture
inner shadow. This does not close any current clean object fixture because the
tracked corpus has no `a:innerShdw` source objects; remaining exact-gate and
clean-fixture failures stay supported-scope implementation work, not
Unsupported.

## 2026-06-01 - M12 Simple DrawingML EffectDag Flattening

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1615`
  `EG_Effect`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1655`
  `CT_EffectContainer`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` rows
  `dml-main.xsd:1615 EG_Effect` and
  `dml-main.xsd:1655 CT_EffectContainer`

Change:

- `parseShapeProperties` now routes `a:effectDag` through a bounded parser
  instead of treating every effect graph as wholly unrendered.
- Simple `a:cont` trees containing effects already supported by the static
  renderer (`blur`, `fillOverlay`, `glow`, `innerShdw`, `outerShdw`,
  `prstShdw`, `reflection`, and `softEdge`) are flattened into the normal
  effect-list path.
- Ordering-sensitive graph containers and graph-only effect nodes such as
  `blend` remain explicit partial reports; they are not silently dropped or
  marked impossible. Later M12 work implemented `alphaOutset` as a
  source-backed static alpha-mask expansion, `relOff` as source-backed
  object-layer translation, and the `xfrm` `tx`/`ty` subset as source-backed
  coordinate translation instead of leaving them in this reported-only group.
- The coverage generator now classifies `EG_Effect` as Partial instead of
  Unsupported because the supported static subset has synthetic fixture proof.

Validation:

```text
go test ./internal/render -run 'TestM10.*EffectDag|TestM10.*Effect|TestM10.*Glow|TestM10.*Reflection|TestM10.*InnerShadow|TestM10.*FillOverlay|TestM10.*Blur|TestM10.*SoftEdge' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary
```

Result:

```text
focused effectDag/effect tests: passed
coverage summary: passed; queue totals core-static=16 common-partial=363 hard-rendering=91 unsupported-preserve=397 out-of-scope=140
go test ./internal/render -count=1: passed
current clean object fixture corpus precise effectDag tag search: no effectDag objects found
go test ./... -count=1: passed
git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as a source-backed implementation of the simple supported
effectDag subset. Full effect-graph ordering/compositing remains partial, and
remaining graph-only effect declarations stay supported-scope implementation
work unless source evidence proves they are impossible for the static renderer.

## 2026-06-01 - M12 DrawingML Blend Mode Ledger Correction

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1597`
  `ST_BlendMode`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1606`
  `CT_FillOverlayEffect`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` row
  `dml-main.xsd:1597 ST_BlendMode`

Change:

- Added synthetic coverage for all five schema blend values: `over`, `mult`,
  `screen`, `darken`, and `lighten`.
- Corrected the coverage generator so `ST_BlendMode` is Partial instead of
  Unsupported. The enum is implemented for supported `fillOverlay` rendering;
  `CT_BlendEffect` graph usage remains a separate partial graph-compositing
  boundary.

Validation:

```text
go test ./internal/render -run 'TestM10.*FillOverlay|TestM10FillOverlayImplementsSchemaBlendModes|TestM10.*Effect|TestM10.*EffectDag' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary
```

Result:

```text
focused fillOverlay/blend/effect tests: passed
coverage summary: passed; queue totals core-static=16 common-partial=364 hard-rendering=91 unsupported-preserve=396 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as a source-backed ledger correction with fixture proof.
`ST_BlendMode` is not Unsupported because all of its schema enum values are
implemented by the fill-overlay renderer. Remaining blend graph behavior stays
explicitly separate from the enum support.

## 2026-06-01 - M12 Simple DrawingML Blend EffectDag Flattening

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1665`
  `CT_BlendEffect`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1597`
  `ST_BlendMode`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` row
  `dml-main.xsd:1665 CT_BlendEffect`

Change:

- `parseShapeEffectDag` now handles simple `a:blend` graph nodes by flattening
  their child `a:cont` when that child contains already-supported static
  effects.
- The renderer still reports the graph as partial, because this is not full
  DrawingML blend graph compositing. The implemented subset prevents supported
  child effects from being dropped just because they are wrapped by `a:blend`.
- The coverage generator now classifies `CT_BlendEffect` as Partial instead of
  Unsupported.

Validation:

```text
go test ./internal/render -run 'TestM10.*Blend|TestM10.*EffectDag|TestM10.*Effect|TestM10.*Glow' -count=1

python3 tools/generate_ooxml_drawingml_audit.py --print-summary
```

Result:

```text
focused blend/effectDag/effect tests: passed
coverage summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
current clean object fixture corpus precise blend tag search: no blend objects found
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9341017, worst=EPA-generate-2021-presentation.pptx slide 001 with 307961 pixels, top unsupported rendering gaps=none
```

Decision: accepted as a bounded source-backed implementation. `CT_BlendEffect`
is no longer Unsupported because supported static child effects are rendered
through the flattened effectDag path. Full blend graph compositing remains
partial M12 renderer work.

## 2026-06-01 - M12 Custom Geometry Fractional Fill Bounds

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2042`
  `CT_Path2D`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2074`
  `CT_CustomGeometry2D`
- `docs/specs/ecma-376/part1/schema/strict/pml.xsd:1209`
  `CT_Shape`
- EPA generate 2021 presentation slide 7 master object `Freeform 6`, whose
  custom geometry has fractional output bounds
  `x=0..233.30763779527558 y=26.548582677165353..99.24086614173228`.

Change:

- Added a fractional-bound polygon fill rasterizer for custom geometry solid
  fills. `renderShape` now maps `a:custGeom` fill paths through the
  source-derived fractional `xfrm` bounds instead of snapping the fill mask to
  the rounded integer target before coverage sampling.
- Preset polygons, outlines, gradients, shadows, pictures, and table rendering
  are unchanged by this step.
- Added a synthetic renderer test proving a fractional custom geometry edge is
  partially covered instead of fully rounded away or expanded.

Validation:

```text
go test ./internal/render -run 'TestRenderShapePaintsCustomGeometryFill|TestRenderShapeFlipsCustomGeometryFill|TestDrawPolygonAntialiasesEdges' -count=1

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-freeform6-fractional-fill PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-freeform6-shape-fractional-fill PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-008/micro-fixtures/shape-0003-7-Freeform-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v
```

Result:

```text
focused custom geometry tests: passed
EPA slide 7 Freeform 6 underpaint fixture: still fails exact visible-crop comparison, but improves from the recorded 3,688 differing pixels to 3,672
EPA slide 8 Freeform 6 fixture: still fails exact visible-crop comparison at 11,522 differing pixels, unchanged from the recorded manifest target scope
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
git diff --check: passed
```

Decision: accepted as a source-backed custom geometry primitive improvement.
`CT_CustomGeometry2D` and `CT_Path2D` remain Partial, not Unsupported: the
renderer can represent these static fill paths and must continue closing the
remaining exact fixture residuals. This does not satisfy M12 completion because
the clean fixture suite still requires expected-failure accounting and exact
object fixtures still fail.

## 2026-06-01 - M12 Row-Spanned Table Cell Text Minimums

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 12
- Object: `graphicFrame` id `2`, name `Table 3`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json`
- Schema anchors: `pml.xsd:1263 CT_GraphicalObjectFrame`,
  `dml-main.xsd:842 CT_GraphicalObjectData`,
  `dml-main.xsd:2423 CT_Table`, `dml-main.xsd:2386 CT_TableCell`,
  `dml-main.xsd:2398 CT_TableRow`, and
  `dml-main.xsd:2347 CT_TableCellProperties`.

Source semantics:

- `Table 3` contains vertical merge cells, including a source
  `a:tc rowSpan="10"` whose text body belongs to the full spanned cell
  rectangle, with following `a:tc vMerge="1"` continuation cells.
- The row-height text-minimum path previously assigned the full measured text
  height to the origin row only. That contradicts the `CT_TableCell/@rowSpan`
  geometry already used by the paint-time `tableCellRect`.

Implementation:

- `tableTextMinimumRowHeights` now distributes a row-spanned cell's measured
  text minimum across the rows covered by `rowSpan`, instead of applying the
  full minimum to the origin row.
- Added a synthetic row-span fixture proving that row-spanned text which fits
  the spanned rectangle does not inflate only the origin row.
- No fixture-specific row offsets, colors, fonts, sampling, or unsupported
  classifications were added.

Validation:

```text
go test ./internal/render -run 'TestTableTextMinimumRowHeightsDistributesRowSpanText|TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth|TestTableRowOffsetsWithTextMinimums|TestTableRowOffsets|TestAdjustTableRowOffsets' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-rowspan PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table15-rowspan PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/table-0003-15-Table-15/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table2-rowspan PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/table-0010-3-Table-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

python3 tools/generate_ooxml_drawingml_audit.py --print-summary
```

Result:

```text
focused table row-span tests: passed
Table 3 fixture: still failed, unchanged at 284470 differing pixels
Table 15 fixture: still failed, unchanged at 72605 differing pixels
Table 2 fixture: still failed, unchanged at 55832 differing pixels
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
coverage summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
```

Decision: accepted as source-backed `CT_TableCell/@rowSpan` text-minimum
semantics with synthetic proof and no observed table fixture regression. This
does not satisfy table parity or M12 completion: the same table object fixtures,
the clean fixture suite, and the exact Apple Notes gate still fail.

## 2026-06-01 - M12 Table Cell Anchor Center Lowering

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2347`
  `CT_TableCellProperties`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2365`
  `@anchor`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2366`
  `@anchorCtr`
- WHO HIV slide 012 `Table 3`, which remains the current top clean table
  fixture and exercises table-cell text anchoring, style fills, borders, and
  spans.

Source semantics:

- `CT_TableCellProperties` defines `anchorCtr` next to `anchor` for table-cell
  text anchoring.
- Shape text body properties already lower `anchorCtr` into Puppt's
  `TextAnchorCenter` behavior, but table cells previously parsed only
  `a:tcPr/@anchor`.

Implementation:

- `parseTableCell` now preserves `a:tcPr/@anchorCtr` on `tableCell`.
- `tableCellTextElement` lowers that value to the existing text element
  `HasTextAnchorCenter`/`TextAnchorCenter` fields.
- No fixture-specific offsets, font choices, colors, row heights, or
  unsupported classifications were added.

Validation:

```text
go test ./internal/render -run 'TestParseTableCellAnchorCenterLowersToTextElement|TestDrawShapeTextHonorsAnchorCenter|TestTableCellTextAnchorDoesNotInferRowSpanCentering|TestParseTableCellMarginsKeepsDefaultsForOmittedSides' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-anchorctr PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

go test ./... -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-anchorctr.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1

git diff --check
```

Result:

```text
focused table/text anchoring tests: passed
WHO slide 012 Table 3 fixture: still failed, unchanged at 284470 differing pixels
coverage summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate after manifest refresh: failed; 61/61 slides differ, total_diff=9340612, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
git diff --check: passed
```

Decision: accepted as bounded `CT_TableCellProperties/@anchorCtr` semantics
coverage. This does not complete table parity or M12: `Table 3`, the clean
fixture suite, and the exact Apple Notes gate still fail.

## 2026-06-01 - M12 Table Cell Overflow And Vertical Text Metadata

Source anchors:

- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2347`
  `CT_TableCellProperties`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2363`
  `@horzOverflow`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2364`
  `@vert`
- `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2369`
  `@vertOverflow`

Source semantics:

- Table-cell properties define text overflow and vertical-text attributes next
  to the table-cell text anchor attributes.
- Shape `a:bodyPr` already lowers equivalent text properties into Puppt's text
  layout and reporting path, but table-cell `a:tcPr` previously dropped them.

Implementation:

- `parseTableCell` now preserves `a:tcPr/@horzOverflow`,
  `a:tcPr/@vertOverflow`, and `a:tcPr/@vert` on `tableCell`.
- `tableCellTextElement` lowers those values to the existing text element
  overflow and vertical-text fields.
- No fixture-specific offsets, colors, font choices, row heights, picture
  sampling, or unsupported classifications were added.

Validation:

```text
go test ./internal/render -run 'TestParseTableCellAnchorCenterLowersToTextElement|TestDrawShapeTextHonorsAnchorCenter|TestTableCellTextAnchorDoesNotInferRowSpanCentering|TestParseTableCellMarginsKeepsDefaultsForOmittedSides' -count=1 -v

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-cell-text-props PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v

python3 tools/generate_ooxml_drawingml_audit.py --print-summary

go test ./internal/render -count=1

go test ./... -count=1

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-cell-text-props.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1
```

Result:

```text
focused table text-property tests: passed
WHO slide 012 Table 3 fixture: still failed, unchanged at 284470 differing pixels
coverage summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9340612, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as bounded `CT_TableCellProperties` text-property coverage.
This closes a dropped source-property path by reusing existing text
rendering/reporting behavior. It does not complete table parity or M12:
`Table 3`, the clean fixture suite, and the exact Apple Notes gate still fail.

## 2026-06-01 - M12 Table Style Cell FillRef Resolution

Source anchors:

- `dml-main.xsd:2499 CT_TableStyleCellStyle` includes
  `EG_ThemeableFillStyle` inside `tcStyle`.
- `dml-main.xsd:2440 EG_ThemeableFillStyle` permits both direct `fill` and
  theme-style `fillRef`.
- EPA Residential Wood has real package evidence in `ppt/tableStyles.xml`,
  including `tcStyle/fillRef` table-style data and slide 015
  `Google Shape;193;p12`.

Change:

- `parseTableStyleRegion` now receives the package theme fill style matrix.
- `tcStyle/fillRef` resolves through the theme fill style matrix with the
  `fillRef` placeholder color applied before flattening to the table-cell style
  fill color.
- The existing table background `fillRef` path remains intact. The style
  region still stores only a flat fill color, so gradients, patterns, effects,
  and full Office table-style precedence remain Partial.

Validation:

```text
focused table style fillRef tests: passed
EPA slide 015 Google Shape;193;p12 fixture: failed at 64393 differing pixels
EPA slide 013 Google Shape;179;p9 fixture: still failed, unchanged at 127315 differing pixels
coverage summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9340612, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as a bounded source-style resolution fix for
`CT_TableStyleCellStyle/fillRef`. It does not complete table-family parity or
M12: the EPA table fixtures, `Table 3`, the clean fixture suite, and the exact
Apple Notes gate still fail.

## 2026-06-01 - M12 Table Properties Fill And NoFill Backgrounds

Source anchors:

- `dml-main.xsd:2405 CT_TableProperties` includes `EG_FillProperties` before
  the table style choice.
- `dml-main.xsd:2440 EG_ThemeableFillStyle` is separate table-style fill
  semantics; direct `tblPr` fill/noFill is table-property source data, not a
  style-region fallback.
- EPA Residential Wood slide 013 `Google Shape;179;p9` has direct
  `a:tblPr/a:noFill` next to table style
  `{D1725187-6464-411F-8C7F-DCDDFD2443DF}`.

Change:

- `parseTableModel` now records direct table-property `solidFill` and `noFill`
  on `tableModel`.
- `renderTableGraphicFrame` paints a direct table-property background before
  falling back to the style table background.
- Direct table-property `noFill` suppresses the style table background fill.
  Table-property effects and non-solid direct table fills remain Partial and
  explicitly reported through the existing table unsupported-feature path.

Validation:

```text
focused table-property fill/noFill tests: passed
EPA slide 013 Google Shape;179;p9 fixture: still failed, unchanged at 127315 differing pixels
WHO slide 012 Table 3 fixture: still failed, unchanged at 284470 differing pixels
coverage summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9340612, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as bounded `CT_TableProperties/EG_FillProperties`
coverage. It does not complete table parity or M12: the EPA table fixture,
`Table 3`, the clean fixture suite, and the exact Apple Notes gate still fail.

## 2026-06-02 - M12 Rectangle And Picture Round Line Joins

Source anchors:

- `dml-main.xsd:2134 CT_LineJoinRound`
- `dml-main.xsd:2138 EG_LineJoinProperties`
- `dml-main.xsd:2206 CT_LineProperties`

Source evidence:

- Current object sources include visible `a:round` line joins on DrawingML table
  borders.
- The shape/picture parser already preserved `a:round` as `LineJoin=round`,
  but rectangular shape and picture outlines only passed dash, cap, align, and
  compound stroke metadata to the rectangle helper.

Change:

- `drawStyledRectOutlineCompound` now accepts a join parameter.
- Rectangular shape outlines pass `LineJoin` through to the shared helper.
- Picture outlines pass `LineJoin` and `LineCompound` through to the same helper.
- Round-join rectangle strokes use the existing path-outline renderer so the
  same `drawRoundLineJoin` behavior applies to supported path, rectangle, and
  picture outlines.

Validation:

```text
go test ./internal/render -run 'TestM06|TestRenderPicture.*Outline|Test.*LineJoin|Test.*LineDash' -count=1: passed
go test ./internal/render -count=1: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./... -count=1: passed
```

Decision: accepted as bounded `CT_LineProperties` / `EG_LineJoinProperties`
coverage. Bevel and miter still use the existing segment stroke model, gradient
or pattern stroke fills remain partial, and M12 remains incomplete until clean
fixtures and the exact Apple Notes gate pass.

## 2026-06-02 - M12 Supported Picture Metadata Is Not Unsupported

Source anchors:

- `dml-main.xsd:1455 CT_StretchInfoProperties`
- `dml-main.xsd:1460 EG_FillModeProperties`
- `dml-main.xsd:1502 CT_BlipFillProperties`

Source evidence:

- WHO HIV slide 009 `Picture 2` is a `p:pic` with `a:blip r:embed="rId4"`,
  `a:stretch/a:fillRect`, no crop, and a rectangular transform.
- Stretch/fillRect is static DrawingML picture fill semantics. The renderer
  already paints non-tiled pictures by scaling the decoded source into the
  picture target, so `fillMode=stretch` is supported metadata, not an
  unsupported record.
- Current manifests also contained supported metadata summaries
  `alphaModFix=100000`, `rotWithShape=true`, and `softEdge=203200` under
  `expected_unsupported_records`. The renderer has supported static paths for
  these source-backed primitives; unsupported records should contain explicit
  unrendered-feature messages, not supported metadata summaries.

Change:

- `ObjectStyleSummary` now separates explicit `image_unsupported` and
  `effect_unsupported` diagnostics from descriptive `image_effects` metadata.
- Micro-fixture `expected_unsupported_records` generation now reads only real
  unsupported diagnostics plus table/custom-path unsupported records. Supported
  metadata such as `fillMode=stretch`, `rotWithShape=false`, and
  `alphaModFix=...` no longer becomes an expected unsupported record.
- The existing object-debug fixture manifests were refreshed mechanically.
  `fillMode=stretch` was removed from 61 unsupported-record lists, and the
  remaining metadata-shaped records were removed from 38 manifests:
  `alphaModFix=100000` and `rotWithShape=true` appeared 36 times each, and
  `softEdge=203200` appeared once. Their descriptive `image_effects` metadata
  is unchanged.
- No picture sampling, resampling, color, crop, or source-image heuristic was
  changed.

Validation:

```text
go test ./internal/render -run TestMicroFixtureManifestsDoNotClassifyStretchFillAsUnsupported -count=1 before first manifest refresh: failed; 61 stale manifests listed fillMode=stretch as expected unsupported
first mechanical manifest refresh: removed fillMode=stretch only from spec_fixture.expected_unsupported_records in 61 manifests
metadata audit before broadened refresh: 36 alphaModFix=100000, 36 rotWithShape=true, and 1 softEdge=203200 expected unsupported records remained
second mechanical manifest refresh: removed supported metadata-shaped expected unsupported records from 38 manifests
direct manifest scan after refresh: metadata-shaped expected unsupported records=0; unique remaining expected unsupported records=0
go test ./internal/render -run 'TestExpectedUnsupportedRecordsIgnoreSupportedImageMetadata|TestM07ParsesBlipFillModeLinkAndEffects|TestRenderOutputSupportsPicture' -count=1: passed
go test ./internal/render -run 'TestMicroFixtureManifestsDoNotClassifySupportedImageMetadataAsUnsupported|TestExpectedUnsupportedRecordsIgnoreSupportedImageMetadata|TestM07ParsesBlipFillModeLinkAndEffects|TestM10.*SoftEdge|TestRenderOutputSupportsPicture' -count=1: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
targeted git diff --check: passed
clean micro-fixture suite in expected-failure accounting mode: passed; total=59 passed=0 failed=59
exact Apple Notes gate: failed; 61/61 slides differ, total_diff=9340612, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as a source-backed M12 reporting correction for `CT_Blip`,
`CT_BlipFillProperties`, `EG_FillModeProperties`, and `CT_SoftEdgesEffect`.
The renderer must not use Unsupported as a completion exception for
implementable static picture/effect semantics. `Picture 2` and related picture
fixtures still remain Partial because exact sampling/color correspondence is
not yet closed.

## 2026-06-02 - M12 Conditional Table-Style Boundary Borders

Source anchors:

- `dml-main.xsd:2480 CT_TableCellBorderStyle`
- `dml-main.xsd:2499 CT_TableStyleCellStyle`
- `dml-main.xsd:2512 CT_TableStyle`

Source evidence:

- WHO HIV slide 012 `Table 3` is a `p:graphicFrame` table with
  `a:tblPr firstRow="1" bandRow="1"` and
  `a:tableStyleId={5C22544A-7EE6-4342-B048-85BDC9FD1C3A}`.
- `ppt/tableStyles.xml` for `Medium Style 2 - Accent 1` defines
  `wholeTbl/tcBdr/insideH` as a 12,700 EMU white line and
  `firstRow/tcBdr/bottom` as a 38,100 EMU white line.
- The previous resolved profile flattened the first-row bottom border to the
  inherited 12,700 EMU `insideH` line. That is a table-style precedence bug,
  not an unsupported feature.

Change:

- Resolved table-style borders now remember when a non-`wholeTbl` conditional
  region explicitly supplies a top/bottom/left/right boundary.
- The table renderer repaints those explicit conditional-region boundaries
  after inherited inside borders, while direct cell borders still take
  precedence.
- The `Table 3` style profile now records the first-row bottom border as
  38,100 EMUs.
- No table colors, fonts, row offsets, screenshot thresholds, or unsupported
  classifications were changed.

Validation:

```text
go test ./internal/render -run 'TestM09TableStyleRegionBoundaryBorderOverridesInsideBorder|TestM09TableStyleDiagonalBordersApplyThroughResolvedCellStyle|TestM09RenderGraphicFramePaintsDiagonalCellBorders' -count=1: passed
PUPPT_TABLE_STYLE_COLOR_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json PUPPT_TABLE_STYLE_COLOR_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/table-style-color-profile-m12.json go test ./internal/render -run TestMicroFixtureTableStyleColorProfile -count=1 -v: passed; first-row bottom_border_width_emu=38100
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 still differs by 284470 pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
git diff --check for touched M12 files: passed
go test ./... -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-boundary-borders.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1: expected failure; 61/61 slides differ, total_diff=9341866, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as bounded `CT_TableCellBorderStyle` /
`CT_TableStyleCellStyle` support. It does not complete table parity or M12:
`Table 3`, the clean fixture suite, and the exact Apple Notes gate still fail.

## 2026-06-02 - M12 Authored Empty Text Paragraph Lines

Source anchors:

- `dml-main.xsd:2540 CT_TextParagraph`
- `dml-main.xsd:2873 CT_TextCharacterProperties`
- `dml-main.xsd:2994 CT_TextParagraphProperties`

Source evidence:

- WHO HIV slide 003 `TextBox 7` is a `p:sp` with `a:bodyPr wrap="square"`
  and multiple bullet paragraphs.
- The same text body contains authored empty `a:p` elements with
  `a:endParaRPr sz="2200"` between visible bullet paragraphs.
- The parser already preserved those empty paragraphs and resolved their
  end-paragraph run metrics; the layout stage skipped them because they had no
  renderable text segments.

Change:

- `textRenderLinesForElement` now emits a blank layout line for authored
  paragraphs that have no text segments, preserving resolved font size,
  paragraph spacing, alignment, line spacing, and tab stops.
- No bullet geometry, font fallback, wrapping heuristic, unsupported
  classification, or fixture threshold was changed.

Validation:

```text
go test ./internal/render -run 'TestTextRenderLinesPreserveAuthoredEmptyParagraphs|TestTextParagraphsFromNodeUsesEndParagraphDefaultForParagraphOnly|TestTextRenderLinesPreserveDrawingMLBreakRuns' -count=1: passed
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 7 still differs by 130250 pixels
go test ./internal/render -run 'TestTextRenderLinesPreserveAuthoredEmptyParagraphs|TestM08|Test.*Text.*Paragraph|Test.*Text.*Spacing' -count=1: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
git diff --check for touched M12 files: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-empty-paragraph-lines.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1: expected failure; 61/61 slides differ, total_diff=9341866, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as bounded `CT_TextParagraph` / `endParaRPr` layout
coverage. It does not complete text parity or M12: the slide 003 `TextBox 7`
object fixture still fails, the clean fixture suite remains expected-failure
only, and the exact Apple Notes gate still fails.

## 2026-06-02 - M12 Authored Slash Text Wrap Point

Source anchors:

- `dml-main.xsd:2540 CT_TextParagraph`
- `dml-main.xsd:2543 EG_TextRun`
- `dml-main.xsd:3035 CT_RegularTextRun`

Source evidence:

- EPA Residential Wood slide 013 `Google Shape;179;p9` is a
  `p:graphicFrame` table with `a:tblPr firstRow="1" bandRow="1"`.
- The first-row header text includes authored runs around
  `Emission Rate PM2.5 (g/hr)`, including `(g/`, `hr`, and `)`.
- The existing text wrapper already treated authored hyphens as wrap points,
  but treated `/` as part of an unbreakable token. That left a source-authored
  separator unavailable to table header layout.

Change:

- Text tokenization now treats an authored slash as a wrap point, preserving
  the slash on the preceding line and not inventing whitespace.
- Added synthetic coverage for plain and styled text wrapping at `/`.
- No table row offsets, colors, fonts, borders, unsupported classifications,
  fixture thresholds, or reference artifacts were changed.

Validation:

```text
go test ./internal/render -run 'TestWrapTextWithPrefixesBreaksAfterAuthoredSlash|TestStyledWordTokensExposeSlashWrapPoint|TestWrapTextWithPrefixesBreaksAfterAuthoredHyphen|TestStyledWordTokensExposeHyphenWrapPoint|TestTableRowOffsetsWithTextMinimums|TestTableTextMinimum' -count=1: passed
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-013/micro-fixtures/table-0005-179-Google-Shape-179-p9/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; moved from 127315 to 127167 differing pixels
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 remains 284470 differing pixels
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/table-0003-15-Table-15/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 15 remains 72605 differing pixels
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/table-0010-3-Table-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 2 is 55911 differing pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
go test ./internal/render -count=1: passed
go test ./... -count=1: passed
git diff --check for touched M12 files: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-slash-wrap.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; Table 3=284470, Google Shape;179;p9=127167, Table 15=72605, Table 2=55911
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1: expected failure; 61/61 slides differ, total_diff=9340975, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as bounded `CT_TextParagraph` / `CT_RegularTextRun` text
wrap coverage. It does not complete table text parity or M12: the EPA
`Google Shape;179;p9` object fixture still fails, the clean fixture suite
remains expected-failure only, and the exact Apple Notes gate still fails.

## 2026-06-02 - M12 Blocker Recheck: Picture Sampling, Table Color/AA, And Soft Edge

Source-backed probes:

- WHO HIV slide 009 `Picture 2` is a `p:pic` with
  `a:blip r:embed="rId4"`, `a:stretch/a:fillRect`, rectangular transform, no
  crop, and no mask/effect wrapper.
- WHO HIV slide 012 `Table 3` is a `p:graphicFrame` table with
  `a:tblPr firstRow="1" bandRow="1"` and
  `a:tableStyleId={5C22544A-7EE6-4342-B048-85BDC9FD1C3A}`.
- EPA slide 005 `Content Placeholder 6` is a picture placeholder with
  `a:stretch/a:fillRect` and `a:effectLst/a:softEdge rad="203200"`.

Validation:

```text
PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_OUTPUT=/tmp/puppt-picture2-fractional-bounds-search.json go test ./internal/render -run TestMicroFixturePictureFractionalBoundsSearch -count=1 -v: passed diagnostic; best candidate `converted_icc/bilinear_center` with target y offset 0.157 still worsened exact fixture diff to 154772 versus current 154741

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-current PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 remains 284470 differing pixels

PUPPT_TABLE_STYLE_COLOR_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json PUPPT_TABLE_STYLE_COLOR_PROFILE_OUTPUT=/tmp/puppt-table3-color-profile-current.json go test ./internal/render -run TestMicroFixtureTableStyleColorProfile -count=1 -v: passed; source style colors still resolve to Display P3 colors nearest the current render while the fixture residual remains broad fill/color/antialias difference

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-content-placeholder6-current PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-005/micro-fixtures/0004-7-Content-Placeholder-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Content Placeholder 6 remains 60187 differing pixels
```

Decision:

- Rejected changing picture fractional target bounds for `Picture 2`; the
  diagnostic candidate worsened the exact object fixture and did not expose a
  missing `CT_BlipFillProperties` primitive.
- Rejected a production table color/AA override for `Table 3`; the source style
  profile already maps table fills and first-row border precedence, and the
  remaining residual is not a source-backed table-layout primitive.
- Rejected a soft-edge production change for `Content Placeholder 6`; the
  fixture diagnostic records only 1,459 differing pixels inside partial-alpha
  soft-edge mask pixels, while most residual is inside full-alpha picture
  sampling/color area.

These are not Unsupported classifications. `Picture 2`, `Table 3`, and
`Content Placeholder 6` remain supported-scope M12 blockers until a later
source-backed primitive closes their object fixtures or source evidence proves
an actual static-renderer impossibility.

## 2026-06-02 - M12 Text Box Shape-Autofit Recheck

Source anchors:

- `dml-main.xsd:805 CT_NonVisualDrawingShapeProps/@txBox`
- `dml-main.xsd:2617 CT_TextShapeAutofit`
- `dml-main.xsd:2620 EG_TextAutofit`
- `dml-main.xsd:2625 CT_TextBodyProperties`

Source evidence:

- WHO HIV slide 008 `TextBox 4` is a `p:sp` with
  `p:cNvSpPr txBox="1"`, `a:bodyPr wrap="square"` and `a:spAutoFit`.
  Its current geometry target is `x=78..912 y=111..183`; the production
  shape-autofit path expands the text target to `y=187` after measuring the
  authored paragraphs.
- WHO HIV slide 005 `TextBox 11` is also a `p:sp` text box with
  `a:spAutoFit`. Its source geometry max y is 288 and current output max y is
  293.

Validation:

```text
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox4-current PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/shape-0002-5-TextBox-4/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 4 remains 26639 differing pixels

PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/shape-0002-5-TextBox-4/manifest.json PUPPT_SHAPE_OBJECT_PROFILE_OUTPUT=/tmp/puppt-textbox4-shape-profile.json go test ./internal/render -run TestMicroFixtureShapeObjectProfile -count=1 -v: passed; geometry target y=111..183, text target y=111..187, measured text 730x69

PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-008/micro-fixtures/shape-0002-5-TextBox-4/manifest.json PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=/tmp/puppt-textbox4-text-stroke-profile.json go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v: passed diagnostic; current diff 26639, got text mask x=7..730 y=10..70, reference text mask x=0..834 y=6..76, best simple text-mask shift -3 lowers only to 24737

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox11-current PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-005/micro-fixtures/shape-0009-12-TextBox-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 11 remains 22020 differing pixels

PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-005/micro-fixtures/shape-0009-12-TextBox-11/manifest.json PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=/tmp/puppt-textbox11-text-stroke-profile.json go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v: passed diagnostic; current diff 22020, text-like difference 313 pixels, best simple text-mask shift -4 lowers only to 21937
```

Decision:

- Rejected suppressing `spAutoFit` or adding a small-height exception for these
  text boxes. The source XML explicitly authors `a:spAutoFit`, the schema
  defines it as a real autofit choice, and existing focused tests intentionally
  cover shape-autofit growth/shrink behavior.
- Rejected a generic text-mask vertical shift. `TextBox 4` improves only
  partially, and `TextBox 11` shows the same shift barely affects the object
  diff because the residual is dominated by fill/edge/antialias pixels.

These text boxes remain supported-scope M12 blockers. The next accepted change
must explain the residual as a source-backed text metrics, fill/edge coverage,
or autofit primitive and prove it on focused object fixtures before production
code changes.

## 2026-06-02 - M12 Table Border Marker And Compound Line Implementation

Source anchors:

- `dml-main.xsd:2096 ST_LineEndType`
- `dml-main.xsd:2120 CT_LineEndProperties`
- `dml-main.xsd:2206 CT_LineProperties`
- `dml-main.xsd:2347 CT_TableCellProperties`
- `dml-main.xsd:2480 CT_TableCellBorderStyle`

Change:

- Table border parsing now preserves `headEnd` and `tailEnd` type, width, and
  length metadata.
- Normal and diagonal table borders render known DrawingML marker types
  (`triangle`, `stealth`, `diamond`, `oval`, and `arrow`) through the existing
  line-end marker renderer.
- Table borders now reuse the existing compound-line renderer for known
  DrawingML compound border types (`dbl`, `thickThin`, `thinThick`, and `tri`).
- Known marker and compound border metadata is no longer reported as
  Unsupported. Unknown marker names remain reported as partial unsupported
  diagnostics.

Validation:

```text
go test ./internal/render -run 'TestM09|TestRenderTableCellBorderPaintsKnownLineEndMarkers|TestRenderTableCellDiagonalBorderPaintsKnownLineEndMarkers|TestParseTableModelRecordsUnsupportedVisibleFeatures|TestRenderTableCellBorderPaintsDoubleCompoundLine' -count=1 -v: passed
go test ./internal/render -run 'TestM06RendersSchemaLineEndMarkerTypes|TestM06ReportsUnknownLineMarkerType|TestM06RendersCompoundConnectorAndCustomDash' -count=1 -v: passed
go test ./internal/render -count=1: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140
```

Decision: accepted. This implements source-backed static table border stroke
metadata instead of using Unsupported as a shortcut. It does not close M12:
the exact Apple Notes gate and clean fixture suite still have supported-scope
visual residuals.

## 2026-06-02 - M12 Shape Style FontRef Text Color Precedence

Source anchors:

- `dml-main.xsd:2252 CT_FontReference`
- `dml-main.xsd:2258 CT_ShapeStyle`
- `dml-main.xsd:2540 CT_TextParagraph`
- `dml-main.xsd:3035 CT_RegularTextRun`

Source evidence:

- WHO HIV slide 013 object 6 `Rectangle 5` is a `p:sp` whose style contains
  `<a:fontRef idx="minor"><a:schemeClr val="lt1"/></a:fontRef>`.
- The two authored text runs have size/bold properties but no direct text fill:
  one leading-space run at `sz="4000"` and one visible run at `sz="3600"`.
- The source style therefore resolves the shape's default text color to white.
  A later inherited non-placeholder paragraph default could still inject black
  as a paragraph color, and run segments preferred that inherited paragraph
  color over the element's source `fontRef` color.

Change:

- `textRenderLinesForElement` now applies element text-color defaults before
  converting styled runs into render segments, while preserving direct run
  colors.
- Inherited paragraph text styles no longer overwrite an element that already
  has a resolved text color. Other inherited paragraph style fields still flow
  normally.

Validation:

```text
go test ./internal/render -run 'TestTextRenderLinesForElementAppliesElementTextColorToStyledRuns|TestApplyInheritedTextStylesDoesNotOverrideElementTextColor' -count=1: passed

PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect5-slide13-after2 PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-013/micro-fixtures/cumulative-shape-0001-6-Rectangle-5/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Rectangle 5 improved from 12332 to 10432 visible-crop differing pixels and now renders white source fontRef text

go test ./internal/render -count=1: passed

python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=365 hard-rendering=91 unsupported-preserve=395 out-of-scope=140

PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-fontref.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167

PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1: expected failure; 61/61 slides differ, total_diff=9305437, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted. This implements a source-backed `CT_ShapeStyle` /
`CT_FontReference` text-color precedence primitive instead of treating the
visible mismatch as unsupported. It does not close the object fixture or M12:
Rectangle 5 still has 10,432 differing pixels from remaining text metrics and
edge parity, and the clean fixture suite and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Blocker Selection Policy And Google Table Recheck

Source anchors:

- `pml.xsd:1263 CT_GraphicalObjectFrame`
- `dml-main.xsd:842 CT_GraphicalObjectData`
- `dml-main.xsd:2423 CT_Table`
- `dml-main.xsd:2386 CT_TableCell`
- `dml-main.xsd:2347 CT_TableCellProperties`

Source evidence:

- EPA residential wood slide 013 `Google Shape;179;p9` is a source-backed
  table fixture with explicit table grid, row heights, `firstRow="1"`,
  `bandRow="1"`, cell margins, paragraph alignment, and table style
  `{D1725187-6464-411F-8C7F-DCDDFD2443DF}`.
- The numeric body cells explicitly author paragraph `algn="ctr"`, so the
  current centered text behavior is source-backed and must not be "fixed" by a
  visual-only left-alignment override.
- The fixture still fails at 127,167 differing pixels after previous accepted
  slash wrapping and table-style work. The visible residual is still a table
  layout/text-metrics implementation problem, not an Unsupported boundary.

Validation:

```text
PUPPT_TABLE_STYLE_COLOR_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-013/micro-fixtures/table-0005-179-Google-Shape-179-p9/manifest.json PUPPT_TABLE_STYLE_COLOR_PROFILE_OUTPUT=/tmp/puppt-google179-table-style-profile.json go test ./internal/render -run TestMicroFixtureTableStyleColorProfile -count=1 -v: passed; style={D1725187-6464-411F-8C7F-DCDDFD2443DF}, firstRow=true, bandRow=true, source fills/text colors/borders resolved
```

Decision:

- No Google table production patch was accepted from this recheck. The current
  crop mismatch does not justify overriding source-authored paragraph
  alignment, changing table style colors, or classifying the table as
  Unsupported.
- M12 blocker selection is based on source XML, schema row, render primitive,
  and fixture evidence that identify an implementable renderer change.
- Failed probes and rejected candidate fixes leave the affected source-backed
  objects in the implementation queue unless source evidence proves they
  cannot be implemented by the static PPTX renderer.

## 2026-06-02 - M12 AlphaOutset Effect Implementation

Source anchors:

- `dml-main.xsd:1255 CT_AlphaOutsetEffect`
- `dml-main.xsd:1625 EG_Effect`
- `dml-main.xsd:1671 CT_EffectList`
- `dml-main.xsd:1689 CT_EffectProperties`

Source evidence:

- `CT_AlphaOutsetEffect` has static source semantics via optional `rad`; it is
  implementable for the static renderer and therefore must not remain
  Unsupported merely because no current clean object fixture contains it.
- Existing M10 coverage previously used `alphaOutset` as the unsupported
  effectDag example. That was too broad for M12: `alphaOutset` is now a
  supported flattened effectDag child, while an unimplemented `xfrm` graph node
  remains the unsupported-effectDag regression.

Change:

- Parsed `alphaOutset` from shape/picture effect lists and flattened
  effectDag containers.
- Preserved `HasAlphaOutset` / `AlphaOutsetRadius` through render primitives.
- Rendered alphaOutset as a bounded source-backed alpha-mask expansion over the
  resolved object layer for supported static shapes and pictures.
- Regenerated the DrawingML audit from the generator so
  `CT_AlphaOutsetEffect` is Partial/common-partial, not Unsupported.

Validation:

```text
go test ./internal/render -run 'TestM10.*AlphaOutset|TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes|TestM10PictureBackendPaintsAlphaOutsetEffect' -count=1 -v: passed

go test ./internal/render -run TestM10 -count=1: passed

go test ./internal/render -count=1: passed

python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; common-partial=366, hard-rendering=91, unsupported-preserve=394
```

Decision: accepted. This removes `alphaOutset` from the Unsupported bucket
because it is a feasible static DrawingML effect with renderer proof. It does
not close M12; remaining exact Apple Notes and clean object fixture gates still
need source-backed primitive work.

## 2026-06-02 - M12 Relative Offset Effect Implementation

Source anchors:

- `dml-main.xsd:1371 CT_RelativeOffsetEffect`
- `dml-main.xsd:1643 EG_Effect/relOff`
- `dml-main.xsd:1671 CT_EffectList`
- `dml-main.xsd:1689 CT_EffectProperties`

Source evidence:

- `CT_RelativeOffsetEffect` has static source semantics through optional
  `tx`/`ty` percentages with defaults of `0%`. The effect is implementable for
  the static renderer as an object-layer translation and therefore must not
  stay in the Unsupported bucket.
- Current clean object fixtures do not contain `relOff`; synthetic fixtures are
  the right proof for this schema-row correction. Full host effect ordering
  remains Partial.

Change:

- Parsed `relOff` from shape/picture effect lists and flattened effectDag
  containers.
- Preserved `HasRelativeOffset`, `RelativeOffsetX`, and `RelativeOffsetY`
  through slide elements, theme effect refs, render scene primitives, shape
  primitives, and picture primitives.
- Rendered `relOff` as source-backed layer translation by `tx`/`ty`
  percentages of the object bounds for supported static shape and picture
  objects.
- Regenerated the DrawingML audit from the generator so
  `CT_RelativeOffsetEffect` is Partial/common-partial, not Unsupported.

Validation:

```text
go test ./internal/render -run 'TestM10.*RelativeOffset|TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes' -count=1 -v: passed

go test ./internal/render -run TestM10 -count=1: passed

go test ./internal/render -count=1: passed

python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; common-partial=367, hard-rendering=91, unsupported-preserve=393
```

Decision: accepted. This removes `relOff` from the Unsupported bucket because
it is a feasible static DrawingML effect with parser, primitive, shape-render,
and picture-render proof. It does not close M12; the exact Apple Notes and
clean object fixture gates still need source-backed primitive work.

## 2026-06-02 - M12 Transform Effect Translation Implementation

Source anchors:

- `dml-main.xsd:1382 CT_TransformEffect`
- `dml-main.xsd:1646 EG_Effect/xfrm`
- `dml-main.xsd:1671 CT_EffectList`
- `dml-main.xsd:1689 CT_EffectProperties`

Source evidence:

- `CT_TransformEffect` has static source attributes `tx`/`ty` with coordinate
  semantics plus `sx`/`sy` scale and `kx`/`ky` skew defaults. The `tx`/`ty`
  subset is implementable for the static renderer as source-coordinate
  object-layer translation and therefore must not stay in the Unsupported
  bucket.
- Non-default `sx`/`sy`/`kx`/`ky` still require full effect transform geometry
  and remain explicit Partial diagnostics, not silent drops.

Change:

- Parsed `xfrm` from shape/picture effect lists and flattened effectDag
  containers.
- Preserved `HasEffectTransform`, scale/skew metadata, and `tx`/`ty` offsets
  through slide elements, theme effect refs, render scene primitives, shape
  primitives, and picture primitives.
- Rendered `tx`/`ty` as source-backed EMU-to-canvas layer translation for
  supported static shape and picture objects.
- Regenerated the DrawingML audit from the generator so
  `CT_TransformEffect` is Partial/hard-rendering, not Unsupported.

Validation:

```text
go test ./internal/render -run 'TestM10.*Transform|TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes' -count=1 -v: passed

python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; common-partial=367, hard-rendering=92, unsupported-preserve=392
```

Decision: accepted. This removes `xfrm` from the Unsupported bucket for the
implemented `tx`/`ty` translation subset because parser, primitive,
shape-render, and picture-render proof exist. It does not close M12; scale/skew
effect transforms, exact Apple Notes parity, and clean object fixture gates
still need source-backed primitive work.

## 2026-06-02 - M12 HSL And Tint Blip Effect Rendering

Source anchors:

- `dml-main.xsd:1292 CT_HSLEffect`
- `dml-main.xsd:1378 CT_TintEffect`
- `dml-main.xsd:1475 CT_Blip`
- `dml-main.xsd:1493 CT_Blip/hsl`
- `dml-main.xsd:1495 CT_Blip/tint`
- ECMA-376 section 20.1.8.60 `tint (Tint Effect)`

Source evidence:

- `CT_HSLEffect` has static source attributes `hue`, `sat`, and `lum` with
  default zero values. The existing picture backend already applies neighboring
  blip effects in source-image space before sampling, so `hsl` belongs in that
  same primitive family instead of staying as a reported-only visible effect.
- `CT_TintEffect` has static source attributes `hue` and `amt`; ECMA-376
  describes tint as shifting effect color values toward or away from the target
  hue by the specified amount. The bounded source-space implementation
  interpolates image pixel hue toward the target hue by `amt`, with negative
  amounts moving away. This change does not broaden into generic color tuning.

Change:

- Parsed `a:blip/a:hsl` into image hue/saturation/luminance effect fields.
- Parsed `a:blip/a:tint` into image target-hue and amount effect fields.
- Lowered HSL and tint fields through render picture primitives.
- Applied HSL and tint in source-image space before picture sampling for both
  legacy and scene-lowered picture paths.
- Regenerated the DrawingML audit from the generator so `CT_HSLEffect` and
  `CT_TintEffect` are documented as rendered source-space blip behavior
  instead of reported-only partial behavior.

Validation:

```text
go test ./internal/render -run 'TestM07.*Blip.*(HSL|Tint)|TestM07ParsesBlipFillModeLinkAndEffects|TestM07PictureSourceAppliesBlipTintEffect' -count=1 -v: passed
```

Decision: accepted. `hsl` and `tint` now have parser, primitive, and
source-image render proof. This does not close M12; the exact Apple Notes and
clean object fixture gates still need source-backed primitive work.

## 2026-06-02 - M12 Simple Blip Blur Rendering

Source anchors:

- `dml-main.xsd:1264 CT_BlurEffect`
- `dml-main.xsd:1487 CT_Blip/blur`
- `dml-main.xsd:1629 EG_Effect/blur`

Source evidence:

- `CT_BlurEffect` has static source attributes `rad` and optional `grow` with
  schema default `true`. The existing renderer already has a source-backed
  alpha-weighted RGBA Gaussian blur for shape/picture object blur; simple blip
  blur can use that primitive after sampling the blip into its target.
- Combined blip blur with higher-order object effects remains partial because
  the current effect pipeline does not yet compose all image-source and object
  effects in arbitrary DrawingML order.

Change:

- Parsed `a:blip/a:blur` into source-blur image metadata.
- Lowered source-blur metadata through render picture primitives.
- Rendered simple blip blur by sampling the picture into its target layer,
  applying the existing Gaussian blur using slide-scaled `rad`, and respecting
  `grow` when choosing the paint bounds.
- Regenerated the DrawingML audit from the generator so `CT_BlurEffect` and
  `CT_Blip` no longer describe blip blur as report-only behavior.

Validation:

```text
go test ./internal/render -run 'TestM07.*Blip.*(Blur|HSL|Tint)|TestM07ParsesBlipFillModeLinkAndEffects|TestM07PictureBackendAppliesBlipBlurEffect' -count=1 -v: passed
```

Decision: accepted. Simple blip blur now has parser, primitive, and render
proof. This does not close M12; combined effect ordering, exact Apple Notes
parity, and clean object fixture gates still need source-backed primitive work.

## 2026-06-02 - M12 Blip Fill Overlay Rendering

Source anchors:

- `dml-main.xsd:1491 CT_Blip/fillOverlay`
- `dml-main.xsd:1606 CT_FillOverlayEffect`

Source evidence:

- `CT_FillOverlayEffect` requires an `EG_FillProperties` fill and a
  `ST_BlendMode` blend value. The renderer already implements these blend
  modes for supported shape and picture object fill overlays, so the feasible
  source-backed M12 work is carrying the same effect as blip source-image
  metadata.
- This is source-space blip rendering. Full arbitrary effect-graph ordering and
  host edge parity remain partial.

Change:

- Parsed `a:blip/a:fillOverlay` into source-fill-overlay metadata.
- Lowered that metadata through render picture primitives without conflating it
  with object-level `fillOverlay`.
- Applied the existing `over`, `mult`, `screen`, `darken`, and `lighten`
  overlay modes to the decoded/cropped/effected image source before sampling.
- Regenerated the DrawingML audit so `CT_FillOverlayEffect` and `CT_Blip`
  describe source-space blip rendering instead of report-only behavior.

Validation:

```text
go test ./internal/render -run 'TestM07.*Blip.*(Blur|HSL|Tint|FillOverlay)|TestM07ParsesBlipFillModeLinkAndEffects|TestM07PictureSourceAppliesBlipFillOverlayEffect|TestM07PictureBackendAppliesBlipBlurEffect' -count=1 -v: passed
```

Decision: accepted. Blip fillOverlay now has parser, primitive, and
source-image render proof. This does not close M12; exact Apple Notes parity
and clean object fixture gates still need source-backed primitive work.

## 2026-06-02 - M12 Scalar Blip AlphaMod Rendering

Source anchors:

- `dml-main.xsd:1482 CT_Blip/alphaMod`
- `dml-main.xsd:1660 CT_AlphaModulateEffect`
- `dml-main.xsd:1652 CT_EffectContainer`

Source evidence:

- `CT_AlphaModulateEffect` requires a `cont` child, so bare `a:alphaMod`
  remains invalid/unresolved and must not be silently treated as supported.
- A container whose direct or nested children collapse to `alphaModFix` scalar
  percentages can be represented in the current static source-image pipeline by
  multiplying decoded image alpha before sampling.
- Arbitrary effect containers are still graph-compositing work and remain
  explicit partial diagnostics.

Change:

- Parsed `a:blip/a:alphaMod/a:cont` when the container collapses to supported
  scalar `alphaModFix` children.
- Lowered the source alpha-modulation percentage through render picture
  primitives and object-debug image metadata.
- Applied the scalar alpha modulation in source-image space before sampling.
- Kept missing or non-scalar `alphaMod` containers as explicit partial
  diagnostics.
- Regenerated the DrawingML audit so `CT_AlphaModulateEffect` is no longer
  described as report-only behavior.

Validation:

```text
go test ./internal/render -run 'TestM07.*Blip.*Alpha|TestM07ParsesBlipFillModeLinkAndEffects|TestM07ReportsUnsupportedBlipAlphaModContainer|TestM07PictureSourceAppliesBlipAlphaColorAndLuminanceEffects|TestM07PictureSourceAppliesBlipAlphaModulateEffect' -count=1 -v: passed
```

Decision: accepted. Scalar blip alphaMod now has parser, primitive, and
source-image render proof. This does not close M12; arbitrary effect-container
composition, exact Apple Notes parity, and clean object fixture gates still
need source-backed primitive work.

## 2026-06-02 - M12 Shape 3-D Scene Metadata Reporting

Source anchors:

- `dml-main.xsd:633 CT_Point3D`
- `dml-main.xsd:638 CT_Vector3D`
- `dml-main.xsd:643 CT_SphereCoords`
- `dml-main.xsd:1033 ST_PresetCameraType`
- `dml-main.xsd:1099 ST_FOVAngle`
- `dml-main.xsd:1105 CT_Camera`
- `dml-main.xsd:1156 CT_LightRig`
- `dml-main.xsd:1163 CT_Scene3D`
- `dml-main.xsd:1171 CT_Backdrop`
- `dml-main.xsd:1195 CT_Bevel`
- `dml-main.xsd:1200 ST_PresetMaterialType`
- `dml-main.xsd:1219 CT_Shape3D`
- `dml-main.xsd:1233 CT_FlatText`
- `dml-main.xsd:1236 EG_Text3D`

Source evidence:

- `CT_Scene3D` carries required `camera` and `lightRig` children plus optional
  `backdrop`. These are static source semantics that can be detected and
  reported for local shape properties and theme effect styles.
- `CT_Shape3D` carries `z`, `extrusionH`, `contourW`, and bevel children.
  `CT_Bevel` defaults both `w` and `h` to `76200`, so `<a:bevelT/>` is visible
  3-D metadata and must not be treated as absent.
- Text-body `EG_Text3D` uses the same `CT_Shape3D` plus `CT_FlatText` source
  declarations. Because those are detectable in static source XML, they are
  reported as Partial text unsupported diagnostics rather than left in the
  Unsupported bucket.

Change:

- Parsed local `a:scene3d` under shape properties into explicit 3-D feature
  diagnostics for camera preset, field of view, zoom, camera rotation, light
  rig, light-rig rotation, and backdrop.
- Parsed theme effect-style `a:scene3d` so `a:effectRef` preserves/report 3-D
  scene metadata alongside existing `sp3d` metadata.
- Reported non-zero `a:sp3d@z`.
- Honored schema-default bevel dimensions and kept explicitly zero or
  one-dimensional bevels out of visible 3-D reports.
- Parsed `a:bodyPr/a:scene3d`, `a:bodyPr/a:sp3d`, and
  `a:bodyPr/a:flatTx` into text-body 3-D diagnostics.
- Regenerated the DrawingML audit so shape/effect-style 3-D scene, camera,
  light-rig, bevel, material, depth, and text-depth rows are Partial, not
  Unsupported.

Validation:

```text
go test ./internal/render -run 'TestRenderShapeReportsUnsupportedVisibleShape3DProperties|TestCollectSlideElements.*Shape3D|TestCollectSlideElementsParsesScene3DWithoutShape3D|TestParseStylePropertiesAppliesThemeShape3DEffectReference' -count=1 -v: passed
go test ./internal/render -run 'TestRenderShapeReportsUnsupportedVisibleShape3DProperties|TestCollectSlideElements.*Shape3D|TestCollectSlideElementsParsesScene3DWithoutShape3D|TestParseStylePropertiesAppliesThemeShape3DEffectReference|TestParseBodyPropertiesReadsText3DMetadata|TestRenderShapeReportsSpecificUnsupportedTextLayoutFeatures' -count=1 -v: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed
```

Decision: accepted. Shape/effect-style 3-D metadata is now a detected and
reported Partial static-rendering gap, not an Unsupported shortcut. Text-body
3-D metadata is also detected and reported as Partial. This does not close M12;
actual 3-D surface rendering, exact Apple Notes parity, and clean object
fixture gates remain open.

## 2026-06-02 - M12 Chart Schema Classification

Source anchors:

- `pml.xsd:1263 CT_GraphicalObjectFrame`
- `dml-main.xsd:842 CT_GraphicalObjectData`
- `dml-chart.xsd`
- `dml-chartDrawing.xsd`

Source evidence:

- Chart graphic frames are visible static slide content. The renderer already
  detects chart payloads, preserves the chart relationship/part, and emits a
  deterministic render skip record.
- Chart and chartDrawing schema declarations describe implementable static
  chart graphics and chart user-shape drawing parts. They are not a
  source-proven impossibility boundary for the static renderer.

Change:

- Reclassified `dml-chart.xsd` rows from Unsupported to Partial
  `hard-rendering` implementation work.
- Reclassified `dml-chartDrawing.xsd` rows from Unsupported to Partial
  `hard-rendering` implementation work.
- Changed chart graphic-frame render skip records from
  `render_unsupported_object` to `render_partial_object`, keeping the
  relationship/part preservation detail and avoiding a false Unsupported
  classification.
- Updated M12 wording so next-fix selection is source-backed milestone-order
  implementation evidence, not a completion shortcut.

Validation:

```text
go test ./internal/render -run 'TestM11CollectSlideElementsClassifiesChartGraphicFrame|TestM11RenderGraphicFrameReportsChartPayload' -count=1 -v: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; chart-only queue totals were superseded by the later SmartArt/diagram and lockedCanvas reclassification
```

Decision: accepted. Chart payloads remain unrendered, but they are now tracked
as implementable hard-rendering Partial work rather than Unsupported. This does
not close M12; chart rendering, exact Apple Notes parity, and clean object
fixture gates remain open.

### M12 SmartArt/diagram and lockedCanvas Unsupported Reclassification

Source/schema anchors:

```text
dml-diagram.xsd
dml-lockedCanvas.xsd
dml-main.xsd:938 CT_GvmlUseShapeRectangle
dml-main.xsd:939 CT_GvmlTextShape
dml-main.xsd:955 CT_GvmlShape
dml-main.xsd:1018 CT_GvmlGroupShape
internal/render/render_paint.go renderDiagramGraphicFrame
internal/render/render_test.go diagram drawing fallback fixtures
```

Rationale:

- `dml-diagram.xsd` describes static SmartArt data, layout, constraints,
  styles, color transforms, and related drawing semantics. Missing SmartArt
  layout implementation is not a source-proven static-renderer impossibility.
- The renderer already implements a source-backed subset: related diagram
  drawing fallbacks lower into shape/text primitives, theme color/font/fill/
  effect references resolve through the slide context, and unavailable drawing
  fallbacks or non-shape diagram subcontent are reported as partial render
  records.
- `dml-lockedCanvas.xsd` describes static DrawingML grouping content. The
  renderer should lower supported locked canvas children instead of treating
  the graphicData payload as unknown.
- `dml-main.xsd` GVML rows describe the child object model used by
  `lockedCanvas`; they are no longer out of scope once lockedCanvas lowering is
  part of the static renderer.

Change:

- Reclassified all `dml-diagram.xsd` declarations from Unsupported/common
  partial accounting into Partial `hard-rendering` implementation work.
- Reclassified `dml-lockedCanvas.xsd:8 lockedCanvas` from Unsupported into
  Partial `hard-rendering` implementation work.
- Reclassified GVML host-drawing rows from Out of renderer scope into Partial
  `common-partial` implementation work because lockedCanvas uses
  `a:CT_GvmlGroupShape`.
- Implemented `graphicData/lc:lockedCanvas` child lowering: the parser now
  treats lockedCanvas as a `CT_GvmlGroupShape`, composes the graphic-frame
  offset and group transform, and emits supported child shapes and standalone
  `txSp` text shapes through existing render primitives instead of reporting an
  unknown graphic payload.
- Kept OLE/control/media playback rows in `unsupported-preserve`; those are
  execution/playback boundaries for a static renderer rather than missing
  static vector layout primitives.
- Kept the M12 next-fix rule as source-backed milestone-order implementation
  evidence, not a completion shortcut.

Validation:

```text
go test ./internal/render -run 'TestM12.*LockedCanvas|TestMicroFixtureCoverageQueueSummaryReadsGeneratedMetadata|TestRenderGraphicFramePaintsSupportedDiagramDrawing|TestRenderGraphicFrameUsesPackageThemeForDiagramDrawing|TestDiagramDrawingElementsResolveSlideThemeColorMapAndFonts|TestDiagramDrawingElementsResolveSlideThemeFillAndEffectStyles|TestRenderGraphicFrameReportsUnsupportedDiagramContent' -count=1 -v: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=389 hard-rendering=458 unsupported-preserve=16 out-of-scope=128
go test ./...: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-gvml.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
```

Decision: accepted. SmartArt layout remains incomplete, and full lockedCanvas
and GVML host-drawing parity remains Partial, but supported lockedCanvas child
shapes and standalone `txSp` text shapes now lower through existing static
render primitives instead of being classified as Unsupported/out-of-scope. This
does not close M12; exact Apple Notes parity and clean object fixture gates
remain open.

### M12 Paragraph fontAlgn Metric Alignment

Source/schema anchors:

```text
dml-main.xsd:2979 ST_TextFontAlignType
dml-main.xsd:2994 CT_TextParagraphProperties
dml-main.xsd:3015 CT_TextParagraphProperties@fontAlgn
WHO HIV slide 003 TextBox 7 source-object.xml fontAlgn="auto"
WHO HIV slide 012 Table 3 source-object.xml fontAlgn="auto"
```

Rationale:

- `ST_TextFontAlignType` is a horizontal text-line metric alignment enum with
  `auto`, `t`, `ctr`, `base`, and `b`.
- The coverage matrix already tracked the row as Partial text work, but the
  renderer did not preserve `a:pPr@fontAlgn` in the paragraph model or apply
  metric alignment while drawing styled text segments.
- This is source-backed text primitive coverage, not a screenshot-tuning path.

Change:

- Parsed direct paragraph and list-style paragraph `fontAlgn` values.
- Carried `fontAlgn` into render lines.
- Applied top, center, and bottom metric alignment to styled horizontal text
  segments. `auto` and `base` preserve the existing baseline behavior.
- Kept vertical, bidi, and full Office text parity as Partial work.

Validation:

```text
go test ./internal/render -run 'TestTextParagraphsFromNodeCapturesParagraphFontAlign|TestTextParagraphsFromNodeInheritsListStyleFontAlign|TestSegmentFontAlignmentShiftUsesLineMetrics|TestTextParagraphsFromNodeCapturesRunBaseline|TestFaceWithSegmentKerningHonorsDrawingMLKernThreshold|TestTextRenderLinesPreserveAuthoredEmptyParagraphs' -count=1 -v: passed
go test ./internal/render -run 'TestTextParagraphsFromNodeCapturesParagraphFontAlign|TestTextParagraphsFromNodeInheritsListStyleFontAlign|TestSegmentFontAlignmentShiftUsesLineMetrics|TestM08|Test.*Text.*Baseline|Test.*Text.*Spacing|Test.*Text.*Paragraph' -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-fontalign PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 7 remains 130250 differing pixels
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-fontalign PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 remains 284470 differing pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16 common-partial=389 hard-rendering=458 unsupported-preserve=16 out-of-scope=128
go test ./...: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-fontalign.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v: expected failure; 61/61 slides differ, total_diff=9305437, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as bounded `ST_TextFontAlignType` /
`CT_TextParagraphProperties@fontAlgn` coverage for horizontal styled text. It
does not close M12: the targeted object fixtures, clean fixture suite, and
exact Apple Notes gate remain open.

### M12 CT_Blip cstate Metadata Carry-Through

Source/schema anchors:

```text
dml-main.xsd:1466 ST_BlipCompression
dml-main.xsd:1475 CT_Blip
dml-main.xsd:1498 CT_Blip@cstate
EPA Generate slide 003 Picture 25 source-object.xml: <a:blip r:embed="rId3" cstate="print">
```

Rationale:

- `CT_Blip@cstate` is authored DrawingML source metadata for BLIP compression
  state. It is static source semantics and should be preserved in Puppt's
  picture metadata/debug surface.
- The current Picture 25 fixture remains a picture sampling residual, but
  `cstate` itself is not an unsupported visible effect and should not be
  dropped or converted to an Unsupported record.
- This is a metadata carry-through fix, not a sampling, color, or reference
  image change.

Change:

- Parsed `a:blip/@cstate` into the resolved slide element.
- Carried the value through `renderPicturePrimitive`.
- Added `cstate=...` to object debug `image_effects` summaries while leaving
  `image_unsupported` empty for supported metadata.
- Updated the `CT_Blip` coverage-matrix evidence note.

Validation:

```text
go test ./internal/render -run 'TestM07ParsesBlipFillModeLinkAndEffects|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestObjectStyleSummaryIncludesImageAndTableProperties' -count=1 -v: passed
go test ./internal/render -run 'TestM07|TestRenderPicturePrimitiveFromElement|TestObjectStyleSummaryIncludesImageAndTableProperties|TestRenderOutputSupportsPicture|TestRenderElementsPaintsShapeBlipFill|TestRenderElementsPaintsRotatedShapeBlipFill|TestCollectSlideElementsParsesBlipRotWithShape' -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-picture25-cstate PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-003/micro-fixtures/0004-26-Picture-25/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Picture 25 remains 65,347 visible-crop differing pixels
jq '.resolved_style.image_effects, .resolved_style.image_unsupported' /tmp/puppt-picture25-cstate/current-object.json: image_effects contains ["fillMode=stretch", "cstate=print"] and image_unsupported is null
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals unchanged at core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./...: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-cstate.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v: expected failure; 61/61 slides differ, total_diff=9305437, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
git diff --check -- internal/render/render_types.go internal/render/render_parse.go internal/render/render_scene.go internal/render/render_object_debug.go internal/render/render_m07_test.go internal/render/render_scene_test.go internal/render/render_object_debug_test.go tools/generate_ooxml_drawingml_audit.py docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md docs/renderer-coverage-summary.json docs/RENDERING.md docs/renderer-milestones/12-final-conformance-and-release-audit.md docs/RENDERER_COMPLETION_CHECKLIST.md docs/RENDERER_EXPERIMENT_LOG.md: passed
```

Decision: accepted as bounded `CT_Blip@cstate` metadata preservation and
reporting. It does not close M12: Picture 25 and the higher-priority picture
fixtures still fail due source sampling/color correspondence, the clean fixture
suite still requires expected-failure accounting, and the exact Apple Notes gate
remains open.

### M12 ST_TextAutonumberScheme Common Marker Formats

Source/schema anchors:

```text
dml-main.xsd:2660 ST_TextBulletStartAtNum
dml-main.xsd:2666 ST_TextAutonumberScheme
dml-main.xsd:2747 CT_TextAutonumberBullet
WHO HIV slide 006 Rectangle 6 source-object.xml: buAutoNum type="arabicPeriod", including startAt="2"
WHO HIV slide 007 Rectangle 7 source-object.xml: buAutoNum type="alphaLcPeriod"
```

Rationale:

- `ST_TextAutonumberScheme` explicitly declares alpha, arabic, Roman,
  circled, East Asian, Thai, Hindi, and Hebrew numbering families.
- The renderer already parsed `a:buAutoNum/@type` and `@startAt`, but common
  schema variants beyond a narrow alpha/arabic subset collapsed to the default
  arabic-period marker.
- Common Latin alpha, arabic, and Roman marker text is deterministic static
  source semantics. Locale-specific numbering families and picture bullets
  remain Partial implementation work rather than Unsupported.

Change:

- Added deterministic marker formatting for common alpha, arabic, and Roman
  `ST_TextAutonumberScheme` variants, including both-parentheses,
  right-parenthesis, period, and plain arabic forms.
- Kept unimplemented locale-specific numbering families on the existing
  arabic-period fallback path and documented that as Partial.
- Updated the DrawingML coverage generator, coverage matrix, rendering docs,
  M12 milestone, and completion checklist.

Validation:

```text
go test ./internal/render -run 'TestTextParagraphsFromNodeNumbersAutoBullets|TestTextParagraphsFromNodeInheritsStyledAutoNumberBullets|TestAutoNumberBulletFormatsCommonDrawingMLSchemes' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect6-autonum PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-006/micro-fixtures/shape-0004-7-Rectangle-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Rectangle 6 remains 192327 differing pixels
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect7-autonum PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-007/micro-fixtures/shape-0005-8-Rectangle-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Rectangle 7 remains 56812 differing pixels
go test ./internal/render -run 'TestM08|TestTextParagraphsFromNode.*Auto|TestAutoNumberBullet' -count=1: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./...: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-autonum.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v: expected failure; 61/61 slides differ, total_diff=9305437, worst=EPA-generate-2021-presentation.pptx slide 001 with 307925 pixels, top unsupported rendering gaps=none
```

Decision: accepted as bounded common `ST_TextAutonumberScheme` marker-format
coverage. This implements supported source semantics and does not close the
remaining object fixtures or M12; those residuals stay supported-scope
rendering work unless source evidence later proves an impossible static
renderer boundary.

### M12 CT_TextParagraphProperties Direction And Line-Break Flags

Source/schema anchors:

```text
dml-main.xsd:2994 CT_TextParagraphProperties
dml-main.xsd:3013 CT_TextParagraphProperties@rtl
dml-main.xsd:3014 CT_TextParagraphProperties@eaLnBrk
dml-main.xsd:3016 CT_TextParagraphProperties@latinLnBrk
dml-main.xsd:3017 CT_TextParagraphProperties@hangingPunct
WHO HIV slide 003 TextBox 7 source-object.xml: rtl="0" eaLnBrk="1" latinLnBrk="0" hangingPunct="1"
WHO HIV slide 012 Table 3 source-object.xml: rtl="0" eaLnBrk="1" latinLnBrk="0" hangingPunct="1"
```

Rationale:

- `CT_TextParagraphProperties` line-break and paragraph-direction flags are
  authored source semantics adjacent to already preserved paragraph alignment,
  margins, tab stops, and spacing.
- `TextBox 7` and `Table 3` both carry these flags in failing supported-scope
  fixtures. The fixtures author `rtl="0"`, so they should not gain unsupported
  records merely for preserving metadata.
- Authored `rtl="1"` is a real paragraph-layout requirement. Until bidi layout
  is implemented, it must be reported as an LTR fallback instead of silently
  disappearing.

Change:

- Parsed direct paragraph and list-style `rtl`, `eaLnBrk`, `latinLnBrk`, and
  `hangingPunct` attributes with explicit `Has...` flags so authored `false`
  differs from a missing attribute.
- Preserved the flags through render text primitives.
- Added object-debug `resolved_style.text_paragraph_properties` metadata.
- Added a specific partial text diagnostic for authored `rtl="1"` paragraphs.

Validation:

```text
go test ./internal/render -run 'TestTextParagraphsFromNodeCapturesParagraphLineBreakFlags|TestTextParagraphsFromNodeInheritsParagraphLineBreakFlags|TestM08TextLayoutReportsAuthoredRTLParagraphFallback|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-paragraph-flags PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 7 remains 130250 differing pixels
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-paragraph-flags PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 remains 284470 differing pixels
jq '.resolved_style.text_paragraph_properties, .unsupported' /tmp/puppt-textbox7-paragraph-flags/current-object.json: text_paragraph_properties contains ["rtl=false","eaLnBrk=true","latinLnBrk=false","hangingPunct=true"] and unsupported is null
jq '.resolved_style.text_paragraph_properties, .unsupported' /tmp/puppt-table3-paragraph-flags/current-object.json: text_paragraph_properties contains ["rtl=false","eaLnBrk=true","latinLnBrk=false","hangingPunct=true"] and unsupported is null
go test ./internal/render -run 'TestM08|TestTextParagraphsFromNodeCapturesParagraphLineBreakFlags|TestTextParagraphsFromNodeInheritsParagraphLineBreakFlags|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects' -count=1: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
```

Decision: accepted as bounded `CT_TextParagraphProperties` source-metadata
preservation and partial fallback reporting for authored RTL paragraph
direction. This does not close M12: the targeted `TextBox 7` and `Table 3`
fixtures still fail, and the remaining text/table residuals remain
supported-scope implementation work.

### M12 Direct Table-Cell Gradient And Pattern Fills

Source/schema anchors:

```text
dml-main.xsd:144 EG_FillProperties
dml-main.xsd:1438 CT_GradientFillProperties
dml-main.xsd:1569 CT_PatternFillProperties
dml-main.xsd:1587 CT_FillProperties
```

Rationale:

- Table cell properties consume DrawingML fill properties, and static gradient
  and pattern fills are renderable with the existing `backgroundPaint` model.
- The table renderer already used shared paint for table backgrounds, but cell
  fills were narrowed to `color.RGBA`, so direct `gradFill`/`pattFill` and
  style fill references could lose source paint semantics.
- Unsupported reporting must stay narrow: `blipFill` and unresolved `grpFill`
  table-cell fills remain reported as image/group cell fills, but gradient and
  pattern cell fills are implemented.

Change:

- Added `backgroundPaint` storage for direct table cells and table style
  regions.
- Parsed direct `a:tcPr/a:gradFill` and `a:tcPr/a:pattFill` into the shared
  paint model.
- Rendered table cell fills through the same solid/gradient/pattern paint path
  used by table backgrounds and other DrawingML fill surfaces.
- Updated unsupported table feature detection so only image/group cell fills
  keep the cell-fill unsupported report.

Validation:

```text
go test ./internal/render -run 'TestRenderGraphicFramePaintsGradientTableCellFill|TestParseTableModelRecordsUnsupportedVisibleFeatures|TestTableCellFillDirectNoFillSuppressesStyleFill' -count=1 -v: passed
go test ./internal/render -run 'TestRenderGraphicFrame|TestParseTable|TestTableCell|Test.*Table.*Fill' -count=1: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals unchanged at core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-cell-paints.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
```

Decision: accepted as feasible static table-fill rendering. This does not close
M12 because image/tile fill details, advanced gradient clauses, exact Apple
Notes parity, and clean object fixture gates remain incomplete.

### M12 Unsupported Classification Rule Tightening

Source/schema anchors:

```text
pml.xsd:813 AG_Ole
pml.xsd:840 CT_OleObject
pml.xsd:852 CT_Control
dml-main.xsd:49 EG_Media
```

Rationale:

- Unsupported requires source evidence proving the static PPTX renderer cannot
  represent the declaration as still output.
- The coverage matrix still had generated prose that described Unsupported too
  broadly, which could hide implementable static PresentationML/DrawingML gaps.
- The only remaining Unsupported rows are active or time-based payloads:
  OLE runtime, active controls, and audio/video playback. Chart, SmartArt,
  lockedCanvas, 3-D, effects, tables, pictures, and text remain Partial or
  hard-rendering implementation work when their static semantics are
  renderable.

Change:

- Updated `tools/generate_ooxml_drawingml_audit.py` status definitions,
  promotion rules, and queue definitions so Unsupported requires
  source-proven static-renderer impossibility plus preservation/reporting.
- Regenerated `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` and
  `docs/renderer-coverage-summary.json`.
- Updated `docs/RENDERING.md` and M12 evidence text to keep out-of-scope
  separate from Unsupported and to describe remaining table gaps as Partial
  implementation work.

Validation:

```text
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
Unsupported-row audit: passed; exactly 16 Unsupported rows remain, all OLE/control/media runtime or playback impossibility rows
rg stale wording scan across tools/generate_ooxml_drawingml_audit.py, docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md, docs/RENDERING.md, docs/renderer-milestones/12-final-conformance-and-release-audit.md, docs/RENDERER_COMPLETION_CHECKLIST.md, and docs/RENDERER_EXPERIMENT_LOG.md: no hits for obsolete Unsupported wording or typo variants
git diff --check -- tools/generate_ooxml_drawingml_audit.py docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md docs/renderer-coverage-summary.json docs/RENDERING.md docs/renderer-milestones/12-final-conformance-and-release-audit.md docs/RENDERER_COMPLETION_CHECKLIST.md docs/RENDERER_EXPERIMENT_LOG.md: passed
```

Decision: accepted as M12 audit-doctrine cleanup. This does not close M12:
clean object fixtures and the exact Apple Notes gate still fail, and visible
static PresentationML/DrawingML residuals remain implementation work.

### M12 Picture Source-Blur Outline Preservation

Source/schema anchors:

```text
dml-main.xsd:2822 CT_Blip
dml-main.xsd:2944 CT_BlipFillProperties
dml-main.xsd:3054 CT_Picture
dml-main.xsd:2598 CT_LineProperties
```

Rationale:

- DrawingML `a:blip/a:blur` is a source-image effect. The authored picture
  outline belongs to the picture shape properties and should be painted after
  the blurred source image, not blurred with it.
- The unrotated path already ended up painting the outline through the shared
  post-picture outline path, but the rotated source-blur path drew the outline
  into the temporary image layer before blur.
- This is static DrawingML content and remains implementation work, not an
  Unsupported boundary.

Change:

- Removed the picture line from the temporary source-blur sampling primitive.
- Painted source-blur picture outlines after the blurred image layer.
- Added a rotated-outline helper that draws the outline in local target space,
  rotates it with the picture, and composites it over the blurred image.
- Added a regression proving exact green outline pixels survive a rotated
  source-blur render.

Validation:

```text
go test ./internal/render -run 'TestM07PictureBackend(AppliesBlipBlurEffect|KeepsRotatedOutlineOutsideBlipBlur)' -count=1: passed
go test ./internal/render -run TestM07 -count=1: passed
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-source-blur-outline.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
M12 active wording scan: no hits for obsolete priority wording or typo variants
```

Decision: accepted as a supported static picture-effect fix. This does not
close M12: the 59 tracked clean object fixtures and exact Apple Notes gate still
need source-backed implementation work.

### M12 Explicit Empty Bullet Paragraph Preservation

Source/schema anchors:

```text
dml-main.xsd:2540 CT_TextParagraph
dml-main.xsd:2751 CT_TextCharBullet
dml-main.xsd:2760 EG_TextBullet
dml-main.xsd:2873 CT_TextCharacterProperties
dml-main.xsd:2994 CT_TextParagraphProperties
```

Rationale:

- WHO HIV slide 003 `TextBox 7` authors empty bullet paragraphs with local
  `a:pPr` containing `a:buFont typeface="Arial"`, `a:buChar char="•"`,
  `marL="285750"`, and `indent="-285750"`, followed only by `a:endParaRPr`.
- The previous empty-paragraph cleanup suppressed all bullets when the
  paragraph had no text runs. That was correct for empty paragraphs with no
  local bullet choice, but wrong for source-authored empty `buChar` paragraphs.
- This is static DrawingML text layout content and remains implementation work,
  not an Unsupported boundary.

Change:

- Preserved local bullet choices on empty paragraphs while keeping the existing
  no-bullet blank-line behavior for empty paragraphs without `buChar`,
  `buAutoNum`, or `buNone`.
- Added synthetic parser and render-line regressions for explicit empty
  `buChar` paragraphs and blank empty paragraphs.
- Updated the generated DrawingML coverage note for text paragraph rows to
  mention explicit empty bullet paragraph rendering.

Validation:

```text
go test ./internal/render -run 'TestTextParagraphsFromNodePreservesEmptyParagraphs|TestTextParagraphsFromNodePreservesExplicitEmptyBulletParagraphs|TestTextRenderLinesPreserveAuthoredEmptyParagraphs|TestTextRenderLinesPreserveExplicitEmptyBulletParagraphs' -count=1: passed
go test ./internal/render -run 'TestTextParagraphsFromNode|TestTextRenderLines|TestTextLayoutParagraphLines|TestM08' -count=1: passed
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-empty-bullet PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; visible crop still differs by 130,250 pixels
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-empty-bullet.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
```

Decision: accepted as source-semantics coverage for `CT_TextParagraph` /
`EG_TextBullet` rendering. This does not close M12: the targeted `TextBox 7`
fixture, clean object fixture suite, and exact Apple Notes gate still fail.

### M12 Text Run Language Preservation

Source/schema anchors:

```text
dml-main.xsd:2544 CT_TextParagraph/endParaRPr
dml-main.xsd:2873 CT_TextCharacterProperties
dml-main.xsd:2891 CT_TextCharacterProperties@lang
dml-main.xsd:3037 CT_RegularTextRun/rPr
```

Rationale:

- WHO HIV slide 003 `TextBox 7`, WHO slide 012 `Table 3`, and neighboring
  clean text/table fixtures author `a:rPr/@lang` and `a:endParaRPr/@lang`
  values such as `en-US`, `en-ES`, and `en-GB`.
- `@lang` is source text metadata used by text shaping and font fallback
  decisions. Dropping it before the render-line stage loses a source-backed
  input even when current LTR Latin output does not visibly change.
- This is implementable static DrawingML text metadata, not an Unsupported
  boundary.

Change:

- Added language storage to paragraph defaults, text runs, and text line
  segments.
- Parsed `CT_TextCharacterProperties@lang` from direct run properties,
  paragraph default run properties, and `endParaRPr`.
- Resolved segment language from direct run language first, paragraph default
  language second.
- Updated the generated DrawingML text coverage note to mention
  `CT_TextCharacterProperties@lang` preservation.

Validation:

```text
go test ./internal/render -run 'TestTextParagraphsFromNodeCapturesRunLanguage|TestTextParagraphsFromNodeCapturesRunFontFamily|TestTextParagraphsFromNodePreservesEmptyParagraphs' -count=1: passed
go test ./internal/render -run 'TestTextParagraphsFromNode|TestTextRenderLines|TestTextLayoutParagraphLines|TestM08' -count=1: passed
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-lang PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; visible crop still differs by 130,250 pixels
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-lang.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167
go test ./...: passed
git diff --check on touched files: passed
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
```

Decision: accepted as source-semantics preservation for
`CT_TextCharacterProperties@lang`. This does not close M12: clean object
fixtures and the exact Apple Notes gate still fail.

### M12 Table Style Text Defaults In Cell Layout

Source/schema anchors:

```text
dml-main.xsd:2471 CT_TableStyleTextStyle
dml-main.xsd:2506 CT_TablePartStyle/tcTxStyle
dml-main.xsd:2423 CT_Table
dml-main.xsd:2386 CT_TableCell
```

Rationale:

- WHO HIV slide 012 `Table 3` uses `ppt/tableStyles.xml` conditional
  `tcTxStyle` regions with `fontRef`, text color, and bold first-row/column
  styling.
- The renderer parsed table-style text color, bold, italic, and font family,
  but font family and italic stayed only on the wrapper `slideElement`.
  Styled table-cell paragraphs with explicit runs resolve actual render
  segments from paragraph/run fields, so parsed defaults could be lost before
  layout.
- `CT_TableStyleTextStyle` is static DrawingML table text styling and remains
  implementation work, not an Unsupported boundary.

Change:

- `tableCellTextElement` now propagates table-style text defaults into copied
  table-cell paragraphs before text layout.
- Added a table italic helper matching the existing bold-copy behavior.
- Added font-family default propagation that fills missing paragraph font
  families while preserving direct run font families.
- Updated the generated DrawingML table-style coverage note.

Validation:

```text
go test ./internal/render -run 'TestTableTextParagraphsWithItalicCopiesParagraphs|TestTableTextParagraphsWithFontFamilySuppliesParagraphDefault|TestTableCellTextElementAppliesStyleTextDefaultsToSegments|TestTableTextParagraphsWithBoldCopiesParagraphs|TestTableTextParagraphsWithColorOverridesParagraphDefaultsButPreservesRuns|TestParseTableStylesReadsDirectTableTextFontAndItalic' -count=1: passed
PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-text-defaults go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; crop still differs by 284,470 pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-text-defaults.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167, EPA Picture 2=95960
go test ./...: passed
git diff --check on touched files: passed
M12 active wording scan: no hits for stale Unsupported-boundary wording or typo variants
```

Decision: accepted as source-semantics coverage for
`CT_TableStyleTextStyle`. This does not close M12: the targeted `Table 3`
fixture, clean object fixture suite, and exact Apple Notes gate still fail.

### M12 Text Body rtlCol Preservation

Source/schema anchors:

```text
dml-main.xsd:2625 CT_TextBodyProperties
dml-main.xsd:2637 CT_TextBodyProperties@rtlCol
dml-main.xsd:2653 CT_TextBody
pml.xsd:1209 CT_Shape
```

Rationale:

- WHO HIV slide 003 `TextBox 7` authors `a:bodyPr wrap="square" rtlCol="0"`
  with `a:spAutoFit`.
- The renderer already preserved text-body wrap, overflow, anchoring, columns,
  and autofit metadata, but dropped `rtlCol` before primitive lowering and
  object-debug reporting.
- `rtlCol="0"` is supported metadata for this single-column fixture and should
  not become an Unsupported record. Authored `rtlCol="1"` on multi-column text
  remains a partial static layout gap until right-to-left column order is
  implemented.

Change:

- Added `rtlCol` storage to `slideElement` and render text primitives.
- Parsed `CT_TextBodyProperties@rtlCol` from `a:bodyPr`.
- Inherited missing `rtlCol` from placeholder body properties.
- Added `resolved_style.text_body_properties` object-debug output with
  `rtlCol=<bool>`.
- Added a static text partial diagnostic for authored right-to-left
  multi-column order.
- Updated the generated DrawingML text-body coverage note.

Validation:

```text
go test ./internal/render -run 'TestParseBodyPropertiesReadsTextAnchor|TestM08TextPrimitiveReportsFontResolutionAndTextUnsupportedModes|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestResolveSlidePlaceholdersInheritsUnspecifiedBodyTextProperties' -count=1: passed
go test ./internal/render -run 'TestParseBodyProperties|TestM08Text|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestResolveSlidePlaceholdersInheritsUnspecifiedBodyTextProperties' -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-rtlcol PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; visible crop still differs by 130,250 pixels
jq '.resolved_style.text_body_properties, .resolved_style.text_paragraph_properties, .unsupported' /tmp/puppt-textbox7-rtlcol/current-object.json: text_body_properties contains ["rtlCol=false"], text_paragraph_properties contains ["rtl=false","eaLnBrk=true","latinLnBrk=false","hangingPunct=true"], and unsupported is null
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-rtlcol.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167, EPA Picture 2=95960
go test ./...: passed
git diff --check on touched files: passed
M12 active wording scan: no hits for stale Unsupported-boundary wording or typo variants
```

Decision: accepted as source-semantics preservation for
`CT_TextBodyProperties@rtlCol`. This does not close M12: the targeted
`TextBox 7` fixture, clean object fixture suite, and exact Apple Notes gate
still fail.

### M12 Text Caps Rendering

Source/schema anchors:

```text
dml-main.xsd:2866 ST_TextCapsType
dml-main.xsd:2873 CT_TextCharacterProperties
dml-main.xsd:2899 CT_TextCharacterProperties@cap
dml-main.xsd:2653 CT_TextBody
```

Rationale:

- `ST_TextCapsType` defines the static text-transform values `none`, `small`,
  and `all`; `CT_TextCharacterProperties@cap` defaults to `none`.
- The coverage matrix described caps as consumed by M08 text layout, but the
  current renderer dropped `a:rPr@cap` before segment measurement and drawing.
- This is implementable source-backed static DrawingML text behavior. It should
  not be left as an implicit unsupported gap or hidden behind fixture residuals.
- WHO HIV slide 003 `TextBox 7` authors `cap="none"` in relevant runs, so the
  object fixture is a no-op guard for explicit override preservation rather than
  expected visual improvement.

Change:

- Added cap storage to paragraph styles, paragraph defaults, runs, and text
  render segments.
- Parsed `CT_TextCharacterProperties@cap` from direct run and `defRPr`
  properties, including explicit `cap="none"` overrides.
- Carried caps through paragraph-style merging and application.
- Rendered `cap="all"` as uppercase text before wrapping/measurement/drawing.
- Rendered `cap="small"` as uppercase text with lowercase source letters split
  into smaller text segments.
- Updated the generated DrawingML common text coverage note.

Validation:

```text
go test ./internal/render -run 'TestTextParagraphsFromNodeParsesRunCaps|TestTextParagraphsFromNodeParsesRunCharacterSpacing|TestTextRunFromNodeReadsDrawingMLKernThreshold|TestMeasureStyledSegmentsIncludesCharacterSpacing' -count=1: passed
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-caps PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; visible crop still differs by 130,250 pixels because the source object authors cap="none"
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./...: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-caps.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167, EPA Picture 2=95960
```

Decision: accepted as source-semantics coverage for
`ST_TextCapsType` / `CT_TextCharacterProperties@cap`. This does not close M12:
the targeted `TextBox 7` fixture, clean object fixture suite, and exact Apple
Notes gate still fail.

### M12 Table Blank Paragraph Row Minimums

Source/schema anchors:

```text
dml-main.xsd:2386 CT_TableCell
dml-main.xsd:2398 CT_TableRow
dml-main.xsd:2540 CT_TextParagraph
dml-main.xsd:2873 CT_TextCharacterProperties
```

Rationale:

- WHO HIV slide 012 `Table 3` authors row-spanned table cells whose
  `a:txBody` contains blank `a:p` paragraphs with `a:endParaRPr sz="1600"`
  before visible text.
- Text parsing already preserves authored empty paragraphs as blank layout
  lines, but table row minimum measurement still skipped cells when the
  aggregate trimmed table-cell text was empty.
- This is implementable static DrawingML table/text behavior. Blank paragraph
  line boxes should contribute to table row sizing without drawing visible
  glyphs; they should not be treated as Unsupported or ignored because the
  current object fixture still has residual table-layout differences.

Change:

- Added a table text-measurement guard that recognizes authored paragraph/run
  metrics, including blank paragraphs with font/spacing/line metrics and NBSP
  runs with run font metrics.
- Changed `tableTextMinimumRowHeights` and `measuredTableCellTextHeight` to
  use that guard instead of requiring non-empty trimmed aggregate text.
- Added focused coverage for a zero-height table row sized by authored blank
  paragraph line boxes.
- Updated the generated `CT_TableCell` coverage note.

Validation:

```text
go test ./internal/render -run 'TestTableTextMinimumRowHeightsMeasuresAuthoredBlankParagraphs|TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth|TestTableTextMinimumRowHeightsDistributesRowSpanText|TestTableRowOffsetsWithTextMinimums' -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-blank-paras PUPPT_MICRO_FIXTURE_MANIFEST=/Users/artpar/workspace/code/puppt/testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; visible crop still differs by 284,470 pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-blank-table-paragraphs.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167, EPA Picture 2=95960
```

Decision: accepted as source-semantics coverage for table text minimum-height
measurement. This does not close M12: the targeted `Table 3` fixture, clean
object fixture suite, and exact Apple Notes gate still fail.

### M12 Inherited-Size Baseline Runs

Source/schema anchors:

```text
dml-main.xsd:2873 CT_TextCharacterProperties
dml-main.xsd:2904 CT_TextCharacterProperties@baseline
dml-main.xsd:3035 CT_RegularTextRun
dml-main.xsd:2540 CT_TextParagraph
```

Rationale:

- WHO HIV slide 003 `TextBox 7` authors a superscript footnote run as
  `a:rPr baseline="30000"` with no local `sz`.
- DrawingML run properties inherit missing character properties from the
  surrounding paragraph/text defaults; the baseline shift code already used a
  fallback font size for positioning, but measurement and drawing scaled the
  run only when a local/paragraph font size was known before layout.
- This is implementable static text behavior. It should be rendered through
  the existing text pipeline and remain supported-scope work, not an
  Unsupported record.

Change:

- Added fallback-aware text segment font-size resolution for render-time
  measurement and drawing.
- Threaded the element fallback font size into styled paragraph wrapping,
  alignment/justification measurement, and segmented line metrics.
- Preserved existing explicit-size baseline behavior; only baseline segments
  with inherited size use the fallback before applying the baseline scale.
- Updated the generated DrawingML text coverage note.

Validation:

```text
go test ./internal/render -run 'TestBaselineRunWithoutLocalSizeUsesElementFallbackForRenderSize|TestTextParagraphsFromNodeCapturesRunBaseline|TestTextParagraphsFromNodeParsesRunCaps|TestTextParagraphsFromNodePreservesRunLanguage|TestMeasureStyledSegmentsIncludesCharacterSpacing' -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-baseline-fallback PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; visible crop still differs by 130,250 pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-baseline-fallback.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284470, Picture 2=154741, TextBox 7=130250, Google Shape;179;p9=127167, EPA Picture 2=95960
```

Decision: accepted as source-semantics coverage for inherited-size baseline
runs. This does not close M12: the targeted `TextBox 7` fixture, clean object
fixture suite, and exact Apple Notes gate still fail.

### M12 EPA Table Font-Precedence Probe

Source/schema anchors:

```text
pml.xsd:1263 CT_GraphicalObjectFrame
dml-main.xsd:2423 CT_Table
dml-main.xsd:2386 CT_TableCell
dml-main.xsd:2347 CT_TableCellProperties
dml-main.xsd:2380 CT_TableStyleTextStyle
```

Rationale:

- EPA Residential Wood slide 013 `Google Shape;179;p9` remains a failing
  source-backed table fixture at 127,167 differing pixels.
- A diagnostic showed the table style resolves `wholeTbl`/`firstRow` text to
  Calibri, but rendered header segments still used Arial after inherited
  presentation defaults populated table-cell paragraph font families.
- A candidate made table styles override inherited paragraph font families
  unless a source-local paragraph/run font was marked explicit.

Validation:

```text
go test ./internal/render -run 'TestTableTextParagraphsWithFontFamilySuppliesParagraphDefault|TestTableTextParagraphsWithFontFamilyPreservesSourceParagraphDefault|TestTableCellTextElementAppliesStyleTextDefaultsToSegments|TestApplyThemeFontFamiliesResolvesTableCellParagraphFonts' -count=1 -v: passed during candidate
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-google179-table-font-precedence PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-013/micro-fixtures/table-0005-179-Google-Shape-179-p9/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; regressed from 127,167 to 152,626 differing pixels
go test ./internal/render -run 'TestTableTextParagraphsWithFontFamilySuppliesParagraphDefault|TestTableCellTextElementAppliesStyleTextDefaultsToSegments' -count=1: passed after rollback
```

Decision: rejected and rolled back. The source/style precedence question stays
in the table implementation queue, but this candidate worsened the object
fixture and is not an accepted M12 renderer change. The EPA table remains
supported-scope work on this basis.

### M12 Text Line Break Run Metrics

Source/schema anchors:

```text
dml-main.xsd:2543 EG_TextRun
dml-main.xsd:2873 CT_TextCharacterProperties
dml-main.xsd:2957 CT_TextLineBreak
pml.xsd:1209 CT_Shape
```

Rationale:

- WHO HIV slide 003 `Rectangle 3` authors a DrawingML hard break as
  `<a:br><a:rPr sz="3600" b="1" dirty="0"/></a:br>` between two bold text
  runs.
- `CT_TextLineBreak` permits `a:rPr`, so the break has text-run metrics even
  though it does not draw visible glyphs.
- The parser already preserved the break run as newline text with run
  properties, but line splitting discarded those properties when it cut on the
  newline.
- This is implementable static DrawingML text behavior. It should flow through
  the existing text layout path and remain supported-scope work while the
  object fixture still has residual visual differences.

Change:

- Preserved the break run as an empty metric segment at the end of the
  preceding rendered line.
- Reused existing segmented line measurement so break-run font size and style
  affect line height without painting a glyph.
- Added focused synthetic coverage for hard-break run metrics.
- Updated the generated `CT_TextLineBreak` coverage note.

Validation:

```text
go test ./internal/render -run 'TestTextRenderLinesPreserveDrawingMLBreakRuns|TestTextRenderLinesPreserveDrawingMLBreakRunMetrics|TestNormalAutofitMaxSoftLinesHonorsWrapNoneAndHardBreaks|TestFitNormalAutofitAllowsWrappingWithinHardBreakLines' -count=1 -v: passed
go test ./internal/render -run 'TestTextParagraphsFromNode|TestTextRenderLines|TestMeasureStyledSegments|Test.*Text.*Baseline|Test.*Text.*Spacing|Test.*Text.*Paragraph' -count=1: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect3-break-metrics PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/cumulative-shape-0001-4-Rectangle-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; visible crop still differs by 21,073 pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
```

Decision: accepted as source-semantics coverage for `CT_TextLineBreak` run
metrics. This does not close M12: the targeted `Rectangle 3` fixture, clean
object fixture suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Non-Visual Text Box Metadata

Source/schema anchors:

```text
pml.xsd:1201 CT_ShapeNonVisual
dml-main.xsd:800 CT_NonVisualDrawingShapeProps
pml.xsd:1209 CT_Shape
dml-main.xsd:2653 CT_TextBody
```

Source object:

- Deck: `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`
- Slide: 3
- Object: `sp` id `8`, name `TextBox 7`
- Fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json`

Rationale:

- The object authors `<p:cNvSpPr txBox="1"/>`, identifying the shape as a
  text box in the non-visual shape properties.
- The parser previously dropped this authored metadata. That left the semantic
  model, render primitive, and object-debug summary unable to prove that the
  source text-box flag was preserved.
- This is implementable static PresentationML metadata. It should be carried
  through the existing source object and primitive path while the fixture
  remains supported-scope text layout work.

Change:

- Added `slideElement.IsTextBox` parsing from `p:cNvSpPr@txBox`.
- Added `renderTextPrimitive.IsTextBox` so text primitive lowering preserves
  the flag.
- Added `resolved_style.text_box` to object-debug summaries.
- Updated the generated non-visual DrawingML coverage note for
  `CT_NonVisualDrawingShapeProps@txBox`.

Validation:

```text
go test ./internal/render -run 'TestCollectSlideElementsParsesNonVisualTextBoxFlag|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestAnchorCenteredTextBoundsCentersNarrowTextBox' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-txbox PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 7 still differs by 130,250 pixels
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
```

Decision: accepted as source-semantics coverage for non-visual text-box
metadata. The current object debug summary now records `"text_box": true`.
This does not close M12: the targeted `TextBox 7` fixture, clean object fixture
suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Non-Visual Lock Metadata

Source/schema anchors:

```text
dml-main.xsd:727 AG_Locking
dml-main.xsd:752 CT_PictureLocking
dml-main.xsd:800 CT_NonVisualDrawingShapeProps
dml-main.xsd:828 CT_NonVisualGraphicFrameProperties
pml.xsd:1245 CT_Picture
pml.xsd:1263 CT_GraphicalObjectFrame
```

Source objects:

- WHO HIV slide 009 `Picture 2` authors
  `<a:picLocks noChangeAspect="1"/>`.
- WHO HIV slide 012 `Table 3` authors
  `<a:graphicFrameLocks noGrp="1"/>`.
- Neighboring source fixtures also author `a:spLocks`, `a:picLocks`, and
  `a:graphicFrameLocks` flags.

Rationale:

- The generated matrix claimed non-visual lock metadata was preserved, but the
  semantic model and render primitives had no field carrying enabled lock
  attributes.
- These flags are static PresentationML/DrawingML metadata. They do not change
  the raster output directly, but they are part of the source object semantics
  and should survive parse/lowering/debug reporting.

Change:

- Added `slideElement.NonVisualLocks`, populated from local `spLocks`,
  `picLocks`, `cxnSpLocks`, `grpSpLocks`, `graphicFrameLocks`, and `cpLocks`
  children.
- Captured only enabled lock attributes and sorted them deterministically as
  strings such as `picLocks.noChangeAspect`.
- Preserved the lock list in picture, shape, connector, graphic-frame, and
  group render primitives.
- Added `resolved_style.non_visual_locks` to object-debug summaries.
- Updated the generated non-visual DrawingML coverage note to name enabled
  lock flags.

Validation:

```text
go test ./internal/render -run 'TestCollectSlideElementsParsesNonVisualTextBoxFlag|TestCollectSlideElementsParsesPictureLockFlags|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-picture2-locks PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Picture 2 still differs by 154,741 pixels and records ["picLocks.noChangeAspect"]
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-locks PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 still differs by 284,470 pixels and records ["graphicFrameLocks.noGrp"]
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-locks.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
```

Decision: accepted as source-semantics coverage for enabled non-visual lock
metadata. This does not close M12: the targeted fixtures, clean object fixture
suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 cNvPr Descriptive Metadata

Source/schema anchors:

```text
dml-main.xsd:788 CT_NonVisualDrawingProps
pml.xsd:1245 CT_Picture
```

Source objects:

- EPA Generate slide 007 `Picture 2` authors `adec:decorative val="1"` inside
  `p:cNvPr/a:extLst`.
- EPA Generate slide 003 `Picture 25` authors
  `descr="Diagram&#xA;&#xA;Description automatically generated"`.

Rationale:

- The generated matrix already described cNvPr descriptions as preserved, but
  the parser only copied cNvPr id/name into the semantic model.
- `descr`, `title`, explicit `hidden`, and Office decorative metadata are
  static source-object semantics. They should survive parse/lowering/debug
  reporting while the visual fixture residuals remain renderer work.

Change:

- Added cNvPr `descr`, `title`, `hidden`, and `adec:decorative@val` parsing
  to `slideElement`.
- Preserved cNvPr description/title in object-debug `PaintedObject` records,
  `resolved_style`, and render primitive provenance.
- Preserved explicit cNvPr boolean metadata as deterministic
  `non_visual_properties` strings such as `decorative=true` and
  `hidden=false`.
- Updated the generated non-visual DrawingML coverage note to name
  title/hidden/decorative preservation.

Validation:

```text
go test ./internal/render -run 'TestCollectSlideElementsParsesPictureLockFlags|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderObjectDebugRecordsArtifactsAndIsolationModes|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-cnvpr-decorative PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/0008-3-Picture-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Picture 2 still differs by 95,960 pixels and records ["decorative=true"]
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-cnvpr-description PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-003/micro-fixtures/0004-26-Picture-25/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Picture 25 still differs by 65,347 pixels and records the authored description
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-cnvpr.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
```

Decision: accepted as source-semantics coverage for cNvPr descriptive and
boolean metadata. This does not close M12: the targeted fixtures, clean object
fixture suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 cNvPr Creation ID Metadata

Source/schema anchors:

```text
dml-main.xsd:788 CT_NonVisualDrawingProps
pml.xsd:1263 CT_GraphicalObjectFrame
pml.xsd:1245 CT_Picture
pml.xsd:1242 CT_Shape
```

Source objects:

- WHO HIV slide 012 `Table 3` authors
  `a16:creationId id="{D32AD674-1F5F-084E-9B33-D94CDE5FD8BD}"` under
  `p:cNvPr/a:extLst`.
- WHO HIV slide 009 `Picture 2` authors
  `{5801FB1F-5610-3B4D-AF42-4F8881B7C4B0}`.
- WHO HIV slide 003 `TextBox 7` authors
  `{93C8E66B-89E0-1A42-98AC-3BF114210851}`.

Rationale:

- The current failing WHO objects carry Office creation IDs in cNvPr extension
  metadata, but parser/debug output only preserved id/name and the previously
  added descriptive, boolean, and lock metadata.
- Creation IDs are static source-object metadata. They do not change raster
  output directly, but they should survive parse/lowering/debug reporting.

Change:

- Added descendant `creationId@id` parsing to the shared cNvPr metadata path.
- Preserved the value as `slideElement.CreationID`.
- Added `cnv_pr_creation_id` to object-debug `PaintedObject` records.
- Added `creation_id` to object-debug `resolved_style` summaries and render
  primitive provenance.
- Updated the generated non-visual DrawingML coverage note to name creation ID
  preservation.

Validation:

```text
go test ./internal/render -run 'TestCollectSlideElementsParsesPictureLockFlags|TestRenderObjectDebugRecordsArtifactsAndIsolationModes|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-cnvpr-creationid-table3 PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 still differs by 284,470 pixels, fixture object preserves cnv_pr_creation_id="{D32AD674-1F5F-084E-9B33-D94CDE5FD8BD}", and current object summary records resolved_style.creation_id="{D32AD674-1F5F-084E-9B33-D94CDE5FD8BD}"
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-creationid.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
```

Decision: accepted as source-semantics coverage for cNvPr creation ID
metadata. This does not close M12: the targeted fixture, clean object fixture
suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Table Row And Column ID Metadata

Source/schema anchors:

```text
dml-main.xsd:2381 CT_TableGrid
dml-main.xsd:2398 CT_TableRow
dml-main.xsd:2423 CT_Table
pml.xsd:1263 CT_GraphicalObjectFrame
```

Source object:

- WHO HIV slide 012 `Table 3` authors six `a16:colId@val` values under
  `a:tblGrid/a:gridCol/a:extLst` and thirteen `a16:rowId@val` values under
  `a:tr/a:extLst`.

Rationale:

- The top clean table fixture carries Office table-grid and row identity
  metadata, but the table model only kept column widths and row heights.
- These IDs are static table-source metadata. They do not change raster output
  directly, but they should survive parse/lowering/debug reporting while table
  visual parity remains open.

Change:

- Added `tableModel.ColumnIDs` populated from descendant `colId@val` under
  each parsed `gridCol`.
- Added `tableRow.ID` populated from descendant `rowId@val`.
- Preserved column IDs and row IDs in table primitive lowering.
- Added `resolved_style.table_column_ids` and `resolved_style.table_row_ids`
  to object-debug summaries.
- Updated the generated table-grid and table-row coverage notes to name the
  preservation.

Validation:

```text
go test ./internal/render -run 'TestParseGraphicFrameReadsTableGrid|TestRenderSceneFromElementsLowersAllPrimitiveFamilies|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table-rowcol-id-table3 PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 still differs by 284,470 pixels while resolved_style.table_column_ids records six source IDs and resolved_style.table_row_ids records thirteen source IDs
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-rowcol-ids.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
```

Decision: accepted as source-semantics coverage for table row and column ID
metadata. This does not close M12: the targeted fixture, clean object fixture
suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Text Object Font Family Summaries

Source/schema anchors:

```text
dml-main.xsd:2814 CT_TextFont
dml-main.xsd:2873 CT_TextCharacterProperties
dml-main.xsd:2994 CT_TextParagraphProperties
pml.xsd:1209 CT_Shape
```

Source object:

- WHO HIV slide 003 `TextBox 7` authors Arial in `a:pPr/a:buFont` and in
  run-level `a:rPr/a:latin` / `a:rPr/a:cs`, while the object-debug summary
  previously reported inherited Calibri as the representative font family.

Rationale:

- The renderer already uses paragraph/run font families for text drawing, but
  object-debug summaries exposed only the shape fallback font. That made the
  current source object look less precisely preserved than it was and hid the
  authored run font-family evidence needed for M12 triage.

Change:

- Added `resolved_style.font_families` as an additive JSON/debug field.
- Changed `resolved_style.font_family` to prefer authored text run families,
  then paragraph/bullet defaults, and finally shape fallback font context.
- Updated the generated text coverage note to mention object-debug font-family
  summaries.

Validation:

```text
go test ./internal/render -run TestObjectStyleSummaryIncludesResolvedParagraphTextStyle -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-font-summary PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 7 still differs by 130,250 visible pixels while resolved_style.font_family records "Arial" and resolved_style.font_families records ["Arial","Calibri"]
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-text-font-summary.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
```

Decision: accepted as source-summary coverage for authored text font family
metadata. This does not close M12: the targeted fixture, clean object fixture
suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Text Body Property Summaries

Source/schema anchors:

```text
dml-main.xsd:2625 CT_TextBodyProperties
dml-main.xsd:2653 CT_TextBody
dml-main.xsd:2705 EG_TextAutofit
dml-main.xsd:2706 CT_TextNoAutofit
dml-main.xsd:2707 CT_TextNormalAutofit
dml-main.xsd:2711 CT_TextShapeAutofit
pml.xsd:1209 CT_Shape
```

Source object:

- WHO HIV slide 003 `TextBox 7` authors
  `<a:bodyPr wrap="square" rtlCol="0"><a:spAutoFit/></a:bodyPr>`.

Rationale:

- The parser and text primitives already carried source body-property values,
  but object-debug summaries only exposed `rtlCol` after the previous text-body
  pass.
- Wrap, autofit choice, normal-autofit scaling fields, and first/last
  paragraph spacing are static text-body source metadata. They should survive
  lowering and appear in object triage output while text visual parity remains
  open.

Change:

- Preserved `fontScale`, `lnSpcReduction`, and `spcFirstLastPara` values in
  lowered render text primitives.
- Extended `resolved_style.text_body_properties` to report `wrap`,
  shape/normal/no autofit, font scale, line-spacing reduction, first/last
  paragraph spacing, and `rtlCol`.
- Updated the generated text-body coverage note to name the additional M12
  preservation and summary evidence.

Validation:

```text
go test ./internal/render -run 'TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestParseBodyPropertiesReadsTextAnchor' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-body-summary PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; TextBox 7 still differs by 130,250 visible pixels while resolved_style.text_body_properties records wrap=square, spAutoFit=true, and rtlCol=false
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-text-body-summary.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
```

Decision: accepted as source-summary coverage for text body properties. This
does not close M12: the targeted `TextBox 7` fixture, clean object fixture
suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Table Style ID And Flag Summaries

Source/schema anchors:

```text
dml-main.xsd:2405 CT_TableProperties
dml-main.xsd:2423 CT_Table
pml.xsd:1263 CT_GraphicalObjectFrame
```

Source object:

- WHO HIV slide 012 `Table 3` authors
  `<a:tblPr firstRow="1" bandRow="1">` with table style ID
  `{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}`.

Rationale:

- The parser and graphic-frame primitive already carried the table style ID and
  table flags used by style resolution.
- Object-debug summaries exposed row/column IDs and table unsupported records
  but not the authored `tblPr` identity/flag inputs. This made Table 3 triage
  less precise even though those source fields drive table rendering.

Change:

- Added additive `resolved_style.table_style_id` and
  `resolved_style.table_properties` object-debug fields.
- Reported authored first/last row/column flags, band row/column flags, and
  direct table background presence.
- Updated the generated `CT_TableProperties` / `CT_Table` coverage notes.

Validation:

```text
go test ./internal/render -run TestObjectStyleSummaryIncludesResolvedParagraphTextStyle -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-style-summary PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Table 3 still differs by 284,470 pixels while resolved_style.table_style_id records "{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}" and resolved_style.table_properties records ["firstRow=true","bandRow=true"]
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-style-summary.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
```

Decision: accepted as source-summary coverage for table style identity and
authored table flags. This does not close M12: the targeted `Table 3` fixture,
clean object fixture suite, and exact Apple Notes gate still fail.

## 2026-06-02 - M12 Picture Source Media Summaries

Source/schema anchors:

```text
pml.xsd:1245 CT_Picture
dml-main.xsd:1477 CT_Blip
dml-main.xsd:1502 CT_BlipFillProperties
```

Source object:

- WHO HIV slide 009 `Picture 2` authors `a:blip r:embed="rId4"` under
  `p:pic/p:blipFill`, uses `a:stretch/a:fillRect`, has no authored crop,
  mask, or effect wrapper, and resolves in the micro-fixture to
  `ppt/media/object.png`, an `image/png` source decoded at 2830x820.

Rationale:

- The picture backend already resolves the relationship target and decodes the
  source image before rendering.
- Object-debug summaries exposed embed/fill-mode information but not the
  resolved media part, content type, or decoded source dimensions. That made
  large picture residuals harder to triage and did not prove any impossible
  renderer boundary.

Change:

- Stored resolved picture media part, content type, and decoded intrinsic size
  on the slide element after `pictureSourceImage` succeeds.
- Extended `resolved_style.image` object-debug summaries to include
  `part=...`, `type=...`, and `size=...`.
- Updated generated `CT_Blip` and `CT_BlipFillProperties` coverage notes.

Validation:

```text
go test ./internal/render -run 'TestObjectStyleSummaryIncludesImageAndTableProperties|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields' -count=1 -v: passed
PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-picture2-source-media PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v: expected failure; Picture 2 still differs by 154,741 pixels while resolved_style.image records "embed=rId4 part=ppt/media/object.png type=image/png size=2830x820", resolved_style.image_effects records ["fillMode=stretch"], and resolved_style.image_unsupported is null
python3 tools/generate_ooxml_drawingml_audit.py --print-summary: passed; queue totals core-static=16, common-partial=389, hard-rendering=458, unsupported-preserve=16, out-of-scope=128
go test ./internal/render -count=1: passed
PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-picture-source-media.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v: passed in expected-failure accounting mode; total=59 passed=0 failed=59; top failures remain Table 3=284,470, Picture 2=154,741, TextBox 7=130,250, Google Shape;179;p9=127,167, EPA Picture 2=95,960
```

Decision: accepted as source-summary coverage for resolved picture source
media. This does not close M12: the targeted `Picture 2` fixture, clean object
fixture suite, and exact Apple Notes gate still fail.

Resume checkpoint: the interrupted closeout still needs `go test ./... -count=1`,
`git diff --check`, confirmation that `git diff --cached --name-only` is empty,
and the active wording scan from the prior closeout instructions.
