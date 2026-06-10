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
| Validation | Supported for package structure and relationship reachability | Validate tests |
| Review summaries | Supported | Review and acceptance tests |
| PNG slide rendering | Partial | `puppt render` writes PNGs and reports unsupported visible objects |
| Renderer core package/geometry rows | Supported | M12 matrix audit keeps only 16 core-static rows Supported |
| Static pictures, shapes, connectors, text, tables, effects, diagrams, and lockedCanvas/GVML child content | Partial | Source-backed subsets render; remaining visible gaps stay Partial with fixture or explicit partial evidence |
| Charts, SmartArt layout, OLE app content, ActiveX controls, content parts, audio, video, and embedded fonts | Preserve/report or Partial | Charts and SmartArt layout are tracked as Partial static-rendering work; OLE app content, controls, content parts, audio/video playback, and embedded fonts are preserved/reported |
| Table cell 3-D | Partial/report | Detected from source XML and reported during render; still-image 3-D cell rendering remains feasible static rendering work |
| Schema rows without coverage evidence | Reconciled | M12 matrix audit has 0 `Unimplemented / no evidence` rows; remaining rows are Supported, Partial, preserve/report, or out of renderer scope |
| Macro/VBA editing | Non-v1 | Warn and preserve where detected |
| Chart/SmartArt editing | Non-v1 | Warn and preserve where detected |
| Rich media metadata | Non-v1 | Relationship-level metadata only |
| PowerPoint visual fidelity guarantees | In progress | M12 exact gate still fails: 61/61 real-world reference slides differ |
| Clean object fixture parity | In progress | M12 clean-fixture accounting found 59 tracked failures, 0 passing |

Unsupported operations must fail explicitly or be reported in `unsupported`; Puppt must not silently flatten or drop unknown content.
