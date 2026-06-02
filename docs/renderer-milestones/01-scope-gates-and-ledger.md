# M01 Scope, Gates, And Ledger

## Objective

Freeze the renderer objective so all later work has one definition of done:
static PPTX rendering from OOXML/DrawingML source semantics, implemented by
Puppt-owned parser/lowering/primitive boundaries, validated by deterministic
fixtures and perceptual real-world gates.

## Inputs

- `docs/RENDERER_COMPLETION_GOAL.md`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/SUPPORT_MATRIX.md`
- `docs/RENDERING.md`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`

## In Scope

- Define the supported static-rendering target.
- Separate supported, partial, unsupported, out-of-scope, and unimplemented.
- Define exact-pixel, structural, and perceptual gates.
- Define what evidence moves a matrix row between statuses.
- Define what final completion is not allowed to claim.

## Out Of Scope

- Implement renderer primitives.
- Change rendering output.
- Add new dependencies except documentation/test tooling if unavoidable.

## Required Work

1. Reconcile `RENDERER_COMPLETION_GOAL.md`, support matrix, and coverage matrix.
2. Make the matrix status definitions unambiguous.
3. Add a policy for synthetic fixtures versus real-world fixtures.
4. Add a policy for perceptual metrics: validation only, never implementation strategy.
5. Add a final evidence packet template.
6. Ensure unsupported clauses require explicit reporting or explicit out-of-scope status.

## Acceptance Criteria

- `docs/RENDERER_COMPLETION_GOAL.md` names the static renderer scope.
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md` has complete schema inventory for current scope.
- Every status has a promotion rule.
- The final gate distinguishes spec conformance, renderer compatibility, CLI/JSON stability, and dependency boundary.
- No milestone after this one needs to decide what "complete" means.

## Verification

```text
python3 tools/generate_ooxml_drawingml_audit.py
git diff --check
```

No Go tests are required unless docs or tooling changes touch code.

## Closeout

Update:

- `docs/RENDERER_COMPLETION_GOAL.md`
- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `docs/RENDERER_COMPLETION_CHECKLIST.md`

Record the exact decisions that future milestones must not reopen.
