# M09 Tables

## Objective

Implement DrawingML table rendering from source semantics: grid layout, cell
spans, merges, styles, text, fills, borders, effects, and unsupported reporting.

## Inputs

- `dml-main.xsd` table declarations
- `ppt/tableStyles.xml` parser and current table renderer
- real-world table object attribution fixtures

## In Scope

- Table grid, row/column extents, scaling, and frame fitting.
- Row/column spans, horizontal/vertical merges.
- Cell margins, text layout, anchors, and inherited text styles.
- Direct and table-style fills, borders, and text style regions.
- Border caps, joins, compound lines, and diagonal borders if implemented.
- Table background fills/effects.

## Out Of Scope

- Spreadsheet table semantics.
- Chart data tables unless chart rendering is in scope.
- Non-static table animation.

## Required Work

1. Lower tables into table primitives.
2. Add synthetic fixtures for every supported table schema subset.
3. Add table-style precedence fixtures.
4. Add unsupported reports for non-solid fills, effects, diagonal/compound borders, and cell 3-D until supported.
5. Generate clean real-world table fixtures and ownership summaries.

## Acceptance Criteria

- Common table rows in the matrix move from partial only when all child behavior is either supported or reported.
- Real-world table fixtures pass perceptual/structural gate or record accepted residuals.
- Table unsupported features appear in JSON with object provenance.

## Verification

```text
go test ./internal/render -run 'Test.*Table|TestRenderGraphicFrame' -count=1
PUPPT_MICRO_FIXTURE_MANIFEST=<focused-table-manifest> go test ./internal/render -run TestMicroFixtureManifestComparison -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Update matrix rows for table grid, cell, row, properties, styles, borders, and
table text behavior.
