# M05 Theme, Color, Fill, And Style Resolution

## Objective

Resolve DrawingML theme/style/color/fill semantics from source XML into stable
paint primitives before geometry, text, image, and table backends consume them.

## Inputs

- `dml-main.xsd` color, fill, style matrix, line style, and font scheme declarations
- `pml.xsd` color map and background declarations
- current theme/color/fill code
- real-world fixtures with theme-derived fills and backgrounds

## In Scope

- Color schemes and color-map overrides.
- `phClr`, scheme colors, sRGB, scRGB, preset colors, system colors where defined.
- Color modifiers: alpha, luminance, shade, tint, offsets, and combinations.
- Fill choices: no fill, solid fill, gradient fill, blip fill, pattern fill reporting, group fill reporting.
- Shape style refs: fill, line, effect, font refs.
- Background `bgPr` and `bgRef`.

## Out Of Scope

- Path rasterization quality.
- Text shaping.
- Full effect rendering beyond resolving style references.

## Required Work

1. Define `PaintStyle` primitives for fill, stroke, effect style, and text color.
2. Add synthetic fixtures for every supported color model/modifier.
3. Add synthetic fixtures for direct fill versus style-derived fill precedence.
4. Add background fixtures for slide/layout/master and `bgRef`.
5. Add explicit unsupported reports for unimplemented fill/color modes.
6. Ensure all downstream renderers consume resolved paint primitives.

## Acceptance Criteria

- Supported color/fill rows have deterministic fixture proof.
- Unsupported fill/color clauses report partial/unsupported when visible.
- Current real-world color diagnostics improve or remain no-regression.
- No renderer backend re-parses theme XML directly.

## Verification

```text
go test ./internal/render -run 'Test.*Color|Test.*Theme|Test.*Fill|Test.*Background|Test.*Style' -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Update matrix rows for color choices, color transforms, fill properties,
backgrounds, and style matrix references.
