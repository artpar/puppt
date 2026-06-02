# M06 Geometry, Stroke, And Connectors

## Objective

Implement a production vector backend for DrawingML path geometry, fills,
strokes, joins, caps, markers, and connectors from source semantics.

## Inputs

- `dml-main.xsd` geometry and line declarations
- `pml.xsd` shape and connector declarations
- current shape/connector renderer
- object fixtures for `Rectangle 5`, `Rectangle 3`, connector controls, and custom paths

## In Scope

- Preset shape geometry used by common decks.
- Custom path commands: move, line, cubic, close, and then arcs/multiple paths.
- Stroke width, cap, dash, join, compound line, pen alignment.
- Head/tail markers.
- Connector line geometry and connection-site semantics where source-backed.
- Shape masks used by pictures and shadows.

## Out Of Scope

- Text layout inside shapes.
- Picture sampling.
- Shadow blur/composite quality except geometry masks.

## Required Work

1. Select/finish the vector backend behind Puppt primitives.
2. Define path primitives independent of source XML.
3. Add synthetic geometry fixtures by schema row and preset shape.
4. Add stroke fixtures for width/cap/join/dash/compound/marker variants.
5. Add connector fixtures for straight, zero-width/height, routed if supported.
6. Prove current top shape fixtures pass or document source-backed residuals.

## Acceptance Criteria

- At least the common preset geometry subset moves from partial to supported.
- Custom paths either support each command or report the exact unsupported command.
- Stroke/marker unsupported cases are reported, not silently ignored.
- Same-family real-world shape fixtures do not regress.

## Verification

```text
go test ./internal/render -run 'TestRenderShape|Test.*Geometry|Test.*Connector|Test.*Line|Test.*Stroke|Test.*Marker' -count=1
PUPPT_MICRO_FIXTURE_MANIFEST=<focused-shape-manifest> go test ./internal/render -run TestMicroFixtureManifestComparison -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Update matrix rows for preset geometry, custom geometry, line properties,
connector objects, and shape properties.
