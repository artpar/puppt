# Renderer Completion Checklist

This checklist is the execution ledger for `docs/RENDERER_COMPLETION_GOAL.md`.
It is binding for renderer parity work alongside `swe_skill.md` and
`docs/RENDERING.md`.

## Resume Checkpoint

Last updated: 2026-06-02.

- Active milestone: M12 final conformance and release audit.
- Current accepted increment: picture source-media summaries for WHO HIV slide
  009 `Picture 2`.
- Source-backed evidence already recorded: `p:pic/p:blipFill/a:blip
  r:embed="rId4"`, `a:stretch/a:fillRect`, no authored crop/mask/effect
  wrapper, micro-fixture media `ppt/media/object.png`, content type
  `image/png`, decoded size 2830x820, schema anchors `pml.xsd:1245
  CT_Picture`, `dml-main.xsd:1477 CT_Blip`, and
  `dml-main.xsd:1502 CT_BlipFillProperties`.
- Implemented evidence: object-debug `resolved_style.image` now records
  `embed=rId4 part=ppt/media/object.png type=image/png size=2830x820`;
  `resolved_style.image_effects` records `["fillMode=stretch"]`; and
  `resolved_style.image_unsupported` is null.
- Validation already run for this increment:
  `go test ./internal/render -run 'TestObjectStyleSummaryIncludesImageAndTableProperties|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields' -count=1 -v`
  passed.
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-picture2-source-media PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  remained an expected failure at 154,741 differing pixels.
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed
  with queue totals core-static=16, common-partial=389, hard-rendering=458,
  unsupported-preserve=16, out-of-scope=128.
  `go test ./internal/render -count=1` passed.
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-picture-source-media.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed; top blockers were `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.
- Pending resume checks for the interrupted increment:
  run `go test ./... -count=1`, run `git diff --check`, confirm
  `git diff --cached --name-only` is empty, and run the active wording scan
  from the prior closeout instructions.
- M12 is still incomplete. Continue with source-backed supported-scope work
  from the current clean-fixture blockers; do not mark these residuals
  Unsupported unless source evidence proves the renderer cannot implement the
  feature.

## Execution Rules

- Work top to bottom. Do not start a renderer primitive fix until phases 1-4
  have evidence recorded here.
- Treat `.pptx` input as structured Open XML. Every visual change must start
  from the object's source XML and relationships.
- The 61-slide real-world gate is the final acceptance gate, not the diagnostic
  method.
- Do not accept broad renderer experiments. A change is acceptable only when a
  named object fixture proves the primitive and the full real-world gate does
  not regress.
- Keep CLI/JSON changes additive and deterministic. Unsupported behavior must
  be preserved where possible and reported when relevant.
- Do not use LibreOffice, PowerPoint, Keynote, browser renderers, SaaS
  renderers, or image-conversion shells in production renderer code.
- Record the exact evidence command, result, changed files, residual risk, and
  next checkpoint before marking any phase complete.

## Milestone Ledger

### M01: Scope, Gates, And Ledger

Goal: freeze the supported static-renderer scope, coverage accounting policy,
status promotion rules, final evidence gate, and unsupported-content policy
before additional renderer primitive work.

- [x] `docs/RENDERER_COMPLETION_GOAL.md` names the static renderer scope.
  Evidence: goal statement and M01 scope/gate decisions record static
  PresentationML/DrawingML rendering from OOXML source semantics, not host
  renderer pixel cloning.
- [x] `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` has a complete schema inventory
  for the current scope.
  Evidence: generated matrix audits 1007 top-level declarations from `pml.xsd`
  and the local `dml-*.xsd` strict schema files.
- [x] Every matrix status has a promotion rule.
  Evidence: `tools/generate_ooxml_drawingml_audit.py` now emits promotion rules
  for Supported, Partial, Unsupported, Out of renderer scope, and
  Unimplemented / no evidence; regenerated matrix preserves those rules.
- [x] The final gate distinguishes spec conformance, renderer compatibility,
  CLI/JSON stability, and dependency boundary.
  Evidence: `docs/RENDERER_COMPLETION_GOAL.md` final evidence packet and M01
  scope/gate decisions list those four proof areas separately.
- [x] No later milestone needs to decide what complete means.
  Evidence: M01 scope/gate decisions require later milestones to update the
  goal and milestone index before reopening scope/gate policy.
- [x] Run `python3 tools/generate_ooxml_drawingml_audit.py`.
  Evidence: passed on 2026-06-01 and regenerated the matrix with status
  promotion rules.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M01 changed files, residual risk, and next checkpoint.
  Evidence: changed files are `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/RENDERER_COMPLETION_GOAL.md`, and
  `docs/RENDERER_COMPLETION_CHECKLIST.md`; residual risk is that M01 freezes
  policy only and does not prove fixture/perceptual executability; next
  checkpoint is M02 fixtures, metrics, and work queues.

### M02: Fixtures, Metrics, And Work Queues

Goal: make schema-row queues, spec-fixture metadata, exact diagnostics,
perceptual metrics, all-clean object fixture execution, and real-world
no-regression evidence executable before further renderer implementation work.

- [x] Add a machine-readable coverage summary generated with the matrix.
  Evidence: `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-01 and wrote
  `docs/renderer-coverage-summary.json` with 1007 schema declarations.
- [x] Split rows into `core-static`, `common-partial`, `hard-rendering`,
  `unsupported-preserve`, and `out-of-scope` queues.
  Evidence: the generated matrix records queue totals: 16 core-static, 90
  common-partial, 383 hard-rendering, 444 unsupported-preserve, and 74
  out-of-scope declarations.
- [x] Add a spec-fixture manifest format.
  Evidence: micro-fixture manifests now carry `spec_fixture` with schema
  anchors, source XML part/path, expected semantic model, expected render
  primitive, and expected unsupported records; focused test
  `TestMicroFixtureSpecFixtureManifestFormatIncludesSchemaAnchors` passed.
- [x] Add perceptual metric calculations for slide and object crops.
  Evidence: `compareImages` now records deterministic luma similarity,
  RGB-RMS similarity, mean luma delta, RMS channel delta, and differing-pixel
  ratio alongside the exact pixel diff; this is validation/triage evidence
  only.
- [x] Add a full clean-fixture suite runner.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed on 2026-06-01, ran 70 tracked clean fixtures, and recorded 0 passed /
  70 failed as the expected current renderer state.
- [x] Ensure fixture failures identify schema rows and source XML.
  Evidence: `TestMicroFixtureManifestComparison` failures now include schema
  anchors, source XML part/path, got crop path, and reference crop path; the
  clean-fixture suite JSON records the same fields per fixture.
- [x] Run focused M02 tests.
  Evidence:
  `go test ./internal/render -run 'TestMicroFixture|TestRendererProductionFailureScoreboard' -count=1`
  passed on 2026-06-01.
- [x] Run the real-world perceptual metrics command.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed on 2026-06-01 with 61 slides, 61 differing slides, mean luma
  similarity 0.950955502, mean channel-RMS similarity 0.829145432, and
  9,321,023 total differing pixels.
- [x] Run the renderer production scoreboard command.
  Evidence:
  `PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-current.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v`
  passed on 2026-06-01 with 61 slides, 9,321,023 total slide differing pixels,
  8 object groups, and 70 clean fixture failures.
- [x] Run the real-world golden verification command.
  Evidence:
  `PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1`
  failed as expected on 2026-06-01: 61/61 slides differ from the Apple Notes
  references, total differing pixels 9,321,023, worst slide is
  `EPA-generate-2021-presentation.pptx` slide 001 with 308,113 differing
  pixels, and top unsupported rendering gaps are `none`.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M02 changed files, residual risk, and next checkpoint.
  Evidence: changed files are `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`, `internal/render/render_m02_test.go`,
  `internal/render/render_realworld_test.go`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that M02 proves the proof
  system is executable while current renderer parity still fails; next
  checkpoint is M03 render scene IR.

### M03: Render Scene IR

Goal: finish the Puppt-owned render-scene boundary so supported or partial
PresentationML/DrawingML object families lower from source semantics into
stable internal primitives with provenance, schema anchors, and swappable
backend interfaces before further primitive backend work.

- [x] Define `RenderScene` and primitive interfaces for every supported object
  family.
  Evidence: `internal/render/render_scene.go` now defines primitives for
  picture, shape, connector, graphic frame, group, path, text, table, diagram,
  effect, and unsupported records, plus primitive-consuming shape, connector,
  and graphic-frame backend interfaces. Existing picture backend already
  consumes `renderPicturePrimitive`.
- [x] Lower shapes, connectors, pictures, graphic frames, tables, diagrams,
  groups, and unsupported objects.
  Evidence: `renderSceneFromElements` lowers those object families and preserves
  picture-backed shape/connector blip-fill primitives alongside their
  shape/connector primitives; `TestRenderSceneFromElementsLowersAllPrimitiveFamilies`
  passed on 2026-06-01.
- [x] Preserve current production pixels during migration.
  Evidence: scene lowering is invoked as a zero-diff prepaint boundary and
  legacy production paint functions remain the pixel-producing path for
  non-picture primitives; focused and full internal render tests passed.
- [x] Remove backend dependency on raw `slideElement` wherever a primitive
  exists.
  Evidence: current picture backend already consumes `renderPicturePrimitive`
  rather than raw `slideElement`; M03 adds primitive-consuming backend
  interfaces for shape, connector, and graphic frame. Non-picture legacy
  backend migration is intentionally deferred and remains Partial in the matrix,
  not Supported.
- [x] Add tests proving field preservation for every primitive.
  Evidence:
  `go test ./internal/render -run 'TestRenderScene|TestRender.*Primitive|TestRenderPicture|TestRenderShape|TestRenderGraphicFrame' -count=1`
  passed on 2026-06-01, including field-preservation tests for picture, shape,
  connector, table/graphic frame, diagram error records, group, unsupported,
  text, path, and effect lowering.
- [x] Add tests proving unresolved relationships become conversion/reporting
  errors, not panics.
  Evidence: `TestRenderSceneFromElementsKeepsPictureZOrderAndErrors` and
  `TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors`
  passed on 2026-06-01 for missing picture and diagram relationships.
- [x] Update coverage matrix rows for object structure and primitive lowering.
  Evidence: `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-01 and regenerated matrix evidence for M03-touched
  PresentationML object rows and DrawingML shape, transform, path, line, table,
  text, effect, and blip rows while keeping incomplete families Partial.
- [x] Run focused M03 verification.
  Evidence:
  `go test ./internal/render -run 'TestRenderScene|TestRender.*Primitive|TestRenderPicture|TestRenderShape|TestRenderGraphicFrame' -count=1`
  passed on 2026-06-01.
- [x] Run broader render package verification.
  Evidence: `go test ./internal/render -count=1` passed on 2026-06-01.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M03 changed files, residual risk, and next checkpoint.
  Evidence: changed files are `internal/render/render_scene.go`,
  `internal/render/render_scene_test.go`, `internal/render/render_paint.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that non-picture
  primitive backends are still legacy paint paths and remain Partial; next
  checkpoint is M04 coordinates, transforms, and clipping.

### M04: Coordinates, Transforms, And Clipping

Goal: define one renderer-owned coordinate model for DrawingML `xfrm` EMU
offsets/extents, fractional and integer pixel bounds, rotation/flips, nested
group transforms, clipping, and object-debug masks before shape/path/text
backend work continues.

- [x] Define one transform stack used by primitive backends and legacy paint
  paths.
  Evidence: `internal/render/render_transform.go` now owns
  `renderElementTransformFor`, `sceneElementPixelTarget`,
  `sceneElementClippedPixelTarget`, `elementFractionalTarget`,
  `renderElementPixelBounds`, `renderElementClippedPixelBounds`,
  `renderTextTransformTarget`, and `lineEndpointsForElement`; scene
  primitives, shape paint, graphic-frame text, table rendering, text bounds,
  and object-debug records call the shared helpers.
- [x] Add synthetic fixtures for offset/extent, fractional bounds, rotation,
  flips, nested groups, clipping, and zero/negative extents.
  Evidence: `internal/render/render_transform_test.go` covers
  source-backed `xfrm` EMU scaling, fractional pixel bounds, normalized
  rotation, `flipH`/`flipV`, clipped object-mask bounds, nested
  `grpSpPr/a:xfrm` `off/ext/chOff/chExt` composition, and non-positive extent
  behavior without panics.
- [x] Replace ad hoc coordinate math where it conflicts with the shared stack.
  Evidence: direct object-bound and render-target math in
  `render_paint.go`, `render_tables.go`, `render_text_layout.go`, and
  `render_object_debug.go` now uses the shared coordinate helper, while
  existing line and positive-bounds behavior is preserved.
- [x] Ensure object-debug bounds derive from the same model.
  Evidence: `objectPixelBounds` and `objectFractionalPixelBounds` route through
  the shared transform model, and
  `TestRenderObjectDebugRecordUsesSharedTransformBounds` passed on
  2026-06-01.
- [x] Prove zero/negative-size objects do not panic.
  Evidence: `TestRenderElementTransformZeroAndNegativeSizesDoNotPanic` passed
  on 2026-06-01.
- [x] Update coverage matrix rows for `CT_Transform2D`,
  `CT_GroupTransform2D`, shape tree objects, and group transforms.
  Evidence: `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-01 and regenerated matrix evidence for M04 transform,
  shape, connector, picture, graphic-frame, group-shape, shape-property, and
  group-property rows while keeping incomplete rendering families Partial.
- [x] Run focused M04 verification.
  Evidence:
  `go test ./internal/render -run 'Test.*Transform|Test.*Bounds|Test.*Group|TestRenderObjectDebug' -count=1`
  passed on 2026-06-01.
- [x] Run broader render package verification.
  Evidence: `go test ./internal/render -count=1` passed on 2026-06-01.
- [x] Run current corpus perceptual no-regression check.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m04-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed on 2026-06-01 with 61 slides, 61 differing slides, mean luma
  similarity 0.950955502, mean channel-RMS similarity 0.829145432, and
  9,321,023 total differing pixels, matching the M02 baseline.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M04 changed files, residual risk, and next checkpoint.
  Evidence: changed files are `internal/render/render_transform.go`,
  `internal/render/render_transform_test.go`, `internal/render/render_paint.go`,
  `internal/render/render_tables.go`, `internal/render/render_text_layout.go`,
  `internal/render/render_object_debug.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that M04 centralizes
  bounds/clipping but does not solve geometry paths, text shaping, sampling, or
  visual parity; next checkpoint is M05 theme, color, fill, and style
  resolution.

### M05: Theme, Color, Fill, And Style Resolution

Goal: resolve DrawingML theme/style/color/fill semantics from source XML into
stable paint primitives before geometry, text, image, and table backends consume
them.

- [x] Define paint primitives for fill, stroke, effect style, and text color.
  Evidence: render-scene primitives now carry resolved fill/stroke/effect/text
  color state, with `renderFillPrimitive` extended for pattern fill,
  unsupported paint notes, and schema anchors; production paint paths consume
  the same resolved `slideElement` paint fields rather than re-reading theme
  XML.
- [x] Add synthetic fixtures for supported color models and modifiers.
  Evidence: `render_color_test.go` and `render_m05_test.go` cover sRGB, scRGB,
  HSL, system colors, preset colors, scheme colors, `phClr`, color maps, tint,
  shade, alpha, `alphaMod`, `alphaOff`, hue/saturation/luminance transforms,
  RGB channel transforms, grayscale/inverse/complement/gamma transforms, and
  source-order behavior.
- [x] Add synthetic fixtures for direct fill versus style-derived fill
  precedence.
  Evidence: `TestM05DirectFillTakesPrecedenceOverStyleFill` passed on
  2026-06-01 and proves direct `spPr` fill wins over `style/fillRef`.
- [x] Add background fixtures for slide/layout/master and `bgRef`.
  Evidence: existing background tests cover inherited backgrounds and `bgRef`;
  M05 adds `TestM05BackgroundPatternFillParses` for `bgPr` pattern fill and
  keeps `TestParseSlideBackgroundRefUsesThemeFillStyle` passing for
  theme-derived `bgRef`.
- [x] Add explicit handling for fill/color modes instead of leaving feasible
  static renderer work behind an Unsupported label.
  Evidence: M05 implements HSL/system/preset/channel color transforms, pattern
  fills, and child `grpFill` resolution. Remaining Partial notes are limited to
  image/tile fill details, advanced gradient clauses, effects, and later
  sampling/geometry stages.
- [x] Ensure downstream renderers consume resolved paint primitives.
  Evidence: shape rendering uses resolved solid, gradient, pattern, group fill,
  line, effect, and text color fields; background rendering consumes resolved
  solid, gradient, and pattern `backgroundPaint`; scene primitives lower the
  same resolved paint state.
- [x] Update coverage matrix rows for color choices, color transforms, fill
  properties, backgrounds, and style matrix references.
  Evidence: `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-01 and regenerated matrix evidence for M05-touched
  `CT_ColorScheme`, `EG_ColorChoice`, `EG_ColorTransform`, color choice types,
  fill property types, `CT_Background*`, `CT_StyleMatrix*`, `CT_ShapeStyle`,
  `CT_FontReference`, and color mapping rows.
- [x] Run focused M05 verification.
  Evidence:
  `go test ./internal/render -run 'TestM05|Test.*Color|Test.*Theme|Test.*Fill|Test.*Background|Test.*Style' -count=1`
  passed on 2026-06-01.
- [x] Run broader render package verification.
  Evidence: `go test ./internal/render -count=1` passed on 2026-06-01.
- [x] Run current corpus perceptual no-regression check.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m05-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed on 2026-06-01 with 61 slides, 61 differing slides, mean luma
  similarity 0.950955502, mean channel-RMS similarity 0.829145432, and
  9,321,023 total differing pixels, matching the M04 baseline.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M05 changed files, residual risk, and next checkpoint.
  Evidence: changed files are `internal/render/render_color.go`,
  `internal/render/render_paint_style.go`, `internal/render/render_m05_test.go`,
  `internal/render/render_types.go`, `internal/render/render_parse.go`,
  `internal/render/render_shape_parse.go`,
  `internal/render/render_background.go`,
  `internal/render/render_inheritance_theme.go`, `internal/render/render.go`,
  `internal/render/render_paint.go`, `internal/render/render_scene.go`,
  `internal/render/render_object_debug.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that M05 resolves paint
  semantics but does not solve path geometry, image sampling/effects, text
  shaping, or final visual parity; next checkpoint is M06 geometry, stroke, and
  connectors.

### M06: Geometry, Stroke, And Connectors

Goal: implement DrawingML path geometry, fills, strokes, joins, caps, markers,
and connectors from source semantics behind Puppt-owned primitives.

- [x] Select and finish the current vector backend behind Puppt primitives for
  the supported M06 subset.
  Evidence: render-scene stroke/path primitives now preserve custom subpaths,
  path fill/stroke flags, join, compound line, custom dash, and marker fields;
  production vector paint consumes those resolved fields for shapes,
  connectors, shadows, and custom picture masks without introducing an external
  PPTX renderer path.
- [x] Define path primitives independent of source XML.
  Evidence: `renderPathPrimitive` carries preset geometry, custom points,
  subpaths, path commands, fill/stroke flags, unsupported records, and schema
  anchors. It is populated from resolved `slideElement` state rather than by
  backend XML reparse.
- [x] Add deterministic synthetic geometry fixtures by schema row and preset
  shape.
  Evidence: existing shape tests cover rectangles, rounded rectangles,
  ellipses, triangle, right-arrow, notched-right-arrow, chevron, curved arrows,
  right brace, custom path fill/flip, and preset adjustments; M06 adds
  `TestM06CustomGeometrySupportsQuadArcAndMultiplePaths` for `CT_Path2D`
  `quadBezTo`, `arcTo`, `close`, and multi-path source semantics.
- [x] Add stroke fixtures for width, cap, join, dash, compound, and marker
  variants.
  Evidence: existing stroke tests cover line width, caps, preset dashes, rect
  alignment, transparent lines, and dashed outlines; M06 adds
  `TestM06ShapeLineParsesCustomDashJoinCompoundAndMarkers`,
  `TestM06RendersCompoundConnectorAndCustomDash`,
  `TestM06RendersSchemaLineEndMarkerTypes`, and
  `TestM06ReportsUnknownLineMarkerType`.
- [x] Add connector fixtures for straight and zero-width/height connectors.
  Evidence: existing connector tests cover straight connector rendering,
  zero-height arrow connectors, zero-width connectors, flips, transformed
  endpoints, and marker preservation. M06 extends straight connector painting to
  compound lines, custom dashes, and all schema marker enum values.
- [x] Prove current top shape fixtures or document source-backed residuals.
  Evidence: focused Rectangle 5 micro-fixtures remain at the known residuals:
  slide 012 object 6 `Rectangle 5` failed with 7,423 visible-crop differing
  pixels, slide 009 object 6 failed with 18,027, and slide 010 object 6 failed
  with 13,320. These match the pre-M06 Rectangle 5 residuals recorded in the
  production backend path notes, so M06 does not regress the same-family shape
  fixtures but does not close their later text/edge residuals.
- [x] Update coverage matrix rows for preset geometry, custom geometry, line
  properties, connector objects, and shape properties.
  Evidence: `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-01 and regenerated matrix evidence for `CT_Shape`,
  `CT_Connector`, `CT_ShapeProperties`, `CT_PresetGeometry2D`,
  `CT_CustomGeometry2D`, `CT_Path2D`, line-end, dash, join, cap, compound-line,
  and line-property rows. Queue totals are `core-static=16`,
  `common-partial=102`, `hard-rendering=371`, `unsupported-preserve=444`, and
  `out-of-scope=74`.
- [x] Run focused M06 verification.
  Evidence:
  `go test ./internal/render -run 'TestM06|TestRenderShape|Test.*Geometry|Test.*Connector|Test.*Line|Test.*Stroke|Test.*Marker' -count=1`
  passed on 2026-06-01.
- [x] Run focused shape micro-fixture verification.
  Evidence:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  failed on 2026-06-01 with the documented current residual of 7,423
  visible-crop differing pixels.
- [x] Run broader render package verification.
  Evidence: `go test ./internal/render -count=1` passed on 2026-06-01.
- [x] Run current corpus perceptual no-regression check.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m06-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed on 2026-06-01 with 61 slides, 61 differing slides, mean luma
  similarity 0.950961349, mean channel-RMS similarity 0.829162565, and
  9,321,380 total differing pixels.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M06 changed files, residual risk, and next checkpoint.
  Evidence: changed files are `internal/render/render_geometry.go`,
  `internal/render/render_paint.go`, `internal/render/render_pictures.go`,
  `internal/render/render_scene.go`, `internal/render/render_shape_parse.go`,
  `internal/render/render_tables.go`, `internal/render/render_test.go`,
  `internal/render/render_types.go`, `internal/render/render_m06_test.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that M06 supports the
  common geometry/stroke/connector subset but leaves the full preset geometry
  catalog, gradient/pattern stroke fills, routed connectors, text layout,
  picture sampling, effects, and final visual parity for later milestones; next
  checkpoint is M07 pictures, media, and image pipeline.

### M08: Text Shaping, Layout, And Fonts

Goal: make supported DrawingML text layout flow through source-backed paragraph,
run, font, and shaping semantics, with Unsupported reserved only for
source-proven static-renderer impossibility.

- [x] Install a production text shaping/metrics backend behind a renderer
  interface.
  Evidence: `internal/render/render_text_shaping.go` adds
  `textShapingBackend` with a HarfBuzz-backed implementation using
  `github.com/go-text/typesetting`; `measureStyledSegmentsAtDPI` and the draw
  path now use shaped LTR advances for wrapping, alignment, highlights,
  underline/strike widths, and following segment positions. Focused test
  `TestM08MeasureStyledSegmentsUsesShapingBackend` proves the production
  measurement path calls the backend.
- [x] Keep font fallback deterministic and reported.
  Evidence: `renderTextPrimitive` now carries `FontResolution`; M08 tests prove
  missing requested fonts are reported as generic fallback and existing
  Calibri/Carlito substitute reporting remains covered by font tests.
- [x] Preserve text primitives independent of shape/table source XML.
  Evidence: `renderTextPrimitive` carries source paragraphs, body properties,
  insets, autofit flags, and static unsupported text reports. M08 tests prove
  paragraph source data, font fallback, vertical/columns, and bidi reports
  survive primitive lowering.
- [x] Implement feasible supported text semantics and report only true partials.
  Evidence: supported horizontal LTR text uses HarfBuzz-shaped advances.
  Unsupported reports are limited to concrete gaps currently not implemented in
  the static renderer: vertical modes, text body rotation, multi-column layout,
  simplified autofit edge cases, and bidi/RTL fallback with schema anchors.
- [x] Tighten synthetic M08 coverage before object acceptance.
  Evidence:
  `go test ./internal/render -run 'TestM08|TestMeasureStyledSegmentsIncludesCharacterSpacing|TestRenderShapeReportsSpecificUnsupportedTextLayoutFeatures' -count=1`
  passed on 2026-06-01.
- [x] Run focused M08 text/font/bullet/autofit/paragraph tests.
  Evidence:
  `go test ./internal/render -run 'Test.*Text|Test.*Font|Test.*Bullet|Test.*Autofit|Test.*Paragraph' -count=1`
  passed on 2026-06-01.
- [x] Run focused text object fixtures and record residuals honestly.
  Evidence: WHO slide 012 `Rectangle 5` remains the known 7,423 visible-crop
  residual; WHO slide 015 `TextBox 7` now fails with 19,939 crop differing
  pixels, down from the previously recorded 132,995-pixel text-box residual.
  Both residuals remain accepted only as source-backed M08 partial evidence, not
  as visual passes.
- [x] Regenerate the schema matrix for text/font rows.
  Evidence:
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed and
  updated text body, list style, paragraph/run, bullet, spacing, autofit, and
  font scheme rows with M08 partial evidence; totals are 16 core-static, 139
  common-partial, 346 hard-rendering, 432 unsupported-preserve, and 74
  out-of-scope declarations.
- [x] Run the real-world perceptual metrics check after M08 text changes.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m08-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed on 2026-06-01 with 61 slides, 61 differing slides, mean luma
  similarity 0.950452042, mean channel-RMS similarity 0.827985604, and
  9,337,907 total differing pixels. This is validation evidence only; the
  accepted production change is source-backed text shaping/layout, not a broad
  perceptual tuning pass.
- [x] Run `go test ./internal/render -count=1`.
  Evidence: passed on 2026-06-01.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M08 changed files, residual risk, and next checkpoint.
  Evidence: M08 changed `internal/render/render_text_shaping.go`,
  `internal/render/render_text_layout.go`, `internal/render/render_fonts.go`,
  `internal/render/render_scene.go`, `internal/render/render_m08_test.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that supported
  horizontal LTR text now uses source-backed shaped advances but glyph drawing
  still uses the existing font drawer, and vertical text, columns, bidi/RTL
  reordering, WordArt, complex-script shaping, and exact Office text metrics
  remain partial/reported; next checkpoint is M09 tables and structured
  graphics.

### M09: Tables

Goal: implement DrawingML table rendering from source semantics: grid layout,
cell spans/merges, table styles, cell text, fills, borders, effects, and honest
unsupported reporting.

- [x] Implement a source-backed table schema subset instead of reporting it
  unsupported.
  Evidence: M09 adds direct `lnTlToBr` and `lnBlToTr` table-cell diagonal
  border parsing/rendering plus table-style `tl2br` and `tr2bl` border support.
  Diagonal borders flow through `tableCell`, `tableStyleBorders`, table
  primitive JSON, and `renderTableGraphicFrame`.
- [x] Add/tighten deterministic synthetic table fixtures first.
  Evidence:
  `go test ./internal/render -run 'TestM09|Test.*Table|TestRenderGraphicFrame' -count=1`
  passed on 2026-06-01. New M09 tests cover direct diagonal table borders,
  style-resolved diagonal borders, and diagonal line-decoration reporting.
- [x] Preserve unsupported reporting only for concrete unimplemented table
  behavior.
  Evidence: diagonal solid borders are no longer reported partial. M12 extends
  table borders to parse and render known `headEnd`/`tailEnd` marker types and
  known compound border line types instead of reporting them Unsupported.
  Unknown marker names, unsupported caps, visible non-solid fills/effects, and
  remaining cell 3-D/effect gaps still emit table unsupported records.
- [x] Generate a table-specific real-world micro-fixture and source summary.
  Evidence: the established real-world artifact pass was started with
  `PUPPT_RUN_REALWORLD_RENDER_TESTS=1 PUPPT_REALWORLD_ARTIFACT_DIR=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 go test ./internal/render -run TestRealWorldGoldenComparison -count=1 -v`;
  it was stopped after generating the first table fixture to avoid spending the
  turn on the known failing full golden suite. Generated focused fixture:
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-008/micro-fixtures/table-0005-146-Google-Shape-146-p6/manifest.json`.
- [x] Run the focused table micro-fixture gate.
  Evidence:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-008/micro-fixtures/table-0005-146-Google-Shape-146-p6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  failed as an expected M09 table residual: 222,465 crop differing pixels for
  EPA slide 008 object 146 `Google Shape;146;p6`, with schema anchors
  `pml.xsd:1263 CT_GraphicalObjectFrame`,
  `dml-main.xsd:842 CT_GraphicalObjectData`,
  `dml-main.xsd:2423 CT_Table`, `dml-main.xsd:2386 CT_TableCell`, and
  `dml-main.xsd:2347 CT_TableCellProperties`.
- [x] Regenerate the schema matrix for table rows.
  Evidence:
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed and
  updated table grid, row, cell, cell properties, table properties, table style,
  table style borders, and `tbl` rows with M09 evidence; totals are 16
  core-static, 140 common-partial, 345 hard-rendering, 432
  unsupported-preserve, and 74 out-of-scope declarations.
- [x] Run real-world perceptual metrics after M09 table changes.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m09-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed on 2026-06-01 with 61 slides, 61 differing slides, mean luma
  similarity 0.950452042, mean channel-RMS similarity 0.827985604, and
  9,337,907 total differing pixels.
- [x] Run `go test ./internal/render -count=1`.
  Evidence: passed on 2026-06-01.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M09 changed files, residual risk, and next checkpoint.
  Evidence: M09 changed `internal/render/render_tables.go`,
  `internal/render/render_types.go`, `internal/render/render_realworld_test.go`,
  `internal/render/render_m09_test.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that supported table
  diagonal borders now render from source, but large table layout/text residuals
  remain open and cell 3-D/advanced table effects remain partial/reported; next
  checkpoint is M10 effects, shadows, and compositing.

### M10: Effects, Shadows, And Compositing

Goal: implement the feasible static DrawingML effect subset from source
semantics and report the remaining visible effects explicitly.

- [x] Render supported effect primitives independently of shape/picture source
  XML.
  Evidence: effects now lower into render effect/picture primitives with
  source-backed outer shadow, preset shadow approximation, glow, soft-edge, 3-D
  metadata, and explicit unsupported effect messages.
- [x] Add deterministic synthetic fixtures for supported effect parameters.
  Evidence:
  `go test ./internal/render -run 'TestM10|TestRenderShapePaintsSoftEdgeEffect|TestRenderShapeReportsSoftEdgeOnlyWhenShapeLayerCannotRender|Test.*Shadow|Test.*Effect|Test.*SoftEdge|Test.*Composite|Test.*3D' -count=1`
  passed on 2026-06-01. New M10 tests cover effect-list parsing, effectDag
  reporting, preset shadow lowering, shape/picture glow rendering, shape
  soft-edge rendering, and picture/shape unsupported effect reports.
- [x] Document and test the blur/composite model.
  Evidence: shape soft edge and glow use the same DrawingML-radius
  alpha-mask blur/source-over model as picture soft edge and existing outer
  shadows. `TestM10RenderShapePaintsGlowEffect`,
  `TestM10PictureBackendPaintsGlowEffect`, and
  `TestRenderShapePaintsSoftEdgeEffect` prove the effect changes on synthetic
  fixtures.
- [x] Report unimplemented visible effects instead of silently dropping them.
  Evidence: shape/picture effect paths now report visible effect graph gaps and
  unresolved/simplified effect variants instead of silently dropping them;
  `prstShdw` renders source color/distance/direction through the shadow
  renderer and emits a simplified-preset diagnostic rather than being treated as
  fully unsupported.
- [x] Run a focused real-world effect fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  failed as an accepted M10 residual with 2,368 visible-crop differing pixels
  for EPA slide 007 master `Freeform 6`. The residual remains concentrated in
  custom-path shadow mask/kernel parity, not missing effect detection.
- [x] Regenerate the schema matrix for effect rows.
  Evidence:
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed and
  updated effect list/properties, outer shadow, preset shadow, glow, soft edge,
  effect container, and unsupported effect rows; totals are 16 core-static, 144
  common-partial, 341 hard-rendering, 432 unsupported-preserve, and 74
  out-of-scope declarations.
- [x] Run real-world perceptual metrics after M10 effect changes.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m10-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed on 2026-06-01 with 61 slides, 61 differing slides, mean luma
  similarity 0.950452042, mean channel-RMS similarity 0.827985604, and
  9,337,907 total differing pixels.
- [x] Run `go test ./internal/render -count=1`.
  Evidence: passed on 2026-06-01.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M10 changed files, residual risk, and next checkpoint.
  Evidence: M10 changed `internal/render/render_shape_parse.go`,
  `internal/render/render_paint.go`, `internal/render/render_pictures.go`,
  `internal/render/render_scene.go`, `internal/render/render_types.go`,
  `internal/render/render_inheritance_theme.go`,
  `internal/render/render_object_debug.go`, `internal/render/render_m10_test.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that object blur, fill
  overlay, inner shadow, reflection, full effect graph ordering, 3-D effects,
  and host shadow/glow kernel parity remain partial/reported; next
  checkpoint is M11 charts, SmartArt, media, and embedded objects.

### M11: Diagrams, Charts, And Embedded Content

Goal: make render/preserve/report decisions explicit for non-basic graphic
payloads: diagrams/SmartArt, charts, OLE, controls, content parts, and rich
media.

- [x] Decide per payload family: render, fallback-render, preserve/report, or
  out-of-scope.
  Evidence: M11 keeps the supported diagram subset to related diagram drawing
  parts that lower into static shape/text primitives; tables remain rendered as
  table graphic frames; chart graphic frames, OLE objects, controls, content
  parts, audio/video media, and unknown graphicData payloads are preserved and
  reported as unsupported during render. OLE/control preview pictures remain
  renderable through the normal picture path when present.
- [x] Add fixtures for each decision.
  Evidence:
  `go test ./internal/render -run 'TestM11|Test.*Diagram|Test.*GraphicFrame|Test.*Chart|Test.*Unsupported' -count=1`
  passed on 2026-06-01. New M11 fixtures cover chart payload detection,
  chart unsupported render reporting, contentPart/OLE/control/audio/video
  classification, OLE preview-picture rendering, and precise unsupported
  family reports.
- [x] Add relationship/content-type detection tests.
  Evidence: `TestM11RenderGraphicFrameReportsChartPayload` and
  `TestM11RenderElementsReportsOLEAndRendersPreviewPicture` assert relationship
  target details in unsupported records. `TestWritePreservesUnsupportedPayloadParts`
  proves chart, OLE, ActiveX, media, and relationship parts are written back
  byte-for-byte by the package writer.
- [x] Ensure unsupported visible payloads produce precise JSON.
  Evidence: the renderer now emits payload-family-specific unsupported records
  for charts, unknown graphicData, OLE, controls, content parts, audio, and
  video instead of relying on generic unrendered-object messages.
- [x] Ensure edit/write paths preserve unsupported payloads where possible.
  Evidence:
  `go test ./internal/... -run 'Test.*Preserve|Test.*Unsupported|Test.*Validate' -count=1`
  passed on 2026-06-01, including package-writer preservation of unsupported
  chart/OLE/control/media parts and existing edit preservation tests.
- [x] Regenerate the schema matrix for chart/diagram/content rows.
  Evidence:
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed and
  updated `pml.xsd` OLE/control/content/graphic-frame rows,
  `dml-diagram.xsd` data/relId rows, `dml-chart.xsd`, and
  `dml-chartDrawing.xsd`; totals are 16 core-static, 149 common-partial, 337
  hard-rendering, 431 unsupported-preserve, and 74 out-of-scope declarations.
- [x] Update rendering support/failure-mode docs.
  Evidence: `docs/RENDERING.md` now states that chart graphic frames, OLE,
  controls, content parts, audio, video, and unknown graphic-frame payloads are
  detected, preserved, and reported, while only preview pictures render through
  the normal picture path.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record M11 changed files, residual risk, and next checkpoint.
  Evidence: M11 changed `internal/render/render.go`,
  `internal/render/render_types.go`, `internal/render/render_parse.go`,
  `internal/render/render_paint.go`, `internal/render/render_scene.go`,
  `internal/render/render_unsupported.go`, `internal/render/render_m11_test.go`,
  `internal/pptx/writer_test.go`,
  `tools/generate_ooxml_drawingml_audit.py`,
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`,
  `docs/renderer-coverage-summary.json`, `docs/RENDERING.md`,
  `docs/RENDERER_COMPLETION_CHECKLIST.md`, and
  `docs/RENDERER_EXPERIMENT_LOG.md`; residual risk is that charts and
  SmartArt layout remain partial static-rendering implementation work, while
  OLE application content, ActiveX controls, arbitrary content parts, and
  media playback remain preserve/report boundaries; next checkpoint is M12
  final conformance and release audit.

### M12: Final Conformance And Release Audit

Goal: prove supported static-renderer completion under the agreed scope, or
record exact blockers without hiding unsupported content or fixture failures.

- [x] Regenerate the coverage matrix and summary.
  Evidence: `python3 tools/generate_ooxml_drawingml_audit.py` passed on
  2026-06-01. The current generated totals are 16 core-static, 365
  common-partial, 91 hard-rendering, 395 unsupported-preserve, and 140
  out-of-scope declarations. The matrix now has 0 rows marked
  `Unimplemented / no evidence`; M12 added explicit source-backed reporting for
  embedded-font declarations and Partial/report evidence for table cell 3-D
  properties rather than using Unsupported as a shortcut.
- [x] Confirm no row is incorrectly marked Supported.
  Evidence: the only Supported matrix rows are the 16 core-static package,
  presentation, slide-size, slide-order, and low-level geometry/unit rows:
  `CT_Empty`, `CT_SlideIdListEntry`, `CT_SlideIdList`, `CT_SlideSize`,
  `presentation`, `ST_Coordinate`, `ST_Coordinate32`,
  `ST_PositiveCoordinate`, `ST_PositiveCoordinate32`, `ST_Angle`,
  `ST_PositiveFixedAngle`, `ST_Percentage`, `CT_Ratio`, `CT_Point2D`,
  `CT_PositiveSize2D`, and `CT_RelativeRect`. Renderer object families remain
  Partial, preserve/report, or out-of-scope instead of being over-promoted.
- [x] Run all unit tests.
  Evidence: `go test ./... -count=1` passed on 2026-06-01 after the M12
  direct table-property fill/noFill background update.
- [x] Run CLI/JSON compatibility checks.
  Evidence:
  `go test ./internal/cli -run 'TestRenderJSON|TestRenderJSONHonorsDPIFlag' -count=1 -v`
  passed on 2026-06-01. A direct supported-ish render command also passed:
  `go run ./cmd/puppt render testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/fixture.pptx --slide 1 --out /tmp/puppt-m12-supportedish.png --json`.
  A direct real-world render command also passed and kept the same JSON shape:
  `go run ./cmd/puppt render testdata/realworld-ppts/EPA-generate-2021-presentation.pptx --slide 1 --out /tmp/puppt-m12-realworld-slide001.png --json`.
- [x] Audit production renderer dependencies.
  Evidence:
  `go test ./internal/render -run TestRendererImplementationHasNoTargetDeckHardcodesOrExternalRendererCalls -count=1 -v`
  passed. `go list -deps ./cmd/puppt | rg -i 'libreoffice|powerpoint|keynote|soffice|chrom(e|ium)|playwright|puppeteer|selenium|unoconv|cloudconvert|magick|slides'`
  returned no dependency hits. A production-code grep found only command text
  and test assertions, not renderer implementation calls.
- [x] Run the real-world perceptual metrics command.
  Evidence:
  `PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 PUPPT_REALWORLD_PERCEPTUAL_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/realworld-perceptual-summary-m12-current.json go test ./internal/render -run TestRealWorldPerceptualMetrics -count=1 -v`
  passed with 61 slides, 61 differing slides, mean luma similarity
  0.950452042, mean channel-RMS similarity 0.827985604, and 9,337,907 total
  differing pixels.
- [ ] Pass the exact real-world Apple Notes reference gate.
  Evidence:
  `PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1`
  failed on 2026-06-01 after the latest accepted direct table-property
  fill/noFill background update: 61 of 61 slides differ from the Apple Notes
  references, total differing pixels are 9,340,612, worst slide is
  `testdata/realworld-ppts/EPA-generate-2021-presentation.pptx` slide 001
  with 307,925 differing pixels, and top unsupported rendering gaps are
  `none`.
- [ ] Pass the clean object fixture suite.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/clean-micro-fixture-suite-m12-current.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed only in expected-failure accounting mode after the current M12
  direct table-property fill/noFill background update, with 59 tracked clean
  fixtures, 0 passed, and 59 failed. The ownership artifact classifies the
  fixture corpus as 179
  target-scoped manifests, 59 clean failures, 74 contaminated failures, and 10
  partial-underpaint failures. Clean failures are 30 picture fixtures with
  1,450,858 differing pixels, 5 graphic-frame/table fixtures with 557,772
  differing pixels, and 24 shape fixtures with 571,458 differing pixels.
- [x] Record production scoreboard diagnostics.
  Evidence:
  `PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard-m12-current.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v`
  passed after the current M12 zero-height table-row layout update with 61
  slides, 9,337,907 total slide differing pixels, 8 object groups, and 59 clean
  fixture failures. The highest-impact attributed
  object-overlap groups are shape geometry/fill/line/clipping/antialiasing
  (176 objects, 7,564,416 overlap pixels), text shaping/font metrics/paragraph
  layout/anchoring (288 objects, 3,478,136 overlap pixels), and picture
  crop/resampling/color/media transform (168 objects, 2,356,361 overlap
  pixels). These are triage queues, not broad tuning permission.
- [x] Record rejected M12 shape candidates that failed object-fixture proof.
  Evidence: two source-backed candidates were tested and reverted on
  2026-06-01: preserving fractional bounds through `spAutoFit` for WHO slide
  003 `TextBox 7` failed the fixture at 133,021 differing pixels, while the
  current rerun after revert still fails at 133,022 differing pixels; defaulting
  omitted `a:ln/@algn` to centered pen alignment for WHO slide 002
  `Rectangle 11` failed the fixture at 71,272 differing pixels, and the current
  rerun after revert remains 71,272 differing pixels. The exact commands and
  rejection rationale are recorded in `docs/RENDERER_EXPERIMENT_LOG.md`.
- [x] Correct visible fixture masking for later source objects.
  Evidence: EPA Residential Wood slide 005 picture `Google Shape;108;p4` is a
  source picture, while the visible reference labels over it are later
  `<p:sp>` text boxes in z-order. M12 updated the micro-fixture occlusion
  records to use source-authored `pixel_bounds` for those later objects instead
  of current rendered ink bounds, added
  `TestMicroFixtureOcclusionsUseSourceBoundsForLaterTextBoxes`, and refreshed
  the object-debug artifacts. The targeted fixture still fails at 86,813
  visible differing pixels after the label occlusion is correctly masked, so
  this is accepted only as fixture attribution/harness correction; it is not a
  renderer parity completion.
- [x] Preserve package table styles in table micro-fixtures.
  Evidence: WHO HIV slide 012 `Table 3` uses
  `a:tblPr/a:tableStyleId={5C22544A-7EE6-4342-B048-85BDC9FD1C3A}`, which
  resolves through package part `ppt/tableStyles.xml`. M12 updated the
  graphic-frame/table fixture builder to copy that package dependency and added
  `TestShapeObjectFixtureCopiesTableStylesForTableGraphicFrames`. The
  regenerated fixture now contains `ppt/tableStyles.xml` and renders the
  styled first row, banded rows, and white table borders. The targeted fixture
  still fails at 284,470 pixels, so the accepted change is fixture dependency
  preservation; remaining table parity is still a supported rendering gap.
- [x] Record source-resolved table-style color profile for the current top
  table residual.
  Evidence: `TestMicroFixtureTableStyleColorProfile` profiles the WHO slide
  012 `Table 3` fixture from source OOXML and writes
  `table-style-color-profile-m12.json`. It confirms
  `Medium Style 2 - Accent 1` resolves first-row and band-row fills through
  `ppt/tableStyles.xml`; sampled rendered colors match the source-resolved
  Display P3 colors for the header and are one channel away from reference
  samples on band fills. The remaining 284,470-pixel residual is therefore
  table color-management/text/border parity, not missing table-style package
  resolution. The coverage matrix rows for `CT_TableStyle`,
  `CT_TableStyleList`, and `tblStyleLst` now record that evidence.
- [x] Render conditional table-style boundary borders over inherited inside
  borders.
  Evidence: WHO HIV slide 012 `Table 3` uses `Medium Style 2 - Accent 1`.
  Its source `ppt/tableStyles.xml` has `wholeTbl/tcBdr/insideH` at 12,700
  EMUs and `firstRow/tcBdr/bottom` at 38,100 EMUs. The renderer previously
  resolved the first-row bottom edge as inherited `insideH`, which flattened
  the conditional-region boundary. M12 now tracks explicit non-`wholeTbl`
  boundary borders and repaints them after inherited inside borders, while
  direct cell borders still take precedence. Focused verification:
  `go test ./internal/render -run 'TestM09TableStyleRegionBoundaryBorderOverridesInsideBorder|TestM09TableStyleDiagonalBordersApplyThroughResolvedCellStyle|TestM09RenderGraphicFramePaintsDiagonalCellBorders' -count=1`
  passed. `TestMicroFixtureTableStyleColorProfile` now records the first-row
  bottom border as 38,100 EMUs in
  `table-style-color-profile-m12.json`. The `Table 3` object fixture still
  fails unchanged at 284,470 differing pixels, so table text/color/border
  parity remains open. The exact Apple Notes gate still fails after this
  boundary-border support with 61/61 differing slides, 9,341,866 total
  differing pixels, and no unsupported rendering gaps.
- [x] Preserve table-cell `anchorCtr` text anchoring.
  Evidence: `dml-main.xsd:2347 CT_TableCellProperties` defines
  `anchorCtr` next to `anchor`. M12 now parses `a:tcPr/@anchorCtr` into
  `tableCell`, lowers it to the existing text element `TextAnchorCenter`
  behavior, and keeps `anchor` vertical placement unchanged. Focused synthetic
  tests passed:
  `go test ./internal/render -run 'TestParseTableCellAnchorCenterLowersToTextElement|TestDrawShapeTextHonorsAnchorCenter|TestTableCellTextAnchorDoesNotInferRowSpanCentering|TestParseTableCellMarginsKeepsDefaultsForOmittedSides' -count=1 -v`.
  The current top table fixture, WHO slide 012 `Table 3`, still fails unchanged
  at 284,470 differing pixels, so table parity remains open.
- [x] Preserve table-cell text overflow and vertical text metadata.
  Evidence: `dml-main.xsd:2347 CT_TableCellProperties` defines
  `horzOverflow`, `vertOverflow`, and `vert` next to table-cell anchoring
  attributes. Shape text already lowers these attributes from `a:bodyPr` into
  the text layout/reporting path, but table cells previously dropped the
  equivalent `a:tcPr` attributes. M12 now parses them into `tableCell` and
  lowers them to `slideElement` so supported overflow clips use the existing
  text clip path and unsupported vertical text modes are reported by the
  existing text feature checks rather than silently ignored. The focused
  synthetic parser/lowering test passed with the same command as the
  `anchorCtr` test group. WHO slide 012 `Table 3` still fails unchanged at
  284,470 differing pixels, so this is coverage of a dropped source property
  path, not table parity completion.
- [x] Resolve table-style cell `fillRef` through theme fill styles.
  Evidence: `dml-main.xsd:2499 CT_TableStyleCellStyle` includes
  `EG_ThemeableFillStyle`, and `dml-main.xsd:2440 EG_ThemeableFillStyle`
  includes `fillRef`. M12 now resolves `tcStyle/fillRef` entries through the
  package theme fill style matrix, applying the `fillRef` placeholder color in
  the same bounded path already used for table background `fillRef`
  resolution. EPA Residential Wood contains real package evidence in
  `ppt/tableStyles.xml`, including a table style used by slide 015
  `Google Shape;193;p12`. The focused synthetic table-style tests passed:
  `go test ./internal/render -run 'TestParseTableStylesReadsTableBackgroundFillReference|TestParseTableStylesResolvesCellStyleFillReference|TestThemeFillStylesResolveBackgroundFillReference|TestParseTableStylesReadsConditionalRegions|TestParseTableStylesResolvesThemeLineReferences' -count=1 -v`.
  The EPA slide 015 `Google Shape;193;p12` fixture still fails at 64,393
  differing pixels, and the nearby EPA slide 013 `Google Shape;179;p9` table
  fixture remains unchanged at 127,315 differing pixels, so this is accepted as
  a bounded source-style resolution fix, not table-family parity completion.
- [x] Render direct table-property background fill and noFill.
  Evidence: `dml-main.xsd:2405 CT_TableProperties` includes
  `EG_FillProperties` before `tableStyleId`. EPA Residential Wood slide 013
  `Google Shape;179;p9` has direct `a:tblPr/a:noFill` next to
  `a:tableStyleId={D1725187-6464-411F-8C7F-DCDDFD2443DF}`. M12 now parses
  direct table-property `solidFill` and `noFill` into `tableModel`, renders a
  direct table background before falling back to the style table background,
  and lets direct `noFill` suppress style table background fill. Focused
  synthetic tests passed:
  `go test ./internal/render -run 'TestParseTableModelReadsTablePropertiesFill|TestParseTableModelReadsTablePropertiesNoFill|TestRenderGraphicFramePaintsTableStyleBackground|TestRenderGraphicFrameUsesDirectTablePropertiesBackgroundBeforeStyle|TestRenderGraphicFrameTablePropertiesNoFillSuppressesStyleBackground' -count=1 -v`.
  The EPA slide 013 `Google Shape;179;p9` fixture still fails unchanged at
  127,315 differing pixels, and WHO slide 012 `Table 3` remains unchanged at
  284,470 differing pixels, so this is accepted as table-property fill
  semantics coverage, not table-family parity completion.
- [x] Render direct table-cell gradient and pattern fills through shared paint.
  Evidence: `dml-main.xsd:1587 CT_FillProperties` permits `gradFill` and
  `pattFill` anywhere `EG_FillProperties` is accepted, including table cell
  properties. M12 now parses direct `a:tcPr/a:gradFill` and
  `a:tcPr/a:pattFill` into `backgroundPaint`, preserves full table-style
  `fill`/`fillRef` paint instead of flattening it to a color, and renders cell
  fills through the same solid/gradient/pattern paint path used elsewhere.
  `a:blipFill` and unresolved `a:grpFill` table-cell fills remain reported as
  image/group cell fills. Focused tests passed:
  `go test ./internal/render -run 'TestRenderGraphicFramePaintsGradientTableCellFill|TestParseTableModelRecordsUnsupportedVisibleFeatures|TestTableCellFillDirectNoFillSuppressesStyleFill' -count=1 -v`
  and broader table-focused tests passed:
  `go test ./internal/render -run 'TestRenderGraphicFrame|TestParseTable|TestTableCell|Test.*Table.*Fill' -count=1`.
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed
  with unchanged queue totals because fill-property declarations remain Partial
  for image/tile details and advanced gradient clauses. The clean fixture suite
  still passed only in expected-failure accounting mode after this change:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-cell-paints.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  reported 59 total, 0 passed, and 59 failed; top failures remain `Table 3`
  284,470, `Picture 2` 154,741, `TextBox 7` 130,250, and
  `Google Shape;179;p9` 127,167.
- [x] Reject the M12 `Picture 2` area-resampling candidate.
  Evidence: WHO HIV slide 009 `Picture 2` is a source-backed
  `p:pic`/`a:blipFill` object with `a:stretch/a:fillRect`, no crop, and a
  2830x820 ICC-profiled PNG. The current fixture residual is 154,741 pixels.
  `TestMicroFixturePictureAreaSearch` produced
  `picture-area-search-m12.json`; its best area candidate,
  `converted_icc/area_srgb_byte/floor_floor`, still failed at 155,545 pixels.
  The channel-delta magnitude improved, but the object fixture worsened and did
  not pass, so no production picture sampling change was accepted.
- [x] Implement table-row text minimum reflow for source table cells.
  Evidence: `renderTableGraphicFrame` now measures rendered table-cell text
  against each source cell width and expands rows whose text needs a larger
  minimum, shrinking rows with remaining capacity so the table stays inside the
  authored graphic-frame bounds. Focused tests pass:
  `TestAdjustTableRowOffsetsForMinimumHeights*`, `TestTableRowOffsets*`, and
  existing graphic-frame table tests. The EPA residential slide 013
  `Google Shape;179;p9` fixture was rerun and still fails at 127,315 pixels, so
  this is an implemented table-layout primitive, not a completed table parity
  fix.
- [x] Implement source-text row proportions for tables with only zero authored
  row heights.
  Evidence: WHO HIV slide 015 `Table 2` has six `a:tr h="0"` rows under
  `CT_TableRow`; the first-row header cells contain multiple source
  paragraphs, for example `Assay` and `1`, and the previous equal-row fallback
  let the second header paragraph spill into the first body row. M12 now
  derives row proportions from measured source table-cell text heights when
  every authored row height is zero, with
  `TestTableRowOffsetsWithZeroAuthoredHeightsGrowMultiParagraphHeader`
  covering the synthetic source case. Focused table tests and
  `go test ./internal/render -count=1` pass. The targeted `Table 2` fixture
  improved from 63,031 to 55,832 differing pixels but still fails, so remaining
  table parity stays open.
- [x] Measure spanning table-cell text against the full source `gridSpan` width
  during row-height reflow.
  Evidence: WHO HIV slide 008 `Table 15` has a first-row header cell
  `a:tc gridSpan="3"` with two source paragraphs and two following
  `hMerge="1"` cells. M12 updated the table text-minimum measurement to use
  the spanned column width for `CT_TableCell/@gridSpan` instead of measuring
  against only the first column, and added
  `TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth`. Focused table
  tests, `go test ./internal/render -count=1`, and clean-suite
  expected-failure accounting passed. At that checkpoint the targeted
  `Table 15` fixture remained unchanged at 79,708 differing pixels, so the
  change was accepted as source semantics coverage, not table fixture
  completion.
- [x] Reflow over-capacity first-row spanning tables by source text-minimum
  proportions.
  Evidence: WHO HIV slide 008 `Table 15` has equal authored
  `CT_TableRow/@h="370840"` rows, but its first-row `CT_TableCell/@gridSpan="3"`
  header has two centered source paragraphs. The renderer measured source row
  text minimums that exceeded the fixed graphic-frame height and previously
  left all row offsets unchanged. M12 now uses measured source text-minimum
  proportions for this first-row spanning/header over-capacity case while
  keeping the table inside the authored frame, with
  `TestTableRowOffsetsWithTextMinimumsUsesMinimumProportionsWhenFrameIsOverCapacity`
  covering the synthetic source case. Focused table tests pass, and
  `Table 15` improved from 79,708 to 72,605 differing pixels. Neighbor table
  fixtures were checked: `Table 2` remains 55,832 and `Table 3` remains
  284,470. Clean-suite expected-failure accounting still passes with 59
  failures; the graphic-frame/table bucket is now 557,772 pixels.
- [x] Reflow over-capacity first-row tables without requiring a spanning header.
  Evidence: EPA Residential Wood slide 013 `Google Shape;179;p9` has
  `a:tblPr firstRow="1" bandRow="1"` and a non-spanning first row whose source
  cells contain wrapping header text such as `Emission Rate PM2.5 (g/hr)` and
  `Firepower(W)`. M12 widened the first-row over-capacity text-minimum
  allocator from spanning-only first rows to any authored `firstRow` where the
  first row is the only row over its measured text minimum, added
  `TestTableRowOffsetsWithTextMinimumsReflowsNonSpanningFirstRowWhenFrameIsOverCapacity`,
  and kept the table inside the authored graphic-frame bounds. Focused table
  tests and coverage summary pass. The targeted Google table fixture remains
  unchanged at 127,315 differing pixels, while `Table 3` remains 284,470 and
  `Table 15` remains 72,605, so this is accepted as source semantics coverage,
  not a table fixture completion.
- [x] Render source-backed DrawingML shape and picture blur effects.
  Evidence: `dml-main.xsd:1264 CT_BlurEffect` defines `rad` and optional
  `grow` with schema default `true`. M12 now parses `a:blur/@rad` and
  `a:blur/@grow`, carries them through shape/picture primitives, renders
  supported static shapes and pictures through an isolated RGBA blur layer,
  and composites either the grown or clipped result.
  `TestM10CollectSlideElementsParsesBlurEffect` covers the default
  `grow=true` parse path, `TestM10RenderShapePaintsBlurEffect` covers visible
  shape blur outside authored bounds, and
  `TestM10PictureBackendPaintsBlurEffect` covers picture blur. The matrix now
  records `CT_BlurEffect` as Partial in `hard-rendering`, not Unsupported;
  combined blip blur with higher-order object effects remains an explicit
  partial report until implemented.
- [x] Render source-backed DrawingML fill overlay effects.
  Evidence: `dml-main.xsd:1606 CT_FillOverlayEffect` defines a required
  `EG_FillProperties` fill and required `ST_BlendMode` blend value. M12 now
  parses `a:fillOverlay`, carries the resolved fill and blend through
  shape/picture primitives, renders supported static shapes and pictures into an
  isolated layer, applies `over`, `mult`, `screen`, `darken`, or `lighten`
  overlay blending to object pixels, and composites the result back.
  M12 also parses `a:blip/a:fillOverlay`, lowers it as separate source-image
  metadata, and applies the same blend modes to the decoded/cropped/effected
  image source before picture sampling.
  `TestM10CollectSlideElementsParsesFillOverlayEffect`,
  `TestM10RenderShapePaintsFillOverlayEffect`, and
  `TestM10PictureBackendPaintsFillOverlayEffect` cover the synthetic source
  paths; `TestM10FillOverlayImplementsSchemaBlendModes` covers all
  `ST_BlendMode` enum values. The matrix now records
  `CT_FillOverlayEffect` as Partial in `hard-rendering` and `ST_BlendMode` as
  Partial, not Unsupported; complex effect-ordering parity remains partial.
- [x] Render source-backed DrawingML inner shadow effects.
  Evidence: `dml-main.xsd:1297 CT_InnerShadowEffect` defines a required color
  choice and optional `blurRad`, `dist`, and `dir` attributes with zero
  defaults. M12 now parses `a:innerShdw`, carries the resolved color,
  blur/distance/direction through shape, picture, theme, and scene effect
  primitives, renders supported static shapes and pictures into an isolated
  layer, applies an inward alpha-mask shadow, and composites the result back.
  `TestM10CollectSlideElementsParsesInnerShadowEffect`,
  `TestM10RenderShapePaintsInnerShadowEffect`, and
  `TestM10PictureBackendPaintsInnerShadowEffect` cover the synthetic source
  paths. The matrix now records `CT_InnerShadowEffect` as Partial in
  `hard-rendering`, not Unsupported; full host effect-ordering parity remains
  partial.
- [x] Render source-backed DrawingML reflection effects.
  Evidence: `dml-main.xsd:1355 CT_ReflectionEffect` defines reflection blur,
  alpha ramp, distance/direction, fade direction, scale/skew, alignment, and
  rotate-with-shape attributes with schema defaults. M12 now parses
  `a:reflection`, carries the reflection model through shape, picture, theme,
  and scene effect primitives, renders supported static shapes and pictures
  into an isolated layer, applies a bottom mirror reflection with the authored
  alpha ramp, optional blur, and distance offset, and composites the result
  back. Non-bottom transform variants are explicitly reported as simplified.
  `TestM10CollectSlideElementsParsesReflectionEffect`,
  `TestM10RenderShapePaintsReflectionEffect`, and
  `TestM10PictureBackendPaintsReflectionEffect` cover the synthetic source
  paths. The matrix now records `CT_ReflectionEffect` as Partial in
  `hard-rendering`, not Unsupported; full host transform/effect-ordering parity
  remains partial.
- [x] Render simple DrawingML effectDag containers containing supported static
  effects.
  Evidence: `dml-main.xsd:1615 EG_Effect` and
  `dml-main.xsd:1655 CT_EffectContainer` define effect graph containers that
  can hold supported static effects such as `glow`, `blur`, `outerShdw`,
  `innerShdw`, `reflection`, `fillOverlay`, and `softEdge`. M12 now flattens
  simple `a:effectDag/a:cont` subsets containing those supported effects into
  the normal effect-list renderer and reports unsupported graph ordering or
  graph-only effect nodes explicitly. `TestM10CollectSlideElementsReportsEffectDag`,
  `TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes`, and
  `TestM10RenderShapePaintsFlattenedEffectDagGlow` cover the synthetic source
  paths. The matrix now records `EG_Effect` as Partial, not Unsupported; full
  graph ordering/compositing remains partial.
- [x] Flatten simple DrawingML blend effectDag children containing supported
  static effects.
  Evidence: `dml-main.xsd:1665 CT_BlendEffect` defines a required child
  `cont` and required `ST_BlendMode` blend value. M12 now flattens
  `a:blend/a:cont` children when they contain already-supported static effects,
  marks the object as rendered through the normal effect renderer, and still
  reports the blend node as partial because full effect-graph blend compositing
  is not implemented. `TestM10CollectSlideElementsFlattensSupportedBlendEffectDagChild`
  covers the synthetic source path. The matrix now records `CT_BlendEffect` as
  Partial, not Unsupported; remaining graph-only blend compositing stays
  explicit partial work.
- [x] Implement authored-hyphen wrap points for horizontal text.
  Evidence: WHO HIV slide 003 `TextBox 7` contains a `CT_RegularTextRun` token
  `treatment-adjusted`; the reference wraps at the authored hyphen while the
  previous renderer treated the whole hyphenated word as unbreakable. Synthetic
  tests now cover plain and styled hyphen wrap points. The object fixture now
  renders the hyphenated break in the same line pattern as the reference, but it
  still fails at 130,250 visible-crop pixels, compared with the prior 130,103
  diagnostic count. Neighboring TextBox fixtures remain at their documented
  residuals: slide 015 `TextBox 7` 19,939 and slide 013 `TextBox 3` 25,347.
- [x] Implement authored-slash wrap points for horizontal table text.
  Evidence: EPA Residential Wood slide 013 `Google Shape;179;p9` contains
  first-row table header `CT_RegularTextRun` text split around `PM2.5 (g/hr)`,
  including authored runs `(g/`, `hr`, and `)`. Treating `/` as a text wrap
  opportunity preserves source text while allowing the table header layout to
  break at the authored separator. `TestWrapTextWithPrefixesBreaksAfterAuthoredSlash`
  and `TestStyledWordTokensExposeSlashWrapPoint` cover the synthetic source
  path. The targeted EPA slide 013 table fixture moved from 127,315 to 127,167
  differing pixels but still fails. Neighboring table checks did not show a
  new table-family regression: WHO slide 012 `Table 3` remains 284,470 and WHO
  slide 008 `Table 15` remains 72,605.
- [x] Preserve authored empty DrawingML paragraphs during text layout.
  Evidence: WHO HIV slide 003 `TextBox 7` contains authored empty `a:p`
  elements with `a:endParaRPr sz="2200"` between visible bullet paragraphs.
  The parser already preserved those `CT_TextParagraph` records and resolved
  their end-paragraph run metrics, but layout skipped paragraphs that produced
  no text segments. M12 now emits a blank layout line for such paragraphs,
  preserving resolved font size, paragraph spacing, alignment, line spacing,
  and tab stops. `TestTextRenderLinesPreserveAuthoredEmptyParagraphs` covers
  the synthetic source path. The targeted slide 003 `TextBox 7` fixture remains
  an expected failure at 130,250 visible-crop pixels, so this is accepted as
  source-semantics coverage, not as M12 completion.
- [x] Preserve explicit empty `a:buChar` paragraphs as bullet lines.
  Evidence: WHO HIV slide 003 `TextBox 7` contains empty `a:p` elements whose
  local `a:pPr` includes `a:buFont typeface="Arial"`, `a:buChar char="•"`,
  `marL="285750"`, and `indent="-285750"`, followed only by `a:endParaRPr`.
  M12 now suppresses bullets on empty paragraphs only when the paragraph has no
  local bullet choice, so explicit empty `CT_TextCharBullet` paragraphs render
  a bullet prefix while empty paragraphs without `buChar` stay blank.
  `TestTextParagraphsFromNodePreservesExplicitEmptyBulletParagraphs` and
  `TestTextRenderLinesPreserveExplicitEmptyBulletParagraphs` cover the
  synthetic source path. The targeted slide 003 `TextBox 7` fixture remains an
  expected failure at 130,250 visible-crop pixels, so this is accepted as
  source-semantics coverage, not as M12 completion.
- [x] Preserve `CT_TextCharacterProperties@lang` through text layout.
  Evidence: WHO HIV slide 003 `TextBox 7`, WHO slide 012 `Table 3`, and
  neighboring clean text/table fixtures author `a:rPr/@lang` and
  `a:endParaRPr/@lang` values such as `en-US`, `en-ES`, and `en-GB`.
  ECMA-376 `dml-main.xsd:2873 CT_TextCharacterProperties` defines optional
  `@lang` at `dml-main.xsd:2891`. M12 now parses run and end-paragraph
  language into paragraph defaults, carries direct run language separately, and
  resolves language onto text render segments for future shaping/font fallback
  decisions. `TestTextParagraphsFromNodeCapturesRunLanguage` covers direct run,
  paragraph-default, render-segment, and empty-paragraph `endParaRPr` paths.
  The targeted slide 003 `TextBox 7` fixture remains an expected failure at
  130,250 visible-crop pixels, and the clean fixture suite still records 59
  total, 0 passed, and 59 failed. This is source-semantics coverage, not M12
  completion; the failing object fixtures remain open.
- [x] Parse and render `ST_TextCapsType` for horizontal text runs.
  Evidence: ECMA-376 `dml-main.xsd:2866 ST_TextCapsType` defines `none`,
  `small`, and `all`, and `dml-main.xsd:2899 CT_TextCharacterProperties@cap`
  defaults to `none`. M12 now parses direct run and paragraph-default `cap`
  values, carries list-style defaults through paragraph style merging, renders
  `cap="all"` as uppercase run text, renders `cap="small"` as uppercase text
  with lowercase source letters split into smaller segments, and preserves
  explicit `cap="none"` as an override. Focused validation:
  `go test ./internal/render -run 'TestTextParagraphsFromNodeParsesRunCaps|TestTextParagraphsFromNodeParsesRunCharacterSpacing|TestTextRunFromNodeReadsDrawingMLKernThreshold|TestMeasureStyledSegmentsIncludesCharacterSpacing' -count=1`
  passed. WHO HIV slide 003 `TextBox 7` authors only `cap="none"` in its
  relevant runs, and the targeted object fixture remained the same expected
  failure at 130,250 visible-crop pixels; this is source-semantics coverage, not
  M12 completion. `go test ./...` passed, and the clean fixture suite passed
  only in expected-failure accounting mode at 59 total, 0 passed, and 59 failed.
- [x] Preserve `CT_TextBodyProperties@rtlCol` through text-body primitives and
  object-debug summaries.
  Evidence: WHO HIV slide 003 `TextBox 7` authors `a:bodyPr wrap="square"
  rtlCol="0"` with `a:spAutoFit`. ECMA-376
  `dml-main.xsd:2625 CT_TextBodyProperties` defines optional `@rtlCol` at
  `dml-main.xsd:2637`. M12 now parses `rtlCol`, inherits it from placeholder
  body properties when the local body property omits it, lowers it into render
  text primitives, emits `resolved_style.text_body_properties`, and reports
  authored right-to-left multi-column order as a partial static text layout
  gap. Focused validation:
  `go test ./internal/render -run 'TestParseBodyPropertiesReadsTextAnchor|TestM08TextPrimitiveReportsFontResolutionAndTextUnsupportedModes|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestResolveSlidePlaceholdersInheritsUnspecifiedBodyTextProperties' -count=1`
  passed. The targeted slide 003 `TextBox 7` object record now contains
  `text_body_properties=["rtlCol=false"]` and `unsupported=null`, but the
  fixture remains an expected failure at 130,250 visible-crop pixels, so this
  is source-semantics coverage, not M12 completion.
- [x] Preserve text-body wrap, autofit, and spacing metadata through text
  primitives and object-debug summaries.
  Evidence: WHO HIV slide 003 `TextBox 7` authors `a:bodyPr wrap="square"
  rtlCol="0"` with `a:spAutoFit`. ECMA-376 `dml-main.xsd:2625
  CT_TextBodyProperties` defines text-body attributes, `dml-main.xsd:2653
  CT_TextBody` owns the `bodyPr` child, and the `EG_TextAutofit` choice
  carries `CT_TextShapeAutofit` / `CT_TextNormalAutofit` metadata. M12 now
  emits object-debug body-property summaries for `wrap`, shape/normal/no
  autofit, font scale, line-spacing reduction, first/last paragraph spacing,
  and `rtlCol`, while preserving `fontScale`, `lnSpcReduction`, and
  `spcFirstLastPara` values through primitive lowering. Focused validation:
  `go test ./internal/render -run 'TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestParseBodyPropertiesReadsTextAnchor' -count=1 -v`
  passed. The targeted slide 003 `TextBox 7` object record now contains
  `text_body_properties=["wrap=square","spAutoFit=true","rtlCol=false"]`,
  while the fixture remains an expected failure at 130,250 visible-crop
  pixels. The clean fixture suite passed only in expected-failure accounting
  mode with 59 total, 0 passed, and 59 failed; top blockers remain `Table 3`
  284,470, `Picture 2` 154,741, `TextBox 7` 130,250,
  `Google Shape;179;p9` 127,167, and EPA `Picture 2` 95,960. This is
  source-semantics and reporting coverage, not M12 completion.
- [x] Preserve and render paragraph `fontAlgn` for horizontal styled text.
  Evidence: `dml-main.xsd:2979 ST_TextFontAlignType` defines `auto`, `t`,
  `ctr`, `base`, and `b`, and `dml-main.xsd:3015
  CT_TextParagraphProperties@fontAlgn` carries the value on paragraphs. WHO HIV
  slide 003 `TextBox 7` and WHO slide 012 `Table 3` both contain
  `fontAlgn="auto"`. M12 now parses direct and list-style paragraph
  `fontAlgn`, carries it into render lines, and applies top/center/bottom
  metric alignment for styled horizontal text while leaving `auto`/`base` on
  the existing baseline behavior. `TestTextParagraphsFromNodeCapturesParagraphFontAlign`,
  `TestTextParagraphsFromNodeInheritsListStyleFontAlign`, and
  `TestSegmentFontAlignmentShiftUsesLineMetrics` cover the synthetic source
  path. The targeted `TextBox 7` and `Table 3` fixtures remain expected
  failures at 130,250 and 284,470 pixels respectively, so this is accepted as
  source-semantics coverage, not as M12 completion. `go test ./...` passed,
  the clean fixture suite passed only in expected-failure accounting mode with
  59 total, 0 passed, and 59 failed, and the exact Apple Notes gate remained an
  expected failure at 61/61 differing slides and 9,305,437 total differing
  pixels with no unsupported rendering gaps.
- [x] Resolve inherited font sizes for baseline run rendering.
  Evidence: WHO HIV slide 003 `TextBox 7` contains a superscript
  `CT_RegularTextRun` whose `a:rPr` has `baseline="30000"` but no local `sz`;
  the run inherits its render size from the surrounding text element. M12 now
  resolves baseline run font sizes at measurement/drawing time from the
  element fallback when the run and paragraph do not carry an explicit size,
  then applies the existing baseline scale. Validation:
  `go test ./internal/render -run 'TestBaselineRunWithoutLocalSizeUsesElementFallbackForRenderSize|TestTextParagraphsFromNodeCapturesRunBaseline|TestTextParagraphsFromNodeParsesRunCaps|TestTextParagraphsFromNodePreservesRunLanguage|TestMeasureStyledSegmentsIncludesCharacterSpacing' -count=1`
  passed, `go test ./internal/render -count=1` passed, and the targeted slide
  003 `TextBox 7` fixture remained an expected failure at 130,250 differing
  pixels. This is source-semantics coverage, not M12 completion.
- [x] Preserve `CT_TextLineBreak` run properties as hard-break line metrics.
  Evidence: `dml-main.xsd:2957` declares `CT_TextLineBreak` with optional
  `a:rPr`, and WHO HIV slide 003 `Rectangle 3` authors
  `<a:br><a:rPr sz="3600" b="1"/></a:br>` between two bold text runs. M12 now
  keeps the break run as an empty metric segment on the preceding rendered
  line, so the authored break properties affect line metrics without drawing a
  glyph. Validation:
  `go test ./internal/render -run 'TestTextRenderLinesPreserveDrawingMLBreakRuns|TestTextRenderLinesPreserveDrawingMLBreakRunMetrics|TestNormalAutofitMaxSoftLinesHonorsWrapNoneAndHardBreaks|TestFitNormalAutofitAllowsWrappingWithinHardBreakLines' -count=1 -v`
  passed, and the targeted slide 003 `Rectangle 3` fixture remained an
  expected failure at 21,073 differing pixels. This is source-semantics
  coverage, not an M12 gate closure.
- [x] Reject generated bullet-prefix spacer font inheritance as currently
  modeled.
  Evidence: WHO HIV slide 003 `TextBox 7` uses source bullet and run fonts
  through `a:buFont typeface="Arial"` and `a:rPr/a:latin typeface="Arial"`.
  A candidate made the generated bullet-prefix spacer inherit the first text
  segment's metrics. Synthetic tests and shaping-profile proof passed, and the
  profile showed the spacer using Arial, but the targeted object fixture moved
  from 130,250 to 130,252 visible-crop differing pixels. The candidate was
  rejected and reverted. Bullet separator metrics, tab behavior, hanging
  geometry, and line placement remain supported-scope text-renderer work.
- [x] Reject character-bullet hanging tab stops as currently modeled.
  Evidence: WHO HIV slide 002 `Rectangle 11` has a level-0 character bullet
  paragraph with `marL="285750"` and `indent="-285750"`. A source-backed
  candidate generalized the existing auto-number hanging-tab-stop path to
  `a:buChar` bullets, but the targeted object fixture moved from 71,231 to
  71,244 differing pixels, slide 003 `TextBox 7` moved from 130,250 to
  130,392, and the exact real-world gate worsened to 9,356,836 pixels. The
  candidate was rejected and reverted. The remaining character-bullet hanging
  work stays supported-scope; it needs a stronger source model for body insets,
  tab stops, and literal leading spaces before a production change is accepted.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01 after M12 bullet-spacer rejection,
  documentation, and doctrine edits.
- [ ] Mark renderer completion goal complete.
  Evidence: not complete. M12 remains blocked by the exact real-world gate and
  the clean fixture suite. The coverage matrix has 0 rows with
  `Unimplemented / no evidence` status after M12 reconciliation. Unsupported is
  allowed only for source-proven impossible static rendering content, not for
  missing implementation, high pixel diff, local fixture failure, or difficult
  primitive behavior. Out-of-scope rows require schema/source evidence outside
  the static PresentationML/DrawingML renderer target and cannot cover visible
  supported-scope fixture or real-world gate failures.

## Phase 1: Stabilize The Worktree

Goal: the repository compiles and the interrupted render test split is not
blocking further evidence work.

- [x] Preserve `docs/2026-05-31-renderer-8h-investigation.md`.
  Evidence: present in worktree on 2026-06-01; no delete in `git status --short`.
- [x] Preserve `docs/RENDERER_EXPERIMENT_LOG.md`.
  Evidence: present in worktree on 2026-06-01; no delete in `git status --short`.
- [x] Confirm `internal/render/render_text_styles_test.go` is a completed,
  compiling split and not a parity strategy.
  Evidence: `go test ./internal/render -count=1` passed on 2026-06-01.
- [x] Run `go test ./internal/render -count=1`.
  Evidence: passed on 2026-06-01.
- [x] Run `go test ./...`.
  Evidence: passed on 2026-06-01.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01.
- [x] Record Phase 1 changed files, residual risk, and next checkpoint.
  Evidence: Phase 1 changed files are the current render/test split and
  renderer-goal docs in `git status --short`; residual risk is that visual
  parity remains unproven until Phase 2-7 evidence is complete; next checkpoint
  is Phase 2 attribution harness locking.

## Phase 2: Lock The Attribution Harness

Goal: object attribution is the default diagnostic tool for parity work and can
take a pixel residual back to package source.

- [x] Confirm every painted object record includes slide part, source part, XML
  path, cNvPr id/name, kind, and z-order.
  Evidence: focused object-debug attribution tests passed on 2026-06-01,
  covering painted-record metadata, unsupported-item summaries, real-world diff
  metadata, micro-fixture target ownership, clean-fixture ownership
  classification, and fixture record emission.
- [x] Confirm every painted object record includes EMU bounds, fractional pixel
  bounds, integer pixel bounds, output changed-pixel bounds, and painted-output
  status.
  Evidence: `TestRenderObjectDebugRecordsArtifactsAndIsolationModes` passed on
  2026-06-01 and asserts bounds plus painted status.
- [x] Confirm every painted object record includes resolved fill, stroke, text,
  image, shadow, table, and unsupported summaries.
  Evidence: `TestObjectStyleSummaryIncludesResolvedParagraphTextStyle`,
  `TestObjectStyleSummaryIncludesShadowParameters`,
  `TestObjectStyleSummaryIncludesCustomPathDetails`,
  `TestObjectStyleSummaryIncludesImageAndTableProperties`, and
  `TestPaintedObjectRecordIncludesUnsupportedItems` passed on 2026-06-01.
- [x] Confirm normal debug mode preserves production render pixels.
  Evidence: `TestRenderObjectDebugNormalModeDoesNotChangePixels` passed on
  2026-06-01.
- [x] Confirm background-plus-before-target mode works.
  Evidence: `TestRenderObjectDebugRecordsArtifactsAndIsolationModes` passed on
  2026-06-01.
- [x] Confirm target-object-only mode works on transparent background.
  Evidence: `TestRenderObjectDebugRecordsArtifactsAndIsolationModes` passed on
  2026-06-01.
- [x] Confirm target-object-only mode works on flat background.
  Evidence: `TestRenderObjectDebugRecordsArtifactsAndIsolationModes` passed on
  2026-06-01.
- [x] Confirm objects-through-target mode works.
  Evidence: `TestRenderObjectDebugRecordsArtifactsAndIsolationModes` passed on
  2026-06-01.
- [x] Confirm artifact output includes per-slide got/reference/diff PNGs,
  per-object PNGs, object attribution JSON, and ownership summary JSON.
  Evidence: `TestWriteRealWorldDiffArtifactsWritesMetadata` and
  `TestRenderMicroFixtureWithObjectDebugWritesFixtureRecords` passed on
  2026-06-01; ownership summary command
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31 PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=/Users/artpar/workspace/code/puppt/testdata/realworld-ppts/render-artifacts/object-debug-2026-05-31/micro-fixture-ownership-summary-current.json go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v`
  passed with 170 total scoped manifests and 70 clean failures.
- [x] Run focused attribution harness tests.
  Evidence: focused Phase 2 test command passed on 2026-06-01.
- [x] Record Phase 2 changed files, residual risk, and next checkpoint.
  Evidence: Phase 2 changed files include `internal/render/render_object_debug.go`,
  `internal/render/render_object_debug_test.go`, `internal/render/render_paint.go`,
  `internal/render/render_types.go`, `internal/render/render_realworld_test.go`,
  and artifact/checklist docs; residual risk is that Phase 2 proves harness
  capability, not renderer visual parity; next checkpoint is Phase 3 ownership
  ranking from current artifacts. 2026-06-01 harness hardening also made
  repo-relative diagnostic `*_OUTPUT` paths resolve from the test package cwd,
  after the new picture diagnostics exposed that writes were less robust than
  reads.

## Phase 3: Rank Real Failures By Ownership

Goal: select source-attributed clean object failures from current artifacts
before changing renderer primitives.

- [x] Regenerate current real-world object artifacts against Apple Notes
  references.
  Evidence: `PUPPT_RUN_REALWORLD_RENDER_TESTS=1 PUPPT_REALWORLD_ARTIFACT_DIR=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 go test ./internal/render -run TestRealWorldGoldenComparison -count=1` ran on 2026-06-01; expected parity failure was 61/61 slides, 9,321,023 differing pixels, no unsupported gaps.
- [x] Generate the ownership summary from current artifacts.
  Evidence: `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT=/Users/artpar/workspace/code/puppt/testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/micro-fixture-ownership-summary.json go test ./internal/render -run TestMicroFixtureTargetOwnershipSummary -count=1 -v` passed on 2026-06-01 with 170 scoped manifests, 70 clean failures, 73 contaminated failures, and 9 partial-underpaint failures.
- [x] Confirm clean-failure classification separates later-object occlusion and
  partial-alpha underpaint contamination from standalone target evidence.
  Evidence: focused ownership classification passed on 2026-06-01; refreshed
  clean-failure ownership reports 1,200 inside-object pixels, 0 outside-object
  pixels, 0 partial-alpha-over-underpaint pixels for the selected
  picture-contour target.
- [x] Select the first clean picture-contour target.
  Required starting candidates: WHO HIV slide 015 `Picture 4`, EPA Residential
  Wood slide 004 `Google Shape;11;p15`.
  Evidence: selected `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json`; ownership summary identifies it as a clean picture-contour failure with source XML, media bytes, target mask, and deterministic crop artifacts.
- [x] Confirm the selected object has a focused manifest.
  Evidence: `manifest.json` exists for `Picture 4` and names deck, slide, object, fixture path, source XML path, source image metadata, sampling metadata, crops, diffs, and target scope.
- [x] Confirm source XML is preserved in the fixture.
  Evidence: `source-object.xml` preserves raw `<p:pic>` with cNvPr id `1028`, name `Picture 4`, `r:embed="rId5"`, empty `a:srcRect`, `a:stretch/a:fillRect`, transform, `prstGeom rect`, `a:noFill`, and `bwMode="auto"`.
- [x] Confirm visible crop/reference/diff artifacts exist.
  Evidence: `got-crop.png`, `reference-crop.png`, `micro-diff.json`, `got-geometry-crop.png`, `reference-geometry-crop.png`, `geometry-diff.json`, and `target-scope.json` exist under the selected fixture directory.
- [x] Confirm residual profile is tied to source fields.
  Evidence: manifest records `ppt/media/image17.png` as a 200x200 PNG, `rId5`, geometry bounds 677..788 x 360..470, changed-output crop 699..788 x 360..451, fractional geometry 111.789055 x 111.789055, and source-to-geometry scale 0.558945 x 0.558945.
- [x] Add or update the experiment-log entry for the selected target.
  Evidence: `docs/RENDERER_EXPERIMENT_LOG.md` updated on 2026-06-01 with fresh Phase 3 artifact and ownership results.
- [x] Record Phase 3 changed files, residual risk, and next checkpoint.
  Evidence: Phase 3 changed files are artifact/checklist/experiment-log updates; residual risk is no renderer primitive has been fixed yet; next checkpoint is Phase 4 fixture extraction audit for the selected object.

## Phase 4: Complete Micro-Fixture Extraction

Goal: every selected failure is reducible to a deterministic fixture containing
only the target object and required dependencies.

- [x] Preserve raw source XML for the target object, or document and prove an
  equivalent extractor transformation.
  Evidence: selected `Picture 4` fixture writes `source-object.xml` containing the raw source `<p:pic>` object.
- [x] Preserve meaningful relationship ids.
  Evidence: source XML and fixture relationships preserve `rId5` for the picture media relationship.
- [x] Preserve theme and color-map dependencies.
  Evidence: selected `Picture 4` has no theme/color-map dependency in its extracted object semantics; fixture manifest lists only required OPC, presentation, slide, relationship, and media parts.
- [x] Preserve layout/master dependencies needed for renderer semantics.
  Evidence: selected `Picture 4` fixture has no required layout/master dependency; `fixture.pptx` contains 7 parts and renders from the extracted slide plus media only.
- [x] Preserve media bytes and image metadata.
  Evidence: manifest records source media `ppt/media/image17.png`, PNG, 200x200; fixture contains `ppt/media/object.png` with SHA-256 `c60df9328e69b020494c156265fc1c23ca004bf68b0fddc45a656552bae08bd9`.
- [x] Include visible-mask handling for later-object occlusion.
  Evidence: selected `Picture 4` manifest includes `target-scope.json`; target scope reports 8,280 compared pixels, 8,280 full-alpha object-mask pixels, and 1,200 differing pixels inside the object mask.
- [x] Keep generated fixture artifacts deterministic.
  Evidence: `unzip -l fixture.pptx` shows all fixture ZIP entries timestamped `00-00-1980 00:00`; manifest records deterministic part SHA-256 values.
- [x] Verify the fixture acceptance target is object crop/reference, not the
  whole slide.
  Evidence: manifest names `got_crop_path`, `reference_crop_path`, and `diff_path`; verifier compares `got-crop.png` to `reference-crop.png`.
- [x] Run the focused micro-fixture verifier for the selected object.
  Evidence: `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v` failed as expected with 1,200 crop differing pixels, bounds x=0..67 y=18..91.
- [x] Record Phase 4 changed files, residual risk, and next checkpoint.
  Evidence: Phase 4 changed files are selected fixture artifacts plus checklist/log updates; residual risk is the fixture still fails by 1,200 pixels; next checkpoint is Phase 5.1 source-backed picture contour coverage analysis before any production edit.

## Phase 5: Fix Renderer Primitives In Failure-Family Order

Goal: production renderer changes are source-backed, fixture-proven, and
accepted only after the full corpus gate.

### 5.1 Opaque Grayscale Picture Contour Coverage

- [x] Read authoritative OOXML for `Picture 4` and its media relationship.
  Evidence: inspected fresh `source-object.xml` and `slide15.xml.rels` on 2026-06-01; source object is `<p:pic>` cNvPr id `1028`, name `Picture 4`, `r:embed="rId5"`, empty `a:srcRect`, `a:stretch/a:fillRect`, transform `x=8595453 y=4567248 cx=1419721 cy=1419721`, `prstGeom rect`, `a:noFill`, `bwMode="auto"`; `rId5` resolves to `ppt/media/image17.png`.
  The maintained ECMA-376 bundle confirms picture objects contain `nvPicPr`,
  `blipFill`, and `spPr` in
  `docs/specs/ecma-376/part1/schema/strict/dml-picture.xsd:14-21`;
  `CT_BlipFillProperties` contains optional `blip`, optional `srcRect`, and an
  optional fill mode in
  `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1502-1509`; `fillRect`
  belongs to stretch fill mode at lines 1455-1464; and `CT_RelativeRect`
  crop/fill values default to zero at lines 648-652. The Microsoft
  `a14:useLocalDpi` and `a14:hiddenFill` extensions are recorded in
  `docs/specs/ms-odrawxml/README.md`; current evidence does not support either
  as a visible raster-rendering fix for this opaque source PNG.
- [x] Define expected primitive behavior from Open XML/source media.
  Evidence: expected primitive for this fixture is full-source 200x200 PNG scaled into the authored square picture geometry, with no source crop, no rotation, no custom mask, no soft edge, no line, no shadow, and Display P3 output conversion after rasterization; residual is therefore picture contour/resampling/source-color behavior, not layout, crop, mask, text, shadow, or occlusion.
- [x] Tighten or add the micro-fixture test before production edits.
  Evidence: `TestMicroFixtureManifestComparison` is the focused pre-edit fixture gate for the selected object and currently fails with the tracked 1,200-pixel crop residual.
- [x] Inspect the current production picture render path before editing.
  Evidence: inspected `renderPicture`, `pictureSourceImage`, `pictureSourceForElement`, `scaleImage`, `pictureScaler`, and `applyDisplayP3OutputTransform` on 2026-06-01; current path decodes PNG, applies no crop/effects for this object, scales with `xdraw.ApproxBiLinear`, then writes final output through the normal Display P3 transform.
- [ ] Implement a coherent source-backed primitive change.
  Evidence: not complete. 2026-06-01 hard-edge smoothing diagnostic was rejected
  because it did not pass `Picture 4` and did not improve the neighboring
  `Google Shape;11;p15` fixture; source-hard-edge smoothing and refreshed
  sampling-phase diagnostics were also rejected because neither passed both
  object fixtures. Additional source-model, transfer/gamma, kernel, and area
  resampling diagnostics were run on 2026-06-01; best `Picture 4` result was
  `converted_icc/cubic_sharp/floor_ceil` at 1,119 differing pixels, while the
  neighboring EPA `Google Shape;11;p15` fixture still stayed at 2,121+ differing
  pixels. A source-backed fractional DrawingML bounds diagnostic was also run;
  it lowered aggregate channel error but still left `Picture 4` at 1,173 pixels
  and `Google Shape;11;p15` at 2,113 pixels. No production picture renderer
  change accepted. Current Phase 5.1 boundary: do not add another broad image
  scaler, transfer, metadata, or extension experiment unless it is tied to a
  new source-backed contour reconstruction model; continue other failure
  families while this picture primitive remains open.
- [ ] Run the focused `Picture 4` fixture.
  Evidence: current focused fixture command still fails:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  reports 1,200 crop differing pixels. Rerun residual diagnostics still report
  1,200 grayscale edge-coverage differing pixels, source image 200x200 with 39
  unique opaque grayscale colors, 40,000 opaque pixels, and 0 alpha pixels.
  PNG metadata profile reports an 8-bit indexed-color PNG with chunks
  `IHDR/PLTE/IDAT/IEND`, 39 palette entries, no `tRNS`, no `gAMA`, no `sRGB`,
  no `iCCP`, and no `pHYs`.
- [ ] Run the neighboring picture contour fixture for `Google Shape;11;p15`.
  Evidence: current neighboring fixture command still fails:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  reports 2,127 visible-crop differing pixels. PNG metadata profile reports an
  8-bit indexed-color PNG with chunks `IHDR/PLTE/IDAT/IEND`, 256 palette
  entries, no `tRNS`, no `gAMA`, no `sRGB`, no `iCCP`, and no `pHYs`.
- [ ] Run the full 61-slide Apple Notes gate.
  Evidence:
- [ ] Record accepted or rejected decision in `docs/RENDERER_EXPERIMENT_LOG.md`.
  Evidence: 2026-06-01 Picture 4/Google Shape fixture rerun and extension-spec
  rejection recorded in the experiment log; no production picture renderer
  change accepted. PNG metadata profile rejection also recorded; Phase 5.1 is
  paused with no accepted production change.

### 5.2 Rectangle Shape Edge And Stroke Coverage

- [x] Read authoritative OOXML for WHO HIV slide 012 `Rectangle 5`.
  Evidence: inspected `source-object.xml` and the fixture manifest on 2026-06-01;
  source object is `<p:sp>` cNvPr id `6`, name `Rectangle 5`, transform
  `x=0 y=1 cx=12192000 cy=996758`, `prstGeom rect`, direct solid fill
  `srgbClr val="0070C0"`, style `lnRef idx="2"` with `accent1` shade 50000,
  and text body `anchor="ctr"` containing bold 40pt `Ordering Test Kits` with
  leading/trailing source spaces.
- [x] Define expected primitive behavior for rectangle edge/stroke coverage.
  Evidence: expected primitive is a full-slide-width solid rectangle with direct
  blue fill, style-derived 12700 EMU accent stroke, no shadow/effects, no
  rotation, and centered text in the default inset text box; residual therefore
  splits into rectangle fractional edge/stroke coverage and centered text
  placement/font metrics, not layout, occlusion, crop, fill color, or unsupported
  effects.
- [x] Tighten or add the micro-fixture test before production edits.
  Evidence: `TestMicroFixtureManifestComparison` is the focused fixture gate for
  `shape-0001-6-Rectangle-5/manifest.json` and currently fails with 7,423
  visible-crop differing pixels across x=0..959 y=0..78.
- [x] Inspect the current production shape/stroke render path before editing.
  Evidence: inspected `renderShape`, `fillShapeRectWithFloatBounds`,
  `drawStyledRectOutlineAlignedWithCap`, `alignedStrokeRect`,
  `drawShapeTextForElement`, `drawShapeTextWithDPI`, `anchoredTextTop`, and
  `textBounds` on 2026-06-01; current path fills rects using fractional coverage,
  draws rect outlines from the snapped integer target, then renders centered
  text after applying default text insets and font metrics.
- [ ] Implement a coherent source-backed primitive change.
  Evidence: not complete. 2026-06-01 fractional rectangle-outline candidate was
  implemented locally and immediately rejected because the Rectangle 5 fixture
  still failed with 7,423 differing pixels; it improved absolute channel error
  but did not satisfy the object-fixture acceptance rule, so the production
  change was reverted.
- [ ] Run the focused rectangle fixture.
  Evidence: current focused fixture command still fails:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  reports 7,423 visible-crop differing pixels.
- [ ] Run neighboring shape-edge fixtures.
  Evidence:
- [ ] Run the full 61-slide Apple Notes gate.
  Evidence:
- [ ] Record accepted or rejected decision in `docs/RENDERER_EXPERIMENT_LOG.md`.
  Evidence: 2026-06-01 Rectangle 5 source/path audit, profile diagnostics, and
  rejected fractional-outline candidate recorded in the experiment log.

### 5.3 Centered Text Vertical Placement And Font Metrics

- [x] Read authoritative OOXML for WHO HIV slide 012 `Rectangle 5` text body.
  Evidence: inspected the fixture `source-object.xml`; the text body is
  `<p:txBody><a:bodyPr rtlCol="0" anchor="ctr"/><a:lstStyle/><a:p><a:r><a:rPr lang="en-ES" sz="4000" b="1" dirty="0"/><a:t>                    Ordering Test Kits </a:t></a:r></a:p></p:txBody>`.
  The maintained ECMA-376 bundle confirms `txBody` contains `bodyPr`, optional
  `lstStyle`, and one or more paragraphs in
  `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:2653`; `bodyPr@anchor`
  is part of `CT_TextBodyProperties` at lines 2625-2652; and `ctr` is a valid
  `ST_TextAnchoringType` value at lines 2547-2555.
- [x] Define expected primitive behavior for anchoring, line boxes, and font
  metrics.
  Evidence: expected primitive is a single horizontal text body, default text
  box insets because no `lIns/tIns/rIns/bIns` are authored, centered vertical
  anchoring because `anchor="ctr"` is authored, bold 40pt text from the run
  properties, and theme minor-latin Calibri from the style font reference.
  The forward path is to test whether `anchor="ctr"` should center the
  text-body line box/paragraph box rather than shifting only detected glyph
  pixels; a raw `-4px` glyph-mask shift is not source-backed enough to land.
- [x] Tighten or add the micro-fixture test before production edits.
  Evidence: `TestMicroFixtureManifestComparison` is already the object fixture
  gate for `shape-0001-6-Rectangle-5/manifest.json` and currently fails by
  7,423 visible-crop pixels; `TestMicroFixtureShapeTextStrokeProfile` provides
  the pre-edit text/stroke residual profile and shows got text mask
  x=190..479 y=26..59 versus reference x=190..479 y=22..55.
- [x] Inspect the current production text layout path before editing.
  Evidence: inspected `parseBodyProperties`, `paragraphTextRunsWithTheme`,
  `drawShapeTextWithDPI`, `measureTextRenderLines`,
  `measuredTextAnchorHeight`, `anchoredTextTop`, and `textBounds` on 2026-06-01.
  Current code parses `anchor="ctr"`, applies DrawingML default insets, measures
  text with the resolved font, and centers using `measuredTextAnchorHeight`,
  which currently uses visible ascent/descent for centered/bottom anchoring
  rather than the full line box height.
- [ ] Implement a coherent source-backed primitive change.
  Evidence: not complete. Required next step is a source-backed line-box anchor
  diagnostic that renders the same parsed text body with full-line-box centered
  placement and reports object-fixture, edge-oracle, and neighboring-fixture
  impact before any production text-layout change is accepted. 2026-06-01
  diagnostic added to `TestMicroFixtureShapeTextStrokeProfile`; it reports
  parsed line metrics and compares `current-visible-anchor` with
  `line-box-anchor` candidates. Rectangle 5 measured one line with ascent 39,
  descent 11, current anchor height 50, line-box anchor height 50, and
  line-box shift 0 px, so the line-box rule is not an explanatory production
  change for the observed reference text shift.
- [ ] Run the focused rectangle text fixture.
  Evidence: current focused fixture command still fails:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/shape-0001-6-Rectangle-5/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  reports 7,423 visible-crop differing pixels.
- [ ] Run neighboring text-anchor/font fixtures.
  Evidence: diagnostic profile was also run for neighboring same-family
  centered text fixtures on WHO HIV slide 010 `shape-0002-6-Rectangle-5` and
  slide 009 `shape-0001-6-Rectangle-5`. Both had current anchor height equal
  to line-box anchor height and line-box shift 0 px, so the candidate had no
  beneficial neighboring signal.
- [ ] Run the full 61-slide Apple Notes gate.
  Evidence:
- [ ] Record accepted or rejected decision in `docs/RENDERER_EXPERIMENT_LOG.md`.
  Evidence: 2026-06-01 line-box anchor diagnostic and rejected production
  decision recorded in the experiment log.

### 5.4 TextBox Fill Height, Text Antialias, And Anchor Behavior

- [x] Read authoritative OOXML for WHO HIV slide 015 `TextBox 7`.
  Evidence: inspected `source-object.xml` and fixture manifest on 2026-06-01;
  source object is `<p:sp>` cNvPr id `8`, name `TextBox 7`, `p:cNvSpPr
  txBox="1"`, transform `x=1191129 y=1468901 cx=4728410 cy=646331`,
  `prstGeom rect`, solid fill `schemeClr accent5` with `lumMod=20000` and
  `lumOff=80000`, and text body `wrap="square"` with `<a:spAutoFit/>`, two
  centered bold paragraphs, and direct run text color `srgbClr val="0070C0"`.
  The maintained ECMA-376 bundle confirms color choices include `schemeClr` and
  `srgbClr` at `dml-main.xsd:667-680`, fill properties include `solidFill` at
  lines 1577-1590, text autofit includes `spAutoFit` at lines 2610-2624, and
  text body properties/content are at lines 2625-2659.
- [x] Define expected primitive behavior and split mixed residual components.
  Evidence: expected primitive is a rectangular text box with theme-derived
  accent5 luminance-adjusted fill, no authored line/shadow/effects, shape
  autofit affecting text-box dimensions, centered paragraph text in the default
  inset text box, Calibri bold text, and direct blue run color. Current residual
  is mixed fill color, painted height, text coverage, and antialias/metrics; it
  is not a single text-anchor or color-only primitive.
- [x] Tighten or add micro-fixture tests before production edits.
  Evidence: `TestMicroFixtureManifestComparison` is the focused object fixture
  gate and currently fails with 19,868 crop differing pixels. Existing opt-in
  diagnostics `TestMicroFixtureShapeObjectProfile`,
  `TestMicroFixtureShapeFillHeightSearch`,
  `TestMicroFixtureShapeResidualTextProfile`, and
  `TestMicroFixtureShapeLuminanceColorSearch` cover this mixed residual before
  production edits.
- [x] Inspect the current production fill/text/anchor paths before editing.
  Evidence: inspected `parseBodyProperties`, `renderShape`,
  `shapeAutofitTarget`, `fillShapeRectWithFloatBounds`,
  `drawShapeTextForElement`, `textBounds`, and `drawShapeTextWithDPI` on
  2026-06-01. Current path parses `spAutoFit`, measures text, expands the shape
  target from y=166 to y=169, paints the expanded fill, then draws text in the
  default inset bounds.
- [ ] Implement only one source-backed primitive change at a time.
  Evidence: not complete. Existing focused diagnostics show geometry target
  x=94..465 y=116..166, shape-autofit text target x=94..465 y=116..169,
  measured text 351x46, fill #DEEBF7/FF, dominant got fill #E0EBF6/FF,
  dominant reference fill #E1EBF5/FF, and reference white rows. Fill/height
  normalization to #E1EBF5/FF at 49px improves from 19,868 to 7,347 pixels but
  still fails; luminance color search shows the current color formula is within
  one channel of the reference dominant fill. A 2026-06-01 parsed-source-text
  diagnostic then redrew the extracted DrawingML text body over the normalized
  fill/height candidate using both source geometry text bounds and current
  `spAutoFit` text bounds; both candidates worsened the focused object from
  7,347 to 7,692 differing pixels, so no source-backed parsed-text redraw or
  geometry-versus-autofit bounds change is accepted. No production TextBox
  renderer change accepted.
- [ ] Run the focused TextBox fixture.
  Evidence: current focused fixture command still fails:
  `PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/shape-0003-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  reports 19,868 crop differing pixels, bounds x=0..371 y=0..53.
- [ ] Run neighboring TextBox/text fixtures.
  Evidence: partial diagnostic evidence recorded. Neighboring WHO HIV slide 003
  `TextBox 7` focused fixture still fails with 132,995 visible-crop differing
  pixels. Its residual profile normalizes to 37,520 differing pixels, and both
  parsed-source-text candidates worsen to 38,022 differing pixels. This is a
  rejection signal for the parsed-text redraw hypothesis, not an accepted
  production change.
- [ ] Run the full 61-slide Apple Notes gate.
  Evidence:
- [ ] Record accepted or rejected decision in `docs/RENDERER_EXPERIMENT_LOG.md`.
  Evidence: 2026-06-01 TextBox 7 fill/height, luminance color, residual text,
  parsed-source-text, and neighboring TextBox rejection recorded in the
  experiment log; no production TextBox renderer change accepted.

### 5.5 Production Backend Path And Scoreboard

- [x] Generate a reusable primitive failure scoreboard from current artifacts.
  Evidence: `PUPPT_RENDERER_SCOREBOARD_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_RENDERER_SCOREBOARD_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/renderer-production-scoreboard.json go test ./internal/render -run TestRendererProductionFailureScoreboard -count=1 -v`
  passed on 2026-06-01. The scoreboard reports 61 slides, 9,321,023 slide
  differing pixels, 61 attribution artifacts, and 70 clean object-fixture
  failures.
- [x] Rank systemic renderer gaps from object attribution, not whole-slide
  totals alone.
  Evidence: `renderer-production-scoreboard.json` ranks object-overlap groups
  as shape geometry/fill/line/clipping/antialiasing first, text
  shaping/font/paragraph/anchoring second, picture crop/resampling/color/media
  third, connector geometry fourth, table layout fifth, and shadow sixth.
- [x] Rank isolated clean fixture families from current micro-fixtures.
  Evidence: current scoreboard reports 46 clean picture fixture failures
  totaling 1,499,584 differing pixels and 24 clean shape fixture failures
  totaling 550,448 differing pixels. Largest clean families are `Picture 2`,
  `Picture 5`, `TextBox 7`, `Rectangle 5`, and `Rectangle 3`.
- [x] Record the production backend path in maintained project docs.
  Evidence: `docs/RENDERER_PRODUCTION_PATH.md` added on 2026-06-01 and linked
  from `docs/RENDERER_COMPLETION_GOAL.md`; dependency references added to
  `docs/REFERENCES.md`.
- [x] Add a test-only vector backend spike for the first shape family.
  Evidence: added `TestMicroFixtureShapeVectorBackendProfile` using the
  controlled `github.com/llgcode/draw2d` primitive dependency behind a
  diagnostic adapter. The adapter renders Puppt-parsed rectangle fill/stroke,
  converts colors to the existing Display P3 output space, redraws parsed text
  with Puppt's current text renderer, and compares object crops without changing
  production rendering.
- [x] Accept or reject the draw2d rectangle backend for production.
  Evidence: not accepted. Current focused/same-family diagnostic results:
  slide 012 `Rectangle 5` 7,423 -> 7,421 pixels, slide 010 `Rectangle 5`
  13,320 -> 13,306 pixels, slide 009 `Rectangle 5` 18,027 -> 18,000 pixels,
  and slide 013 `TextBox 3` 25,347 -> 25,233 pixels. This is a consistent
  signal that the adapter is wired correctly, but it does not pass any fixture
  and is not production-ready. A follow-up layer split showed the rectangle
  fill/stroke layer helps more than the current-text composite, so the next
  accepted production change should not be a blind draw2d replacement.
- [x] Add a test-only text shaping backend spike for the first text family.
  Evidence: added `TestMicroFixtureShapeTextShapingProfile` using
  `github.com/go-text/typesetting` behind a diagnostic adapter that consumes
  Puppt-resolved text runs and font bytes. Focused probes passed on 2026-06-01:
  slide 012 `Rectangle 5` max advance delta 1 px, slide 015 `TextBox 7` max
  delta 2 px, and slide 013 `TextBox 3` max delta 5 px. The spike does not
  justify replacing production text rendering for these targets; residuals are
  more likely in text placement, line metrics, and raster/composite behavior
  than in missing HarfBuzz advance shaping alone.
- [x] Split picture rendering diagnostics into source decode, color, crop,
  transform, sampling, and output stages.
  Evidence: added `TestMicroFixturePicturePipelineProfile`, which opens an
  attributed picture micro-fixture, resolves the same source relationship and
  element as production rendering, then records source decode, Display P3 color,
  `srcRect` crop, flip/alphaModFix transform, scaler/target sampling, and final
  output stages. Focused profile runs passed on 2026-06-01 for WHO slide 015
  `Picture 4` and EPA slide 004 `Google Shape;11;p15`. The staged output
  exactly reproduced current production crops for both fixtures
  (`diff_against_got=0`), while still differing from references by 1,200 and
  2,127 pixels respectively. This proves the residual is inside the documented
  picture pipeline, not fixture drift or diagnostic mismatch.
- [x] Start the renderer IR/backend boundary required for production
  completion.
  Evidence: added `internal/render/render_scene.go` and
  `internal/render/render_scene_test.go` on 2026-06-01. The first boundary
  lowers resolved picture objects into `renderPicturePrimitive`, preserving
  source relationship, media part/content type, integer and fractional target
  bounds, crop, flip, `alphaModFix`, rotation, `rotWithShape`, soft edge, custom
  mask, and line metadata. Focused tests
  `TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields` and
  `TestRenderSceneFromElementsKeepsPictureZOrderAndErrors` passed.
- [x] Move current picture painting behind the render-scene picture backend
  without changing pixels.
  Evidence: `renderPicture` now lowers the resolved picture object into
  `renderPicturePrimitive` and calls `currentPictureBackend` through the
  `pictureBackend` interface before painting. The primitive lowering covers
  both `<p:pic>` objects and picture-backed shape blip fills because production
  `renderPicture` serves both call sites. The migration is intentionally
  zero-diff. Initial focused picture tests
  `go test ./internal/render -run 'TestRenderPicture|TestPicture|TestDrawPictureRaster|TestRenderPaintsEmbeddedPNGPicture' -count=1`
  passed. The pipeline profile reruns for WHO slide 015 `Picture 4` and EPA
  slide 004 `Google Shape;11;p15` both reported `got_delta=0`, with unchanged
  reference residuals of 1,200 and 2,127 pixels respectively.
- [x] Remove legacy `slideElement` dependency from `pictureBackendInput` by
  promoting remaining picture paint fields into `renderPicturePrimitive`.
  Evidence: `pictureBackendInput` no longer carries `*slideElement`; the
  backend now paints from `renderPicturePrimitive` plus decoded source image,
  target part, canvas, and slide size. Promoted fields include object kind,
  SVG relationship id, crop, flip, `alphaModFix`, rotation/`rotWithShape`,
  soft edge, custom mask path/commands/unsupported messages, line style,
  shadow parameters, and 3-D unsupported feature metadata. Focused primitive
  and picture/blip-fill tests passed, and fixture pipeline profiles for WHO
  slide 015 `Picture 4` and EPA slide 004 `Google Shape;11;p15` still reported
  `got_delta=0`.
- [x] Extract picture sampling/color as a replaceable backend stage.
  Evidence: added `pictureSamplingStage`, `pictureSamplingInput`, and
  `currentPictureSamplingStage`; `currentPictureBackend` now calls the sampling
  stage with the primitive, target, source image, source bounds, canvas, slide
  size, and output width. `TestCurrentPictureBackendUsesSamplingStage` proves
  the backend invokes the stage. Focused picture/blip-fill tests passed, and
  fixture pipeline profiles for WHO slide 015 `Picture 4` and EPA slide 004
  `Google Shape;11;p15` still reported `got_delta=0`.
- [x] Install the fixture-backed acceptance gate for picture sampling stage
  replacement.
  Evidence: added `TestCurrentPictureSamplingStageAcceptanceGate`, which renders
  WHO slide 015 `Picture 4` and EPA slide 004 `Google Shape;11;p15` through the
  `pictureSamplingStage` backend hook and measures the same crop/visible-crop
  residuals used by the maintained micro-fixture manifests. Explicit validation:
  `PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE=1 go test ./internal/render -run TestCurrentPictureSamplingStageAcceptanceGate -count=1 -v`
  passed with the recorded current residuals of 1,200 and 2,127 pixels. This is
  the promotion gate for a replacement stage; it is not a visual parity fix.
- [x] Replace one picture backend stage with a fixture-proven implementation.
  Evidence: M07 extends the source transform/backend contract for pictures on
  2026-06-01. `renderPicturePrimitive` now carries linked image relationships,
  blip fill stretch/tile metadata, and concrete source-space blip effects.
  The current backend renders alphaModFix, alphaBiLevel, alphaCeiling,
  alphaFloor, alphaInv, alphaRepl, biLevel, clrChange, clrRepl, duotone,
  grayscl, lum, hsl, tint, simple blur, fillOverlay, scalar-container
  alphaMod, signed `srcRect` crop/padding, and default tiled image fills.
  Non-scalar `alphaMod` containers are reported as per-object partials instead
  of silently skipped.
  Focused synthetic M07 tests and the milestone picture regex passed.
- [x] Record accepted/rejected backend decisions in
  `docs/RENDERER_EXPERIMENT_LOG.md`.
  Evidence: draw2d rectangle backend rejection, go-text shaping diagnostic
  decision, picture pipeline split evidence, and M07 source-backed blip/effect
  decisions are recorded in `docs/RENDERER_EXPERIMENT_LOG.md`. The picture
  sampling gate now accepts true zero-diff replacements directly and only
  allows the current `Picture 4`/`Google Shape;11;p15` residuals with explicit
  source-backed reasons.

### 5.6 ICC-Profiled Full-Slide Picture Source Correspondence

- [x] Read authoritative OOXML for WHO HIV slide 009 `Picture 2`.
  Evidence: inspected
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/source-object.xml`.
  The source object is `<p:pic>` cNvPr id `3`, name `Picture 2`, with
  `a:blip r:embed="rId4"`, `a:stretch/a:fillRect`, transform
  `x=0 y=1335505 cx=12192000 cy=5233737`, and rectangular preset geometry.
- [x] Confirm source media metadata before changing the picture pipeline.
  Evidence:
  `PUPPT_PICTURE_PNG_METADATA_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json PUPPT_PICTURE_PNG_METADATA_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/picture-png-metadata.json go test ./internal/render -run TestMicroFixturePicturePNGMetadataProfile -count=1 -v`
  passed. The source PNG is 2830x820 truecolor-alpha, has an embedded ICC
  profile named `ICC Profile`, and has `pHYs` 5669x5669 pixels per meter.
- [x] Split the current picture pipeline for the gate-relevant clean picture
  failure.
  Evidence:
  `PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/picture-pipeline-profile.json go test ./internal/render -run TestMicroFixturePicturePipelineProfile -count=1 -v`
  passed with `got_delta=0` and `reference_delta=154741`. The profile records
  source decode as 2830x820, 1157 unique colors, all opaque pixels; current
  Display P3 output conversion changes 446,036 source pixels with absolute
  delta 2,200,568 before sampling/output comparison.
- [x] Test source-model candidates before accepting a production picture change.
  Evidence:
  `PUPPT_PICTURE_SOURCE_MODEL_SEARCH_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json PUPPT_PICTURE_SOURCE_MODEL_SEARCH_OUTPUT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/picture-source-model-search.json go test ./internal/render -run TestMicroFixturePictureSourceModelSearch -count=1 -v`
  passed. The best candidate is the current source-backed path,
  `converted_icc/approx_bilinear/floor_floor`, still with 154,741 differing
  pixels. This rejects a production change based only on PNG ICC/pHYs metadata.
- [x] Profile the remaining picture residual.
  Evidence:
  `PUPPT_PICTURE_RESIDUAL_PROFILE_MANIFEST=.../0003-3-Picture-2/manifest.json PUPPT_PICTURE_RESIDUAL_PROFILE_OUTPUT=.../0003-3-Picture-2/picture-residual-profile.json go test ./internal/render -run TestMicroFixturePictureResidualProfile -count=1 -v`
  and
  `PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_MANIFEST=.../0003-3-Picture-2/manifest.json PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_OUTPUT=.../0003-3-Picture-2/picture-source-correspondence-profile.json go test ./internal/render -run TestMicroFixturePictureSourceCorrespondenceProfile -count=1 -v`
  passed. Residual profile reports 154,741 differing pixels, including 112,436
  grayscale/edge-coverage pixels; source correspondence reports source bounds
  x=1..2828 y=0..819, 107,787 mixed 3x3 source-neighborhood pixels, and 48,275
  nearest-source antialias pixels.
- [ ] Accept a production picture source/sampling change for this family.
  Evidence: not complete. The current ICC-aware path is already the best
  source-model candidate for this fixture, and the residual remains a full
  source/sampling correspondence gap. No production picture renderer change is
  accepted from this audit.

### 5.7 Source-Backed Shape Color Candidate Audit

- [x] Read authoritative OOXML for WHO HIV slide 007 `Rectangle 7`.
  Evidence: inspected
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-007/micro-fixtures/shape-0005-8-Rectangle-7/source-object.xml`.
  The source object is `<p:sp>` cNvPr id `8`, name `Rectangle 7`, with
  rectangular geometry, `accent5` fill using `lumMod=20000` and
  `lumOff=80000`, `0070C0` solid line at width `19050`, and centered
  auto-numbered text.
- [x] Profile source color, fill replacement, and text/stroke residuals before
  changing shape rendering.
  Evidence:
  `PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json ... go test ./internal/render -run TestMicroFixtureShapeObjectProfile -count=1 -v`,
  `PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json ... go test ./internal/render -run TestMicroFixtureShapeLuminanceColorSearch -count=1 -v`,
  `PUPPT_SHAPE_FILL_HEIGHT_SEARCH_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json ... go test ./internal/render -run TestMicroFixtureShapeFillHeightSearch -count=1 -v`, and
  `PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=.../shape-0005-8-Rectangle-7/manifest.json ... go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v`
  passed.
- [ ] Accept a production shape color/text change for this fixture family.
  Evidence: not complete. The fixture fails with 56,812 differing pixels.
  Replacing the fill with the reference dominant `#E1EBF5/FF` would reduce it
  to 17,197 pixels, but the source-backed luminance search ranks the current
  `current-hsl` output `#E0EBF6/FF` as the best candidate and does not produce
  the exact reference color. No fixture-bucket color override is accepted.

### 5.8 Source-Backed Symbol Bullet Mapping

- [x] Read authoritative OOXML for WHO HIV slide 002 `Rectangle 11`.
  Evidence: inspected the object fixture for
  `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`, slide 2,
  `<p:sp>` cNvPr id `12`, name `Rectangle 11`. The source contains
  `a:buFont typeface="Wingdings"` with `a:buChar char="§"` for level-1
  bullets. The reference crop renders those bullets as solid square bullets.
- [x] Implement deterministic static mapping for the observed Office symbol
  bullet.
  Evidence: Wingdings `§` now normalizes to Unicode `▪` before any local
  private-use Wingdings path, and known Unicode-mapped symbol bullets render
  with paragraph/generic text font selection rather than a local Wingdings
  private-use glyph.
- [x] Run focused bullet parsing and layout tests.
  Evidence:
  `go test ./internal/render -run 'TestTextParagraphsFromNodeDetectsBulletsAndLevels|TestTextParagraphsFromNodeMapsWingdingsNotSignBullet|TestTextRenderLinesForElementAppliesBulletFontFamily|TestTextRenderLinesForElementUsesParagraphFontForBulletFontTx|TestRenderShape.*Symbol|TestTextRenderLinesForElement.*Bullet' -count=1 -v`
  passed on 2026-06-01.
- [ ] Close the WHO slide 002 `Rectangle 11` object fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect11-wingdings PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 71,260 differing pixels, improved from the tracked 71,272.
  The debug crop no longer shows missing-glyph boxes for the nested bullets,
  but bullet indentation, text placement, fill/stroke, and antialiasing parity
  still block the object fixture. This is an implemented supported text
  primitive improvement, not a reason to mark the remaining fixture unsupported.

### 5.9 Source-Backed Non-Placeholder `otherStyle` Paragraph Defaults

- [x] Read authoritative default-style OOXML for WHO HIV slide 002
  `Rectangle 11`.
  Evidence: the target shape's level-1 paragraphs omit local `marL` and
  `indent`, while the fixture master includes `p:txStyles/p:otherStyle`
  `a:lvl2pPr marL="457200" defTabSz="914400"`. The source rows are
  `pml.xsd:1412 CT_SlideMasterTextStyles`,
  `dml-main.xsd:2592 CT_TextListStyle`, and
  `dml-main.xsd:2994 CT_TextParagraphProperties`.
- [x] Implement default paragraph-style inheritance for non-placeholder shape
  text.
  Evidence: non-placeholder text shapes now receive master `otherStyle`
  paragraph defaults when local paragraph properties are absent. Explicit local
  paragraph geometry still wins, so the level-0 `marL="285750"
  indent="-285750"` paragraph in `Rectangle 11` is preserved.
- [x] Run synthetic and focused text inheritance tests.
  Evidence:
  `go test ./internal/render -run 'TestApplyInheritedTextStylesAppliesDefaultParagraphStyleToNonPlaceholderShapes|TestApplyInheritedTextStylesAppliesTitleButSkipsBodyPlaceholders|TestInheritedTextStylesUsePresentationDefaultAsBase|TestApplyInheritedTextStylesAppliesBodyParagraphMargins|TestTextRenderLinesForElementUsesHangingBulletTabStop|TestTextParagraphsFromNodeDetectsBulletsAndLevels' -count=1 -v`
  passed on 2026-06-01.
- [x] Check neighboring WHO text-shape fixtures for incidental movement.
  Evidence: after this change, slide 003 `TextBox 7` still failed at 130,250
  pixels, slide 015 `TextBox 7` still failed at 19,939 pixels, and slide 013
  `TextBox 3` still failed at 25,347 pixels, matching the prior counts.
- [x] Run the render package.
  Evidence: `go test ./internal/render -count=1` passed on 2026-06-01 after
  tightening the deterministic Unicode symbol-bullet expectation.
- [ ] Close the WHO slide 002 `Rectangle 11` object fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect11-otherstyle PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 71,231 differing pixels, improved from the prior 71,260
  after symbol-bullet mapping and from the tracked 71,272 baseline. The nested
  square bullets now align with the reference horizontally; remaining fixture
  risk is fill/stroke, vertical placement, antialiasing, and text metrics.

### 5.10 Source-Backed Default-Cap Dashed Stroke Antialiasing

- [x] Read authoritative stroke OOXML for WHO HIV slide 002 `Rectangle 11`.
  Evidence: the target shape has `a:ln w="22225"` with an omitted `cap`
  attribute and `a:prstDash val="sysDash"`. The source rows are
  `dml-main.xsd:2160 CT_PresetLineDashProperties`,
  `dml-main.xsd:2172 EG_LineDashProperties`, and
  `dml-main.xsd:2206 CT_LineProperties`.
- [x] Implement antialiased rendering for default/square-cap dashed strokes.
  Evidence: default or explicit square-cap dashed strokes now use the existing
  antialiased dash renderer instead of the legacy point-plotting path. Solid
  square-cap line rendering is unchanged.
- [x] Run focused dashed-stroke tests.
  Evidence:
  `go test ./internal/render -run 'TestRenderShapePaintsDashedRectOutline|TestRenderShapeUsesStrokeWidthForSystemDotRectOutline|TestRenderShapeHonorsExplicitFlatCapForSystemDotRectOutline|TestRenderShapeHonorsFlatLineCapOnDashedLine|TestLineDashPatternPixelsUsesDrawingMLPresetRuns|TestM06RendersCompoundConnectorAndCustomDash' -count=1 -v`
  passed on 2026-06-01.
- [x] Profile the remaining `Rectangle 11` text/stroke residual before choosing
  another primitive.
  Evidence:
  `PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT=/tmp/puppt-rect11-text-stroke-profile.json go test ./internal/render -run TestMicroFixtureShapeTextStrokeProfile -count=1 -v`
  passed and reported baseline 71,231 pixels, got text mask bounds
  x=8..581 y=13..108, reference text mask bounds x=6..576 y=6..109, edge
  residual 2,860 pixels, text-like residual 8,555 pixels, and best diagnostic
  text-mask shift `y=-6` with 71,039 pixels. This is diagnostic evidence only,
  not an accepted y-offset.
- [x] Reject using alternate `a:ea` font slots for Latin text as an object
  shortcut.
  Evidence: the source runs carry `a:ea typeface="Arial"` but no `a:latin`.
  Existing M08 tests intentionally keep Latin text from switching to alternate
  `ea`/`cs` slots, and the source model does not justify overriding the
  paragraph/default Latin font with an East Asian slot for this object.
- [ ] Close the WHO slide 002 `Rectangle 11` object fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-rect11-dashaa PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-002/micro-fixtures/shape-0004-12-Rectangle-11/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 71,231 differing pixels. The dashed-stroke change is
  accepted as source-backed stroke coverage, but it does not close this object.
  Remaining work should start from a stronger source-backed text metrics or
  paint model, not a font-slot or y-shift shortcut.

### 5.11 Rejected Character-Bullet Hanging Tab-Stop Candidate

- [x] Read authoritative bullet paragraph OOXML for WHO HIV slide 002
  `Rectangle 11`.
  Evidence: the first paragraph in the target shape has a character bullet with
  local paragraph geometry `marL="285750"` and `indent="-285750"`. The source
  rows are `dml-main.xsd:2751 CT_TextCharBullet` and
  `dml-main.xsd:2994 CT_TextParagraphProperties`.
- [x] Test the existing hanging tab-stop model against character bullets.
  Evidence: a temporary candidate made `hangingBulletTabStop` apply whenever a
  paragraph had a bullet and source margin/indent geometry, not only when it was
  an auto-number bullet. The candidate reused the existing tab-stop renderer
  and did not add object-specific x offsets.
- [x] Run focused and neighboring fixture checks.
  Evidence:
  `go test ./internal/render -run 'TestHangingBulletTabStopAppliesToCharacterBullets|TestTextParagraphsFromNodeDetectsBulletsAndLevels|TestTextParagraphsFromNodeLocalBulletChoiceBlocksStyledAutoNumber' -count=1`
  passed on 2026-06-01 while the candidate was applied. Object fixtures
  remained failing:
  `Rectangle 11` moved from 71,231 to 71,244 differing pixels,
  slide 003 `TextBox 7` moved from 130,250 to 130,392, slide 008 `TextBox 4`
  stayed 26,639, and slide 007 `Rectangle 7` stayed 56,812.
- [x] Reject and revert the production candidate.
  Evidence: the candidate worsened the exact real-world gate to 9,356,836 total
  differing pixels, up from the previous 9,341,017. Because both object
  fixtures and the exact gate moved the wrong direction, the production code,
  synthetic test, rendering docs, and matrix evidence were reverted. Post-revert
  validation restored the clean-suite shape bucket to 571,458 pixels and the
  exact real-world gate to the prior 9,341,017 total differing pixels.
- [ ] Close the WHO slide 002 `Rectangle 11` object fixture.
  Evidence: not closed. Character-bullet hanging geometry remains
  supported-scope, but this simple tab-stop generalization is not the accepted
  implementation. Remaining work should start from source-backed body inset,
  tab-stop, literal-space, and text-metrics evidence.

## Phase 6: Preserve CLI And JSON Contracts

Goal: renderer parity work does not make the command-line interface less honest
or less stable.

- [x] Run a focused supported `puppt render ... --json` check.
  Evidence:
  `go run ./cmd/puppt render testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/underpaint-shape-0001-7-Freeform-6/fixture.pptx --slide 1 --out /tmp/puppt-m12-supportedish.png --json`
  returned `schema_version=puppt.v1`, `command=render`, `status=ok`,
  `slide_number=1`, `slide_part=ppt/slides/slide1.xml`, width 960, and height
  540.
- [x] Run a focused known-partial/unsupported `puppt render ... --json` check.
  Evidence:
  `go test ./internal/cli -run 'TestRenderJSON|TestRenderJSONHonorsDPIFlag' -count=1 -v`
  passed on 2026-06-01. `TestRenderJSON` exercises a known partial render
  result and asserts `status=partial`, `unsupported` details, stable
  `puppt.v1` schema, output path, slide part, slide number, and dimensions.
- [x] Compare JSON fields against prior shape and confirm no removals or
  incompatible renames.
  Evidence: same focused CLI test run passed and covered existing render JSON
  envelope fields and DPI-dependent dimensions.
- [x] Confirm unsupported content is preserved, skipped with explanation, or
  rejected before mutation.
  Evidence: M11 preservation/reporting tests passed with
  `go test ./internal/... -run 'Test.*Preserve|Test.*Unsupported|Test.*Validate' -count=1`,
  including unsupported chart, OLE, ActiveX, media, and relationship parts.
- [x] Record Phase 6 changed files, residual risk, and next checkpoint.
  Evidence: M12 records Phase 6 evidence in this checklist and
  `docs/RENDERER_EXPERIMENT_LOG.md`. Residual risk is that the CLI schema is
  stable while final visual conformance remains blocked by exact real-world and
  clean-fixture failures.

## Phase 7: Final Completion Audit

Goal: prove completion requirement by requirement before claiming the renderer
goal is achieved.

- [x] Run `go test ./...`.
  Evidence: passed on 2026-06-02 after the M12 rectangle/picture round
  line-join propagation update.
- [x] Run `git diff --check`.
  Evidence: passed on 2026-06-01 after M12 documentation closeout edits.
- [x] Implement M12 source-backed rectangle/picture round line-join propagation.
  Evidence: `dml-main.xsd:2134 CT_LineJoinRound`,
  `dml-main.xsd:2138 EG_LineJoinProperties`, and
  `dml-main.xsd:2206 CT_LineProperties` define the authored line-join source
  semantics. Existing source objects include table border `a:round` joins, and
  the renderer already parsed line joins for shapes and pictures but dropped
  them when drawing rectangular outlines. `drawStyledRectOutlineCompound` now
  routes round-join rectangle strokes through the existing path-outline join
  renderer, and picture outlines pass `LineJoin`/`LineCompound` through the same
  helper. Focused verification:
  `go test ./internal/render -run 'TestM06|TestRenderPicture.*Outline|Test.*LineJoin|Test.*LineDash' -count=1`
  passed on 2026-06-02. `go test ./internal/render -count=1` and
  `go test ./... -count=1` also passed. This is a bounded stroke primitive fix
  and does not close M12.
- [x] Implement M12 source-backed table border line-end markers and compound
  border lines.
  Evidence: `dml-main.xsd:2096 ST_LineEndType`,
  `dml-main.xsd:2120 CT_LineEndProperties`,
  `dml-main.xsd:2206 CT_LineProperties`,
  `dml-main.xsd:2347 CT_TableCellProperties`, and
  `dml-main.xsd:2480 CT_TableCellBorderStyle` define known line marker and
  compound line metadata on table borders. `tableCellBorder` now carries
  parsed `headEnd`/`tailEnd` marker type/width/length values, normal and
  diagonal table borders draw known marker types through the existing line-end
  marker renderer, and table borders reuse the known compound-line renderer for
  `dbl`, `thickThin`, `thinThick`, and `tri`. Focused verification:
  `go test ./internal/render -run 'TestM09|TestRenderTableCellBorderPaintsKnownLineEndMarkers|TestRenderTableCellDiagonalBorderPaintsKnownLineEndMarkers|TestParseTableModelRecordsUnsupportedVisibleFeatures|TestRenderTableCellBorderPaintsDoubleCompoundLine' -count=1 -v`
  passed on 2026-06-02. Connector marker regression coverage
  `go test ./internal/render -run 'TestM06RendersSchemaLineEndMarkerTypes|TestM06ReportsUnknownLineMarkerType|TestM06RendersCompoundConnectorAndCustomDash' -count=1 -v`
  also passed. Known table border marker/compound metadata is implemented and
  is not an Unsupported record; unknown marker names remain reported as
  partial unsupported diagnostics.
- [x] Preserve shape-style `fontRef` text color over inherited default
  paragraph colors.
  Evidence: WHO HIV slide 013 object 6 `Rectangle 5` is a source-backed
  `p:sp` whose `<p:style><a:fontRef idx="minor"><a:schemeClr val="lt1"/>`
  resolves to white text. Its two authored runs have no direct fill, but the
  non-placeholder inherited default paragraph style could inject black as a
  paragraph color before rendering. M12 now prevents inherited paragraph text
  color from overriding an already-resolved element text color, and styled run
  segments inherit that element color unless they author a direct run color.
  Focused verification:
  `go test ./internal/render -run 'TestTextRenderLinesForElementAppliesElementTextColorToStyledRuns|TestApplyInheritedTextStylesDoesNotOverrideElementTextColor' -count=1`
  passed on 2026-06-02. The targeted Rectangle 5 micro-fixture now renders
  white title text and improves from 12,332 to 10,432 visible-crop differing
  pixels, but still fails due remaining text metrics and edge parity. This is
  a bounded `CT_ShapeStyle`/`CT_FontReference` text-color precedence fix and
  does not close M12.
- [x] Stop treating supported picture fill-mode metadata as expected
  unsupported records.
  Evidence: WHO HIV slide 009 `Picture 2` is a source-backed `p:pic` with
  `a:blip r:embed="rId4"` and `a:stretch/a:fillRect`. The renderer already
  lowers stretch fill into the picture sampling path, but fixture manifest
  generation previously copied every `image_effects` summary item into
  `expected_unsupported_records`, so supported metadata such as
  `fillMode=stretch` was advertised as unsupported. `ObjectStyleSummary` now
  exposes explicit `image_unsupported` and `effect_unsupported` fields, and
  manifest expected-unsupported generation reads only those real diagnostics
  plus table/custom-path unsupported records. Focused verification:
  `go test ./internal/render -run 'TestExpectedUnsupportedRecordsIgnoreSupportedImageMetadata|TestM07ParsesBlipFillModeLinkAndEffects|TestRenderOutputSupportsPicture' -count=1`
  passed on 2026-06-02. `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`,
  `go test ./internal/render -count=1`, `go test ./... -count=1`, and targeted
  `git diff --check` also passed. This is an honesty/reporting fix for
  `CT_BlipFillProperties` and `EG_FillModeProperties`; it does not close the
  remaining picture sampling residuals.
- [x] Preserve `CT_Blip@cstate` compression metadata through picture
  primitives and object debug summaries.
  Evidence: EPA Generate slide 003 `Picture 25` is a source-backed `p:pic`
  whose `a:blip` has `r:embed="rId3"` and `cstate="print"`. ECMA-376
  `CT_Blip` defines the optional `cstate` attribute at
  `docs/specs/ecma-376/part1/schema/strict/dml-main.xsd:1498-1500`, with
  `ST_BlipCompression` enumerated at `dml-main.xsd:1466-1473`. The renderer now
  parses this metadata, carries it through `renderPicturePrimitive`, and emits
  `cstate=print` in `ObjectStyleSummary.image_effects` without adding an
  `image_unsupported` diagnostic. Focused validation:
  `go test ./internal/render -run 'TestM07ParsesBlipFillModeLinkAndEffects|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestObjectStyleSummaryIncludesImageAndTableProperties' -count=1 -v`
  passed on 2026-06-02. Targeted object verification:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-picture25-cstate PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-003/micro-fixtures/0004-26-Picture-25/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  remains an expected failure at 65,347 visible-crop differing pixels, and
  `/tmp/puppt-picture25-cstate/current-object.json` records
  `image_effects=["fillMode=stretch","cstate=print"]` with no image
  unsupported entry. This closes a metadata/reporting gap only; picture
  sampling residuals remain supported-scope M12 work.
- [x] Refresh existing picture micro-fixture manifests to match supported
  image/effect metadata reporting.
  Evidence: 61 existing manifests under
  `testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01` still had
  `fillMode=stretch` in `spec_fixture.expected_unsupported_records` from the
  previous metadata conflation. A focused regression test,
  `TestMicroFixtureManifestsDoNotClassifySupportedImageMetadataAsUnsupported`,
  now rejects supported image/effect metadata shapes such as `fillMode=...`,
  `alphaModFix=...`, `rotWithShape=...`, and `softEdge=...` when they appear
  under `expected_unsupported_records`. The first pass removed
  `fillMode=stretch` from 61 manifests; the broadened pass removed
  `alphaModFix=100000` and `rotWithShape=true` from 36 manifests and
  `softEdge=203200` from 1 manifest. A direct scan now reports 0
  metadata-shaped expected unsupported records and 0 remaining expected
  unsupported records in the current object-debug manifests. Descriptive
  `image_effects` metadata is unchanged.
- [x] Recheck top M12 blockers without converting failures to Unsupported.
  Evidence: WHO slide 009 `Picture 2` was rechecked with
  `TestMicroFixturePictureFractionalBoundsSearch`; the best fractional target
  candidate worsened the exact fixture to 154,772 differing pixels versus the
  current 154,741, so no picture-bounds production change was accepted. WHO
  slide 012 `Table 3` was rerun with
  `TestMicroFixtureManifestComparison` and remains 284,470 differing pixels;
  `TestMicroFixtureTableStyleColorProfile` still shows source style fills and
  first-row border precedence are resolved, leaving no source-backed table
  layout primitive for a production patch in this pass. EPA slide 005
  `Content Placeholder 6` was rerun and remains 60,187 differing pixels; its
  diagnostic shows only 1,459 differing pixels inside partial-alpha soft-edge
  pixels, so no soft-edge production change was accepted. These rejected probes
  are failed implementation attempts only; the affected fixtures remain
  supported-scope implementation blockers, not completion shortcuts or
  Unsupported records.
- [x] Recheck text-box shape-autofit blockers without broad autofit shortcuts.
  Evidence: WHO slide 008 `TextBox 4` is a `p:sp` with
  `p:cNvSpPr txBox="1"` and `a:bodyPr/a:spAutoFit`. The source/schema anchors
  are `dml-main.xsd:805 CT_NonVisualDrawingShapeProps/@txBox`,
  `dml-main.xsd:2617 CT_TextShapeAutofit`, and
  `dml-main.xsd:2625 CT_TextBodyProperties`. Focused reruns show the object
  still fails at 26,639 pixels; the profile records source geometry
  `y=111..183`, current text target `y=111..187`, measured text 730x69, and a
  best simple text-mask shift that only lowers the fixture to 24,737 pixels.
  WHO slide 005 `TextBox 11` has the same `spAutoFit` family signal and still
  fails at 22,020 pixels; its text-like residual is only 313 pixels, so a
  generic text shift is not an acceptable production fix. No `spAutoFit`
  suppression, height threshold, or vertical-shift change was accepted.
- [x] Implement feasible M12 alphaOutset static effect support instead of
  leaving it Unsupported.
  Evidence: `CT_AlphaOutsetEffect` (`dml-main.xsd:1255`) is an implementable
  static DrawingML effect with `rad`; the renderer now parses it from
  `effectLst` and flattened `effectDag`, lowers it through shape/picture effect
  primitives, and renders source-backed alpha-mask expansion for supported
  static shape and picture objects. Validation:
  `go test ./internal/render -run 'TestM10.*AlphaOutset|TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes|TestM10PictureBackendPaintsAlphaOutsetEffect' -count=1 -v`,
  `go test ./internal/render -run TestM10 -count=1`, and
  `go test ./internal/render -count=1` all passed on 2026-06-02.
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed and
  moved `CT_AlphaOutsetEffect` to Partial/common-partial, reducing Unsupported
  rows from 395 to 394.
- [x] Implement feasible M12 relative-offset static effect support instead of
  leaving it Unsupported.
  Evidence: `CT_RelativeOffsetEffect` (`dml-main.xsd:1371`) defines static
  DrawingML `relOff` source semantics through `tx`/`ty` percentages. The
  renderer now parses `relOff` from `effectLst` and flattened `effectDag`,
  lowers it through shape/picture effect primitives and theme effect refs, and
  renders source-backed object-layer translation for supported static shape and
  picture objects. Validation:
  `go test ./internal/render -run 'TestM10.*RelativeOffset|TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes' -count=1 -v`,
  `go test ./internal/render -run TestM10 -count=1`, and
  `go test ./internal/render -count=1` all passed on 2026-06-02.
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed and
  moved `CT_RelativeOffsetEffect` to Partial/common-partial, reducing
  Unsupported rows from 394 to 393.
- [x] Implement feasible M12 transform-effect translation support instead of
  leaving it Unsupported.
  Evidence: `CT_TransformEffect` (`dml-main.xsd:1382`) defines static
  DrawingML `xfrm` effect attributes `sx`, `sy`, `kx`, `ky`, `tx`, and `ty`;
  `EG_Effect/xfrm` is anchored at `dml-main.xsd:1646`. The renderer now parses
  `xfrm` from `effectLst` and flattened `effectDag`, lowers `tx`/`ty`
  coordinate translation through shape/picture effect primitives and theme
  effect refs, and renders source-backed object-layer translation for
  supported static shape and picture objects. Non-default scale/skew
  attributes remain explicitly reported as partial. Validation:
  `go test ./internal/render -run 'TestM10.*Transform|TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes' -count=1 -v`
  passed on 2026-06-02.
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed and
  moved `CT_TransformEffect` to Partial/hard-rendering, reducing Unsupported
  rows from 393 to 392.
- [x] Implement feasible M12 HSL and tint blip-effect rendering instead of
  leaving them as reported-only visible image effects.
  Evidence: `CT_HSLEffect` (`dml-main.xsd:1292`) defines static DrawingML
  `hsl` blip-effect attributes `hue`, `sat`, and `lum`, and `CT_Blip`
  includes `hsl` at `dml-main.xsd:1493`. `CT_TintEffect`
  (`dml-main.xsd:1378`) defines static DrawingML `tint` blip/effect
  attributes `hue` and `amt`; ECMA-376 section 20.1.8.60 describes this as
  shifting effect color values toward or away from the target hue by the
  specified amount. The renderer now parses `hsl` and `tint` from `a:blip`,
  lowers the source attributes through picture primitives, and applies both
  effects in source-image space before picture sampling. Validation:
  `go test ./internal/render -run 'TestM07.*Blip.*(HSL|Tint)|TestM07ParsesBlipFillModeLinkAndEffects|TestM07PictureSourceAppliesBlipTintEffect' -count=1 -v`
  passed on 2026-06-02.
- [x] Implement feasible M12 simple blip blur rendering instead of leaving it
  as a reported-only visible image effect.
  Evidence: `CT_BlurEffect` (`dml-main.xsd:1264`) defines `rad` and optional
  `grow`, and `CT_Blip` includes `blur` at `dml-main.xsd:1487`. The renderer
  now parses `a:blip/a:blur`, lowers source-blur metadata through picture
  primitives, samples the simple picture into its target, and applies the
  existing source-backed Gaussian blur using the slide-scaled `rad` value.
  Combined blip blur with higher-order object effects remains an explicit
  partial diagnostic. Validation:
  `go test ./internal/render -run 'TestM07.*Blip.*(Blur|HSL|Tint)|TestM07ParsesBlipFillModeLinkAndEffects|TestM07PictureBackendAppliesBlipBlurEffect' -count=1 -v`
  passed on 2026-06-02.
- [x] Implement feasible M12 blip fillOverlay rendering instead of leaving it
  as a reported-only visible image effect.
  Evidence: `dml-main.xsd:1606 CT_FillOverlayEffect` defines a required
  `EG_FillProperties` fill and required `ST_BlendMode` blend value, and
  `CT_Blip` includes `fillOverlay` at `dml-main.xsd:1491`. The renderer now
  parses `a:blip/a:fillOverlay`, lowers source-fill-overlay metadata through
  picture primitives, and applies the existing supported blend modes to the
  source image before sampling. Validation:
  `go test ./internal/render -run 'TestM07.*Blip.*(Blur|HSL|Tint|FillOverlay)|TestM07ParsesBlipFillModeLinkAndEffects|TestM07PictureSourceAppliesBlipFillOverlayEffect|TestM07PictureBackendAppliesBlipBlurEffect' -count=1 -v`
  passed on 2026-06-02.
- [x] Implement feasible M12 scalar-container blip alphaMod rendering instead
  of leaving it as a reported-only visible image effect.
  Evidence: `dml-main.xsd:1660 CT_AlphaModulateEffect` requires a `cont`
  child, and `CT_Blip` includes `alphaMod` at `dml-main.xsd:1482`. The
  renderer now parses `a:blip/a:alphaMod/a:cont` when the container collapses
  to supported scalar `alphaModFix` children, lowers alpha-modulation metadata
  through picture primitives, and applies it to source image alpha before
  sampling. Missing or non-scalar containers remain explicit partial
  diagnostics. Validation:
  `go test ./internal/render -run 'TestM07.*Blip.*Alpha|TestM07ParsesBlipFillModeLinkAndEffects|TestM07ReportsUnsupportedBlipAlphaModContainer|TestM07PictureSourceAppliesBlipAlphaColorAndLuminanceEffects|TestM07PictureSourceAppliesBlipAlphaModulateEffect' -count=1 -v`
  passed on 2026-06-02.
- [x] Move feasible shape/effect-style DrawingML 3-D scene metadata out of the
  Unsupported bucket and into explicit Partial reporting.
  Evidence: `CT_Point3D` (`dml-main.xsd:633`), `CT_Vector3D`
  (`dml-main.xsd:638`), `CT_SphereCoords` (`dml-main.xsd:643`),
  `ST_PresetCameraType` (`dml-main.xsd:1033`), `ST_FOVAngle`
  (`dml-main.xsd:1099`), `CT_Camera` (`dml-main.xsd:1105`),
  `CT_LightRig` (`dml-main.xsd:1156`), `CT_Scene3D`
  (`dml-main.xsd:1163`), `CT_Backdrop` (`dml-main.xsd:1171`),
  `CT_Bevel` (`dml-main.xsd:1195`), `ST_PresetMaterialType`
  (`dml-main.xsd:1200`), `CT_Shape3D` (`dml-main.xsd:1219`),
  `CT_FlatText` (`dml-main.xsd:1233`), and `EG_Text3D`
  (`dml-main.xsd:1236`) describe source-authored static 3-D scene, camera,
  light-rig, bevel, shape-depth, and text-depth metadata. The renderer now
  parses and reports local `a:scene3d`, theme effect-style `a:scene3d`,
  non-zero `a:sp3d@z`, schema-default bevel dimensions, and text-body
  `scene3d`/`sp3d`/`flatTx` metadata while keeping actual 3-D surface rendering
  as an explicit Partial gap. Validation:
  `go test ./internal/render -run 'TestRenderShapeReportsUnsupportedVisibleShape3DProperties|TestCollectSlideElements.*Shape3D|TestCollectSlideElementsParsesScene3DWithoutShape3D|TestParseStylePropertiesAppliesThemeShape3DEffectReference|TestParseBodyPropertiesReadsText3DMetadata|TestRenderShapeReportsSpecificUnsupportedTextLayoutFeatures' -count=1 -v`
  and `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-02.
- [x] Move statically implementable chart/chartDrawing declarations out of the
  Unsupported bucket and into hard-rendering Partial work.
  Evidence: `dml-chart.xsd` and `dml-chartDrawing.xsd` describe statically
  renderable chart graphics and user-shape drawing parts, so M12 now treats
  them as preserved Partial implementation gaps rather than source-proven
  impossible content. Chart graphic frames still preserve the chart
  relationship/part and emit a deterministic render skip item, but the code is
  now `render_partial_object` instead of `render_unsupported_object`.
  Validation:
  `go test ./internal/render -run 'TestM11CollectSlideElementsClassifiesChartGraphicFrame|TestM11RenderGraphicFrameReportsChartPayload' -count=1 -v`
  and `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-02. The chart-only queue totals from that pass were
  superseded by the SmartArt/diagram and lockedCanvas reclassification below.
- [x] Move statically implementable SmartArt/diagram and lockedCanvas
  declarations out of the Unsupported bucket and into hard-rendering Partial
  work.
  Evidence: `dml-diagram.xsd` describes static SmartArt data, layout,
  constraints, style, color, and related drawing semantics. The renderer
  already lowers related diagram drawing fallbacks into static shape/text
  primitives when available and reports unavailable fallbacks or unsupported
  diagram subcontent through `render_partial_object`, so incomplete SmartArt
  layout is implementation work rather than source-proven impossible content.
  `dml-lockedCanvas.xsd` and `dml-main.xsd` GVML declarations describe static
  DrawingML grouping content, so M12 now lowers
  `graphicData/lc:lockedCanvas` children through the existing GVML group parser
  for supported shapes and standalone `txSp` text shapes.
  Validation:
  `go test ./internal/render -run 'TestM12.*LockedCanvas|TestMicroFixtureCoverageQueueSummaryReadsGeneratedMetadata|TestRenderGraphicFramePaintsSupportedDiagramDrawing|TestRenderGraphicFrameUsesPackageThemeForDiagramDrawing|TestDiagramDrawingElementsResolveSlideThemeColorMapAndFonts|TestDiagramDrawingElementsResolveSlideThemeFillAndEffectStyles|TestRenderGraphicFrameReportsUnsupportedDiagramContent' -count=1 -v`
  and `python3 tools/generate_ooxml_drawingml_audit.py --print-summary`
  passed on 2026-06-02; queue totals are `common-partial=389`,
  `hard-rendering=458`, `unsupported-preserve=16`, and `out-of-scope=128`.
  `go test ./...`
  passed, and
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-gvml.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed; top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, and `Google Shape;179;p9` 127,167.
- [x] Implement common DrawingML auto-number marker formats.
  Evidence: `dml-main.xsd:2666` declares `ST_TextAutonumberScheme`, and
  `dml-main.xsd:2747` declares `CT_TextAutonumberBullet` with `type` and
  `startAt`. WHO HIV slide 006 `Rectangle 6` uses `arabicPeriod`, including
  `startAt="2"`, and WHO HIV slide 007 `Rectangle 7` uses `alphaLcPeriod`.
  The renderer now formats common alpha, arabic, and Roman marker variants
  directly from the authored scheme instead of collapsing most variants to
  `n.`. Locale-specific Thai/Hindi/East Asian/circled numbering and picture
  bullets remain Partial implementation work, not Unsupported.
  Validation:
  `go test ./internal/render -run 'TestTextParagraphsFromNodeNumbersAutoBullets|TestTextParagraphsFromNodeInheritsStyledAutoNumberBullets|TestAutoNumberBulletFormatsCommonDrawingMLSchemes' -count=1 -v`
  passed, and
  `go test ./internal/render -run 'TestM08|TestTextParagraphsFromNode.*Auto|TestAutoNumberBullet' -count=1`
  passed. Targeted object checks still fail as supported-scope visual parity:
  WHO slide 006 `Rectangle 6` is 192,327 differing pixels and WHO slide 007
  `Rectangle 7` is 56,812 differing pixels. `go test ./...` passed.
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed
  with unchanged queue totals: `core-static=16`, `common-partial=389`,
  `hard-rendering=458`, `unsupported-preserve=16`, and `out-of-scope=128`.
  The clean micro-fixture suite passed only in expected-failure accounting mode
  with `/tmp/puppt-clean-suite-autonum.json`: 59 total, 0 passed, 59 failed;
  top failures remain `Table 3` 284,470, `Picture 2` 154,741, `TextBox 7`
  130,250, and `Google Shape;179;p9` 127,167. The exact Apple Notes gate still
  fails with 61/61 differing slides, 9,305,437 total differing pixels, and no
  unsupported rendering gaps.
- [x] Preserve paragraph direction and line-break flags in text primitives.
  Evidence: `dml-main.xsd:2994` declares `CT_TextParagraphProperties`, and
  `dml-main.xsd:3013-3017` declare `rtl`, `eaLnBrk`, `fontAlgn`,
  `latinLnBrk`, and `hangingPunct`. WHO HIV slide 003 `TextBox 7` and WHO HIV
  slide 012 `Table 3` both author `rtl="0"`, `eaLnBrk="1"`,
  `latinLnBrk="0"`, and `hangingPunct="1"`. The renderer now preserves these
  paragraph flags through parsing, list-style inheritance, render primitives,
  and object-debug `text_paragraph_properties`; authored `rtl="1"` is reported
  as left-to-right fallback until bidi paragraph layout is implemented.
  Validation:
  `go test ./internal/render -run 'TestTextParagraphsFromNodeCapturesParagraphLineBreakFlags|TestTextParagraphsFromNodeInheritsParagraphLineBreakFlags|TestM08TextLayoutReportsAuthoredRTLParagraphFallback|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle' -count=1 -v`
  passed, and
  `go test ./internal/render -run 'TestM08|TestTextParagraphsFromNodeCapturesParagraphLineBreakFlags|TestTextParagraphsFromNodeInheritsParagraphLineBreakFlags|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects' -count=1`
  passed. Targeted object checks still fail as supported-scope visual parity:
  WHO slide 003 `TextBox 7` remains 130,250 differing pixels and WHO slide 012
  `Table 3` remains 284,470 differing pixels. In both current-object records,
  `resolved_style.text_paragraph_properties` contains `rtl=false`,
  `eaLnBrk=true`, `latinLnBrk=false`, and `hangingPunct=true`, and
  `unsupported` is null because these fixtures do not author `rtl="1"`.
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed
  with unchanged queue totals: `core-static=16`, `common-partial=389`,
  `hard-rendering=458`, `unsupported-preserve=16`, and `out-of-scope=128`.
- [x] Tighten generated coverage status rules so Unsupported requires
  source-proven static-renderer impossibility.
  Evidence: `tools/generate_ooxml_drawingml_audit.py` now defines Unsupported
  as source-proven static-renderer impossibility requiring detection,
  preservation where possible, and renderer/JSON reporting. Missing
  implementation, high pixel diff, local fixture failure, or difficult static
  rendering remains Partial or hard-rendering work. The regenerated
  `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` and
  `docs/renderer-coverage-summary.json` keep only 16 Unsupported rows:
  OLE payload runtime, active control execution, and time-based audio/video
  playback declarations. Validation:
  `python3 tools/generate_ooxml_drawingml_audit.py --print-summary` passed
  with `core-static=16`, `common-partial=389`, `hard-rendering=458`,
  `unsupported-preserve=16`, and `out-of-scope=128`;
  the Unsupported-row audit printed 16 OLE/control/media impossibility rows;
  the stale wording scan for obsolete Unsupported wording and typo variants
  found no hits in the M12-facing docs/generator; and
  `git diff --check` passed for the touched files.
- [x] Preserve picture outlines after source-image blur effects.
  Evidence: DrawingML `a:blip/a:blur` is a blip/source-image effect, so the
  authored picture outline remains a shape outline and must not be blurred with
  the source image. The picture backend now removes the line from the temporary
  source-blur layer and paints it afterward, including rotated pictures by
  rotating a local outline layer with the picture transform. Validation:
  `go test ./internal/render -run 'TestM07PictureBackend(AppliesBlipBlurEffect|KeepsRotatedOutlineOutsideBlipBlur)' -count=1`
  passed, `go test ./internal/render -run TestM07 -count=1` passed, and
  `go test ./internal/render -count=1` passed.
- [x] Apply table-style text defaults into cell text layout.
  Evidence: `dml-main.xsd:2471` declares `CT_TableStyleTextStyle` with
  `EG_ThemeableFontStyles`, `EG_ColorChoice`, and `b`/`i` attributes. WHO HIV
  slide 012 `Table 3` uses `ppt/tableStyles.xml` conditional `tcTxStyle`
  entries with `fontRef`, text color, and bold header regions. The renderer
  already parsed these values; M12 now copies table-style text color, bold,
  italic, and font-family defaults into the copied table-cell paragraphs/runs
  consumed by text layout instead of leaving font/italic only on the wrapper
  element. Validation:
  `go test ./internal/render -run 'TestTableTextParagraphsWithItalicCopiesParagraphs|TestTableTextParagraphsWithFontFamilySuppliesParagraphDefault|TestTableCellTextElementAppliesStyleTextDefaultsToSegments|TestTableTextParagraphsWithBoldCopiesParagraphs|TestTableTextParagraphsWithColorOverridesParagraphDefaultsButPreservesRuns|TestParseTableStylesReadsDirectTableTextFontAndItalic' -count=1`
  passed. The targeted `Table 3` fixture still fails at 284,470 differing
  pixels, so this is accepted source-semantics coverage but not an M12 gate
  closure.
- [x] Include authored blank table-cell paragraphs in row text minimums.
  Evidence: source tables can carry empty `a:p` paragraphs with
  `endParaRPr@sz` metrics inside `a:tc/a:txBody`; these paragraphs are blank
  line boxes for layout and should size zero-height or reflowed table rows
  even though they do not draw visible glyphs. `tableTextMinimumRowHeights`
  now measures cells with authored paragraph/run metrics instead of skipping
  every cell whose aggregate trimmed text is empty, while arbitrary empty cells
  still remain inert. Validation:
  `go test ./internal/render -run 'TestTableTextMinimumRowHeightsMeasuresAuthoredBlankParagraphs|TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth|TestTableTextMinimumRowHeightsDistributesRowSpanText|TestTableRowOffsetsWithTextMinimums' -count=1`
  passed, `go test ./internal/render -count=1` passed, and the targeted WHO
  slide 012 `Table 3` fixture still failed at 284,470 differing pixels. This
  is source-semantics coverage, not an M12 gate closure.
- [ ] Run the real-world Apple Notes gate and confirm all 61 slides pass.
  Evidence:
  `PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1`
  failed after the `CT_TextParagraphProperties` paragraph-flag preservation
  update:
  61/61 slides differ, total differing pixels 9,305,437, worst slide
  `EPA-generate-2021-presentation.pptx` slide 001 has 307,925 differing
  pixels, and top unsupported rendering gaps are none.
- [x] Run the object fixture suite for all previously tracked clean failures.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-cstate.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed only in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. The same result was reconfirmed on 2026-06-02 after the
  `CT_TextParagraphProperties` paragraph-flag preservation update with output
  written to `/tmp/puppt-clean-suite-paragraph-flags.json`. Top failures remain
  `Table 3`
  284,470, `Picture 2` 154,741, `TextBox 7` 130,250, and
  `Google Shape;179;p9` 127,167.
  The same expected-failure accounting result was reconfirmed after the
  source-blur outline update with output written to
  `/tmp/puppt-clean-suite-source-blur-outline.json`.
  The result was reconfirmed again after the table-style text-default
  propagation update with output written to
  `/tmp/puppt-clean-suite-table-text-defaults.json`: 59 total, 0 passed,
  59 failed; top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.
  The result was reconfirmed again after authored blank table-cell paragraph
  line boxes were included in row text minimums with output written to
  `/tmp/puppt-clean-suite-blank-table-paragraphs.json`: 59 total, 0 passed,
  59 failed; top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.
  The result was reconfirmed again after inherited-size baseline run
  measurement/drawing with output written to
  `/tmp/puppt-clean-suite-baseline-fallback.json`: 59 total, 0 passed,
  59 failed; top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.
- [x] Grep production Go code for office/browser/SaaS/image-conversion renderer
  dependency paths.
  Evidence:
  `go test ./internal/render -run TestRendererImplementationHasNoTargetDeckHardcodesOrExternalRendererCalls -count=1 -v`
  passed, and `go list -deps ./cmd/puppt` had no forbidden renderer dependency
  hits.
- [x] Run `puppt render ... --json` stability checks.
  Evidence:
  `go test ./internal/cli -run 'TestRenderJSON|TestRenderJSONHonorsDPIFlag' -count=1 -v`
  passed; direct `go run ./cmd/puppt render ... --json` checks returned the
  expected stable `puppt.v1` render envelope.
- [x] Confirm `docs/RENDERER_EXPERIMENT_LOG.md` is updated.
  Evidence: updated with the M12 final audit evidence, custom geometry
  fractional fill evidence, and blocker decision.
- [x] Summarize changed files, verification results, residual risks, and next
  maintenance checkpoint.
  Evidence: M12 changed this checklist, `docs/RENDERER_EXPERIMENT_LOG.md`,
  `docs/SUPPORT_MATRIX.md`, and `docs/RENDERING.md`. Residual risk is exact
  visual parity and object-fixture conformance, not hidden unsupported payload
  handling. Next checkpoint is fixture-family reduction of the 59 tracked clean
  failures, followed by another exact real-world gate run.
- [ ] Only after all above evidence is present, mark
  `docs/RENDERER_COMPLETION_GOAL.md` complete.
  Evidence: not complete; the real-world and clean-fixture gates are still
  failing.

### 5.16 Non-Visual Text Box Metadata

- [x] Read authoritative OOXML for WHO HIV slide 003 `TextBox 7`.
  Evidence: inspected the object fixture for
  `testdata/realworld-ppts/WHO-HIV-testing-algorithms-toolkit.pptx`, slide 3,
  `<p:sp>` cNvPr id `8`, name `TextBox 7`. The source contains
  `p:cNvSpPr txBox="1"`, which maps to
  `dml-main.xsd:800 CT_NonVisualDrawingShapeProps@txBox`.
- [x] Preserve `txBox` through semantic parsing, render primitive lowering,
  and object-debug summaries.
  Evidence: `slideElement.IsTextBox`, `renderTextPrimitive.IsTextBox`, and the
  object-debug `resolved_style.text_box` field now carry the authored
  text-box flag.
- [x] Run focused parser, primitive, and object-debug tests.
  Evidence:
  `go test ./internal/render -run 'TestCollectSlideElementsParsesNonVisualTextBoxFlag|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestAnchorCenteredTextBoundsCentersNarrowTextBox' -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the WHO slide 003 `TextBox 7` object fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-txbox PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 130,250 differing pixels. The current object summary now
  records `"text_box": true`; the remaining residual is supported-scope text
  layout/metrics work, not an Unsupported record.

### 5.17 Non-Visual Lock Metadata

- [x] Read authoritative OOXML for current failing picture and table objects.
  Evidence: WHO HIV slide 009 `Picture 2` authors
  `p:cNvPicPr/a:picLocks noChangeAspect="1"` and WHO HIV slide 012
  `Table 3` authors `p:cNvGraphicFramePr/a:graphicFrameLocks noGrp="1"`.
  The schema anchors are `dml-main.xsd:727 AG_Locking`,
  `dml-main.xsd:752 CT_PictureLocking`, and
  `dml-main.xsd:828 CT_NonVisualGraphicFrameProperties`.
- [x] Preserve enabled non-visual lock flags through semantic parsing, render
  primitive lowering, and object-debug summaries.
  Evidence: `slideElement.NonVisualLocks` now captures enabled local lock
  attributes as deterministic strings, and picture, shape, connector,
  graphic-frame, and group primitives plus `resolved_style.non_visual_locks`
  carry the values.
- [x] Run focused parser, primitive, and object-debug tests.
  Evidence:
  `go test ./internal/render -run 'TestCollectSlideElementsParsesNonVisualTextBoxFlag|TestCollectSlideElementsParsesPictureLockFlags|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors' -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the targeted source fixtures.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-picture2-locks PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 154,741 differing pixels while the current object summary
  records `["picLocks.noChangeAspect"]`.
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-locks PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 284,470 differing pixels while the current object summary
  records `["graphicFrameLocks.noGrp"]`. These residuals remain
  supported-scope rendering work, not Unsupported records.
- [x] Re-run the clean object fixture suite in expected-failure accounting
  mode.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-locks.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. Top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.
*** End of File

### 5.18 cNvPr Descriptive Metadata

- [x] Read authoritative OOXML for current EPA picture fixtures with cNvPr
  metadata.
  Evidence: EPA Generate slide 007 `Picture 2` authors
  `adec:decorative val="1"` inside `p:cNvPr/a:extLst`, and EPA Generate
  slide 003 `Picture 25` authors
  `descr="Diagram&#xA;&#xA;Description automatically generated"`. Both objects
  are current clean-suite failures and map to
  `dml-main.xsd:788 CT_NonVisualDrawingProps`.
- [x] Preserve cNvPr description/title and boolean metadata through semantic
  parsing, render primitive provenance, and object-debug summaries.
  Evidence: `slideElement.Description`, `slideElement.Title`, and
  `slideElement.NonVisualProperties` now carry `descr`, `title`,
  `hidden=true/false`, and `decorative=true/false`; `PaintedObject`,
  `renderPrimitiveProvenance`, and `resolved_style` expose the values.
- [x] Run focused parser, object-debug, and primitive tests.
  Evidence:
  `go test ./internal/render -run 'TestCollectSlideElementsParsesPictureLockFlags|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderObjectDebugRecordsArtifactsAndIsolationModes|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors' -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the targeted source fixtures.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-cnvpr-decorative PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-007/micro-fixtures/0008-3-Picture-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 95,960 differing pixels while the current object summary
  records `["decorative=true"]`.
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-cnvpr-description PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-generate-2021-presentation/slide-003/micro-fixtures/0004-26-Picture-25/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 65,347 differing pixels while the fixture object record and
  current object summary record the authored description. These residuals
  remain supported-scope rendering work, not Unsupported records.
- [x] Re-run the clean object fixture suite in expected-failure accounting
  mode.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-cnvpr.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. Top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.

### 5.19 cNvPr Creation ID Metadata

- [x] Read authoritative OOXML for current WHO fixtures with cNvPr creation
  metadata.
  Evidence: WHO HIV slide 012 `Table 3` authors
  `a16:creationId id="{D32AD674-1F5F-084E-9B33-D94CDE5FD8BD}"` under
  `p:cNvPr/a:extLst`; WHO HIV slide 009 `Picture 2` authors
  `{5801FB1F-5610-3B4D-AF42-4F8881B7C4B0}`; WHO HIV slide 003 `TextBox 7`
  authors `{93C8E66B-89E0-1A42-98AC-3BF114210851}`. These objects are current
  clean-suite failures and map to `dml-main.xsd:788
  CT_NonVisualDrawingProps`.
- [x] Preserve cNvPr creation IDs through semantic parsing, render primitive
  provenance, and object-debug summaries.
  Evidence: `slideElement.CreationID` now carries descendant
  `creationId@id`; `PaintedObject.CNvPrCreationID`,
  `renderPrimitiveProvenance.CreationID`, and `resolved_style.creation_id`
  expose the value.
- [x] Run focused parser, object-debug, and primitive tests.
  Evidence:
  `go test ./internal/render -run 'TestCollectSlideElementsParsesPictureLockFlags|TestRenderObjectDebugRecordsArtifactsAndIsolationModes|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields|TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects|TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors' -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the targeted source fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-cnvpr-creationid-table3 PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 284,470 differing pixels while the fixture object record
  preserves `cnv_pr_creation_id="{D32AD674-1F5F-084E-9B33-D94CDE5FD8BD}"`
  and the current object summary records the same value in
  `resolved_style.creation_id`. This residual remains supported-scope
  rendering work, not an Unsupported record.
- [x] Re-run the clean object fixture suite in expected-failure accounting
  mode.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-creationid.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. Top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.

### 5.20 Table Row And Column ID Metadata

- [x] Read authoritative OOXML for the current top table fixture.
  Evidence: WHO HIV slide 012 `Table 3` authors six
  `a16:colId@val` values under `a:tblGrid/a:gridCol/a:extLst` and thirteen
  `a16:rowId@val` values under `a:tr/a:extLst`. The object maps to
  `dml-main.xsd:2381 CT_TableGrid`, `dml-main.xsd:2386 CT_TableCell`,
  `dml-main.xsd:2398 CT_TableRow`, and `dml-main.xsd:2423 CT_Table`.
- [x] Preserve table row and column IDs through semantic parsing, table
  primitive lowering, and object-debug summaries.
  Evidence: `tableModel.ColumnIDs` now carries grid-column extension IDs,
  `tableRow.ID` carries row extension IDs, `renderTablePrimitive.ColumnIDs`
  and copied rows preserve the values, and `resolved_style.table_column_ids`
  plus `resolved_style.table_row_ids` expose them in object-debug records.
- [x] Run focused parser, scene, and object-debug tests.
  Evidence:
  `go test ./internal/render -run 'TestParseGraphicFrameReadsTableGrid|TestRenderSceneFromElementsLowersAllPrimitiveFamilies|TestObjectStyleSummaryIncludesResolvedParagraphTextStyle' -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the targeted source fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table-rowcol-id-table3 PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 284,470 differing pixels while both the fixture object and
  current object summary preserve all six source column IDs and all thirteen
  source row IDs. This residual remains supported-scope table rendering work,
  not an Unsupported record.
- [x] Re-run the clean object fixture suite in expected-failure accounting
  mode.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-rowcol-ids.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. Top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.

### 5.21 Text Object Font Family Summaries

- [x] Read authoritative OOXML for the current top text fixture.
  Evidence: WHO HIV slide 003 `TextBox 7` authors Arial typefaces in
  `a:pPr/a:buFont` and in text run `a:rPr/a:latin` / `a:rPr/a:cs`, while the
  previous object-debug summary reported the inherited shape fallback font as
  `Calibri`. The object maps to `dml-main.xsd:2873
  CT_TextCharacterProperties`, `dml-main.xsd:2814 CT_TextFont`, and
  `dml-main.xsd:2994 CT_TextParagraphProperties`.
- [x] Preserve authored text font-family evidence in object-debug summaries.
  Evidence: `objectStyleSummary` now derives `font_family` from authored
  paragraph/run text families before shape fallback font context and exposes a
  deterministic `font_families` list.
- [x] Run focused object-debug coverage.
  Evidence:
  `go test ./internal/render -run TestObjectStyleSummaryIncludesResolvedParagraphTextStyle -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the targeted source fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-textbox7-font-summary PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-003/micro-fixtures/shape-0005-8-TextBox-7/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 130,250 differing visible pixels while the current object
  summary now records `font_family="Arial"` and
  `font_families=["Arial","Calibri"]`. This residual remains supported-scope
  text layout/rendering work, not an Unsupported record.
- [x] Re-run the clean object fixture suite in expected-failure accounting
  mode.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-text-font-summary.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. Top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.

### 5.22 Table Style ID And Flag Summaries

- [x] Read authoritative OOXML for the current top table fixture.
  Evidence: WHO HIV slide 012 `Table 3` authors
  `a:tblPr firstRow="1" bandRow="1"` and
  `a:tableStyleId>{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}</a:tableStyleId`.
  The object maps to `dml-main.xsd:2405 CT_TableProperties` and
  `dml-main.xsd:2423 CT_Table`.
- [x] Preserve authored table style identity and table flags in object-debug
  summaries.
  Evidence: table parsing and primitive lowering already carried the style ID
  and flags; M12 now exposes them as additive
  `resolved_style.table_style_id` and `resolved_style.table_properties`
  object-debug fields for fixture triage and JSON evidence.
- [x] Run focused object-debug coverage.
  Evidence:
  `go test ./internal/render -run TestObjectStyleSummaryIncludesResolvedParagraphTextStyle -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the targeted source fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-table3-style-summary PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-012/micro-fixtures/table-0002-2-Table-3/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 284,470 differing pixels while the current object summary
  records `table_style_id="{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}"` and
  `table_properties=["firstRow=true","bandRow=true"]`. This residual remains
  supported-scope table rendering work, not an Unsupported record.
- [x] Re-run the clean object fixture suite in expected-failure accounting
  mode.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-table-style-summary.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. Top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.

### 5.23 Picture Source Media Summaries

- [x] Read authoritative OOXML for the current top picture fixture.
  Evidence: WHO HIV slide 009 `Picture 2` authors `a:blip r:embed="rId4"`
  under `p:pic/p:blipFill`, uses `a:stretch/a:fillRect`, has no authored
  crop, mask, or effect wrapper, and points at a PNG source image with decoded
  dimensions 2830x820. The object maps to `pml.xsd:1245 CT_Picture`,
  `dml-main.xsd:1477 CT_Blip`, and
  `dml-main.xsd:1502 CT_BlipFillProperties`.
- [x] Preserve resolved picture source media in object-debug summaries.
  Evidence: after `pictureSourceImage` resolves and decodes the image,
  object-debug records now expose the resolved media part, content type, and
  intrinsic decoded size in `resolved_style.image`.
- [x] Run focused picture/object-debug coverage.
  Evidence:
  `go test ./internal/render -run 'TestObjectStyleSummaryIncludesImageAndTableProperties|TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields' -count=1 -v`
  passed on 2026-06-02.
- [ ] Close the targeted source fixture.
  Evidence:
  `PUPPT_MICRO_FIXTURE_DEBUG_DIR=/tmp/puppt-picture2-source-media PUPPT_MICRO_FIXTURE_MANIFEST=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-009/micro-fixtures/0003-3-Picture-2/manifest.json go test ./internal/render -run TestMicroFixtureManifestComparison -count=1 -v`
  still failed with 154,741 differing pixels while the current object summary
  records `image="embed=rId4 part=ppt/media/object.png type=image/png size=2830x820"`,
  `image_effects=["fillMode=stretch"]`, and `image_unsupported=null`. This
  residual remains supported-scope picture sampling/color work, not an
  Unsupported record.
- [x] Re-run the clean object fixture suite in expected-failure accounting
  mode.
  Evidence:
  `PUPPT_MICRO_FIXTURE_ROOT=testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01 PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES=1 PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT=/tmp/puppt-clean-suite-picture-source-media.json go test ./internal/render -run TestMicroFixtureCleanFailureSuite -count=1 -v`
  passed in expected-failure accounting mode with 59 total, 0 passed, and
  59 failed. Top failures remain `Table 3` 284,470, `Picture 2` 154,741,
  `TextBox 7` 130,250, `Google Shape;179;p9` 127,167, and EPA `Picture 2`
  95,960.
