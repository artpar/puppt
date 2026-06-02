# M04 Coordinates, Transforms, And Clipping

## Objective

Make all later primitive rendering share one correct coordinate model:
EMU-to-pixel scaling, fractional bounds, transforms, group transforms, z-order,
clipping, and object masks.

## Inputs

- `dml-main.xsd` transform declarations
- `pml.xsd` group/shape tree declarations
- current geometry and object-debug code
- existing `Rectangle 5`, picture, and group fixtures

## In Scope

- EMU, fractional pixel, integer pixel, crop, and output bounds.
- Rotation and flips for shapes, pictures, connectors, and groups.
- Nested group transforms.
- Clip/mask boundaries for shapes, pictures, tables, and text.
- Object ownership masks for diagnostics.

## Out Of Scope

- Geometry path correctness beyond transform application.
- Text shaping and font metrics.
- Image sampling kernel selection.

## Required Work

1. Define one transform stack used by all primitive backends.
2. Add synthetic fixtures for:
   - offset/extent
   - fractional bounds
   - rotation
   - flipH/flipV
   - nested groups
   - clipping against object bounds
3. Replace ad hoc coordinate math where it conflicts with the shared stack.
4. Ensure object-debug bounds are derived from the same model.
5. Prove no zero-size/negative-size objects panic.

## Acceptance Criteria

- Transform fixtures pass exactly.
- Existing real-world object attribution still writes correct bounds.
- No broad visual regression in current corpus.
- Matrix rows for `CT_Transform2D`, `CT_GroupTransform2D`, shape tree, and group transforms are updated.

## Verification

```text
go test ./internal/render -run 'Test.*Transform|Test.*Bounds|Test.*Group|TestRenderObjectDebug' -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Document the coordinate model and every accepted renderer choice that the OOXML
schema leaves underspecified.
