# M10 Effects, Shadows, And Compositing

## Objective

Implement the supported effect/compositing subset from DrawingML source
semantics and report the rest precisely.

## Inputs

- `dml-main.xsd` effect list/container and shadow/effect declarations
- current shadow and soft-edge renderer
- object fixtures for shadow, soft edge, and effect residuals

## In Scope

- Alpha compositing model.
- Outer shadows: blur, distance, direction, alignment, color/alpha, transforms.
- Soft edges for shapes and pictures.
- Blur, glow, inner shadow, preset shadow, reflection, fill overlay if supported.
- Effect ordering and `effectDag` policy.
- 3-D detection and reporting.

## Out Of Scope

- Animated effects.
- GPU-specific or host-renderer-specific behavior without documented model.

## Required Work

1. Define effect primitives independent of shape/picture/table source XML.
2. Add synthetic fixtures for supported effect parameters.
3. Add a documented blur/composite model.
4. Add unsupported reports for unimplemented visible effects.
5. Prove effect changes on focused fixtures before corpus acceptance.

## Acceptance Criteria

- Supported effect rows have deterministic fixtures.
- Unsupported effects are not silently dropped.
- Shadow/effect object fixtures pass or record source-backed accepted residuals.
- Same-family shape/picture/table fixtures do not regress.

## Verification

```text
go test ./internal/render -run 'Test.*Shadow|Test.*Effect|Test.*SoftEdge|Test.*Composite|Test.*3D' -count=1
PUPPT_MICRO_FIXTURE_MANIFEST=<focused-effect-manifest> go test ./internal/render -run TestMicroFixtureManifestComparison -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Update matrix rows for effect lists, effect containers, shadows, soft edges, 3-D,
and compositing policy.
