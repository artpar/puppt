# Failure Modes

Puppt favors explicit failures over broad best-effort behavior.

| Failure | Behavior |
|---|---|
| Non-`.pptx` input | Fails before package read |
| Invalid ZIP/package | Fails with package context |
| Missing content types/root relationships/presentation parts | Fails validation/open |
| Missing slide relationship target | `validate` returns `invalid` with `missing_relationship_target` |
| Unsupported edit operation | `plan`/`edit` return `unsupported` |
| Unsupported target type or operation-target mismatch | Rejected before mutation |
| No matching target | `plan`/`edit` return `no_match` |
| Ambiguous target | `plan`/`edit` return `ambiguous` |
| Unsupported advanced visual edit | Rejected before mutation |
| Output validation failure | Command returns non-OK status with validation errors |

Known non-v1 gaps:

- Advanced non-text object extraction.
- Rich media metadata such as dimensions and durations.
- Macro, chart, SmartArt, and OLE editing.
- Preview rendering.
- Legacy binary `.ppt` support.
- Design-rich generated layouts beyond the deterministic editable layouts in v1.
