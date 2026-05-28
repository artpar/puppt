# State Handoff

## Current State

Puppt has implemented v1 checkpoint workflows for deterministic fixtures, but it should not be described as fully production-grade against arbitrary real-world `.pptx` files.

Implemented command surface:

- `puppt inspect <input.pptx> --json`
- `puppt plan <input.pptx> --edit <edit.json> --json`
- `puppt edit <input.pptx> --edit <edit.json> --out <output.pptx> --json`
- `puppt create --input <deck.json> --out <output.pptx> --json`
- `puppt validate <input.pptx> --json`
- `puppt review <input.pptx> --changes <changes.json> --json`
- `puppt version`

## Verification Last Expected

```sh
make verify
```

## Key Risks

- Real-world deck coverage is still limited compared with PowerPoint's full Open XML surface.
- Validation is structural plus workflow-specific content checks; it is not a rendered visual comparison.
- Rich media metadata, advanced non-text extraction, charts, SmartArt, OLE internals, and macro editing are not implemented.
- Notes update requires an existing notes relationship; creating notes parts for existing slides is not implemented.
- Mutated XML parts are re-encoded; unrelated package parts are preserved where feasible and structurally validated.

## Next Engineering Actions

1. Add real-world minimized fixture decks with provenance notes.
2. Expand validation to accept expected-content assertions directly.
3. Add support for creating notes relationships on existing slides.
4. Add CI and release packaging.
5. Decide whether richer Open XML helpers are justified for non-core validation or schema checks.

## Important Docs

- `goal.md`
- `project-ops.md`
- `swe_skill.md`
- `docs/TECHNICAL_KT.md`
- `docs/BUILD_RELEASE.md`
- `docs/SUPPORT_MATRIX.md`
- `docs/FAILURE_MODES.md`
