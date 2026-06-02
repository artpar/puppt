# M08 Text Shaping, Layout, And Fonts

## Objective

Implement DrawingML text from source semantics through shaping, metrics, line
layout, paragraph inheritance, bullets, autofit, and font fallback.

## Inputs

- `dml-main.xsd` text body, paragraph, run, list style, and body properties declarations
- `pml.xsd` master text styles and placeholders
- current text layout/font code
- `Rectangle 5`, `TextBox 7`, and neighboring text fixtures

## In Scope

- Font resolution and fallback reporting.
- OpenType shaping backend for supported scripts.
- Paragraph/run inheritance from local, layout, master, theme, and table styles.
- Body insets, wrapping, overflow, anchoring, line spacing, margins, hanging indents.
- Bullets, autonumbering, symbol fonts, underline, baseline, character spacing.
- Autofit: no, normal, and shape autofit.
- Explicit unsupported reports for vertical, columns, WordArt, bidi if not implemented.

## Out Of Scope

- Editing text semantics.
- Full Office font inventory.
- Animated text.

## Required Work

1. Define text primitives independent of shape/table source XML.
2. Install shaping/metrics backend behind a Puppt interface.
3. Add synthetic fixtures for each supported text body property.
4. Add paragraph/run inheritance fixtures.
5. Add object fixtures for centered text and TextBox residual families.
6. Ensure unsupported text modes are reported with precise schema anchors.

## Acceptance Criteria

- Text fixtures prove source semantics before visual acceptance.
- Focused text object fixtures pass or have source-backed accepted residuals.
- Font fallback/substitution is deterministic and reported.
- No broad y-shift/font tweak lands without source model proof.

## Verification

```text
go test ./internal/render -run 'Test.*Text|Test.*Font|Test.*Bullet|Test.*Autofit|Test.*Paragraph' -count=1
PUPPT_MICRO_FIXTURE_MANIFEST=<focused-text-manifest> go test ./internal/render -run TestMicroFixtureManifestComparison -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Update matrix rows for text body, list styles, paragraph/run properties,
placeholders, and font scheme behavior.
