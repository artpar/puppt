# Renderer Milestone Index

This directory breaks `docs/RENDERER_COMPLETION_GOAL.md` into independent,
verifiable milestones. Each milestone is intended to be chased as a standalone
goal in a separate session.

The order is binding unless a milestone explicitly says it may run in parallel.
Do not mark a milestone complete because a screenshot improved. Completion
requires source semantics, fixture proof, no-regression evidence, and updates to
the coverage matrix.

## Milestone Order

| Milestone | Goal | Primary Output |
|---|---|---|
| [M01](01-scope-gates-and-ledger.md) | Freeze scope, gates, and coverage accounting. | Stable goal/checklist/matrix policy. |
| [M02](02-fixtures-metrics-and-work-queues.md) | Make fixtures, metrics, and queues executable. | Reproducible object/spec/perceptual gates. |
| [M03](03-render-scene-ir.md) | Finish the Puppt-owned render scene boundary. | Stable primitives for every supported object family. |
| [M04](04-coordinates-transforms-and-clipping.md) | Make coordinate spaces, transforms, z-order, and clipping correct. | Shared geometry foundation. |
| [M05](05-theme-color-fill-and-style-resolution.md) | Resolve themes, colors, fills, and style refs from source semantics. | Shared paint model. |
| [M06](06-geometry-stroke-and-connectors.md) | Implement path geometry, strokes, markers, and connectors. | Production vector backend. |
| [M07](07-pictures-media-and-image-pipeline.md) | Implement picture/media source, color, sampling, masks, and image effects. | Production image backend. |
| [M08](08-text-shaping-layout-and-fonts.md) | Implement text shaping, metrics, layout, bullets, and font fallback. | Production text backend. |
| [M09](09-tables.md) | Implement table layout, styles, borders, text, and reporting. | Production table backend. |
| [M10](10-effects-shadows-and-compositing.md) | Implement effects, shadows, soft edges, alpha, and compositing. | Production effects backend. |
| [M11](11-diagrams-charts-and-embedded-content.md) | Decide and implement diagram/chart/embedded-content boundaries. | Render or honest preserve/report policy. |
| [M12](12-final-conformance-and-release-audit.md) | Prove final conformance for the supported static renderer. | Completion evidence packet. |

## Global Rules

Every milestone must:

- start from OOXML/DrawingML source semantics
- update `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- add or tighten deterministic synthetic fixtures before real-world tuning
- keep unsupported behavior explicit in JSON and human output
- run focused tests before broad tests
- record accepted/rejected decisions in maintained docs

Every milestone must avoid:

- browser, Office, SaaS, or image-conversion renderer dependency paths
- broad parameter searches without a source-backed primitive model
- silently flattening unsupported content
- changing unrelated renderer behavior without object-level evidence
