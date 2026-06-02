# M03 Render Scene IR

## Objective

Finish the Puppt-owned render scene boundary. Every supported or partial
PresentationML/DrawingML object must lower from source XML into stable internal
render primitives before painting.

## Inputs

- `internal/render/render_scene.go`
- current `slideElement` model and paint paths
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/RENDERER_PRODUCTION_PATH.md`

## In Scope

- Scene model for slides, render parts, inherited objects, and z-order.
- Primitive structs for shape, path, connector, picture, text, table, effect,
  group, diagram, and unsupported records.
- Primitive provenance: part, XML path, cNvPr id/name, relationship ids, schema anchors.
- Backend interfaces that consume primitives, not raw `.pptx` XML.

## Out Of Scope

- Replacing every backend implementation.
- Pixel parity fixes unless needed to preserve existing behavior.
- Changing CLI/JSON output except additive debug fields.

## Required Work

1. Define `RenderScene` and primitive interfaces for every supported object family.
2. Lower shapes, connectors, pictures, graphic frames, tables, diagrams, groups, and unsupported objects.
3. Preserve current production pixels during migration.
4. Remove backend dependency on raw `slideElement` wherever a primitive exists.
5. Add tests proving field preservation for every primitive.
6. Add tests proving unresolved relationships become conversion/reporting errors, not panics.

## Acceptance Criteria

- Current picture primitive boundary remains zero-diff.
- Shape, connector, text, table, group, effect, and unsupported primitive lowering exists.
- Every primitive carries provenance and schema anchors.
- Backends can be swapped per primitive family.
- Existing renderer tests pass.

## Verification

```text
go test ./internal/render -run 'TestRenderScene|TestRender.*Primitive|TestRenderPicture|TestRenderShape|TestRenderGraphicFrame' -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Update the coverage matrix rows for object structure and primitive lowering.
Record any intentionally deferred primitive fields as partial, not supported.
