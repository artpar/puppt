# M11 Diagrams, Charts, And Embedded Content

## Objective

Make explicit render/preserve/report decisions for every non-basic graphic
payload: diagrams/SmartArt, charts, OLE, controls, content parts, and rich media.

## Inputs

- `dml-diagram.xsd`
- `dml-chart.xsd`
- `dml-chartDrawing.xsd`
- `pml.xsd` graphic frame and content-part declarations
- current graphic-frame and diagram code

## In Scope

- Simple diagram drawing parts that lower into shape/text primitives.
- SmartArt data/layout policy.
- Chart policy: unsupported preserve/report, fallback image use, or chart renderer.
- OLE, controls, content parts, media, and unknown graphic payloads.
- JSON/reporting behavior and preservation guarantees.

## Out Of Scope

- Building a chart engine unless this milestone explicitly changes chart scope.
- Editing embedded applications.
- Playback of rich media.

## Required Work

1. Decide per payload family: render, fallback-render, preserve/report, or out-of-scope.
2. Add fixtures for each decision.
3. Add relationship/content-type detection tests.
4. Ensure unsupported visible payloads produce precise JSON.
5. Ensure edit/write paths preserve unsupported payloads where possible.

## Acceptance Criteria

- No chart/SmartArt/OLE/media row remains ambiguous.
- Supported diagram subsets have render fixtures.
- Unsupported payloads are preserved or rejected before mutation, and reported during render.
- Matrix rows for chart/diagram/content parts reflect the decision.

## Verification

```text
go test ./internal/render -run 'Test.*Diagram|Test.*GraphicFrame|Test.*Chart|Test.*Unsupported' -count=1
go test ./internal/... -run 'Test.*Preserve|Test.*Unsupported|Test.*Validate' -count=1
git diff --check
```

## Closeout

Update support matrix, coverage matrix, and failure modes so users know exactly
what happens to charts, SmartArt, OLE, controls, and media.
