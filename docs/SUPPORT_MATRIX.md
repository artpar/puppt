# Puppt v1 Support Matrix

| Capability | v1 status | Evidence |
|---|---|---|
| Modern `.pptx` package open | Supported | `internal/pptx` reader tests |
| Legacy `.ppt` | Out of scope | Explicit `.pptx` extension validation |
| Slide order and titles | Supported | Inspection and acceptance tests |
| Shape-level visible text | Supported for simple text shapes | Inspection golden and edit tests |
| Speaker notes inspection/update | Supported when notes parts exist | Notes inspection and mutation tests |
| Image/media reference inspection | Supported at relationship level | Inspection media tests |
| Image replacement | Supported for explicit image targets | Edit image replacement tests |
| Layout/master references | Supported where relationships exist | Inspection tests |
| Repeated visible text | Supported | Inspection repeated-text tests |
| Targeted text replacement | Supported | Edit round-trip tests |
| Deck-wide text replacement | Supported | Exact match-count tests |
| Slide add/delete/move/duplicate | Supported for fixture-safe package structures | Slide operation tests |
| Metadata update | Supported for title, author, subject | Edit metadata tests |
| Simple editable text boxes/shapes | Supported | Simple addition tests |
| Deck creation | Supported for structured JSON | Creation and CLI tests |
| Validation | Supported for package structure and relationship reachability | Validate tests |
| Review summaries | Supported | Review and acceptance tests |
| Preview rendering | Non-v1 | No rendering path exists |
| Macro/VBA editing | Non-v1 | Warn and preserve where detected |
| Chart/SmartArt editing | Non-v1 | Warn and preserve where detected |
| Rich media metadata | Non-v1 | Relationship-level metadata only |
| PowerPoint visual fidelity guarantees | Non-v1 | Output is editable-package focused |

Unsupported operations must fail explicitly or be reported in `unsupported`; Puppt must not silently flatten or drop unknown content.
