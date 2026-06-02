# M02 Fixtures, Metrics, And Work Queues

## Objective

Make the proof system executable before more renderer implementation work:
schema-row queues, deterministic spec fixtures, object-attributed real-world
fixtures, perceptual metrics, and no-regression gates.

## Inputs

- `docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md`
- `internal/render/render_realworld_test.go`
- existing object-debug and micro-fixture harnesses
- `docs/RENDERER_COMPLETION_CHECKLIST.md`

## In Scope

- Generate work queues from the coverage matrix.
- Add test helpers for structural, exact-pixel diagnostic, and perceptual checks.
- Add an executable "all clean fixtures" gate.
- Add or formalize spec-fixture manifests.
- Ensure fixtures identify source schema rows.

## Out Of Scope

- Implement visual fixes.
- Rebaseline real-world references without explicit approval.
- Claim support for any schema row.

## Required Work

1. Add a machine-readable coverage summary generated from the matrix.
2. Split rows into queues:
   - `core-static`
   - `common-partial`
   - `hard-rendering`
   - `unsupported-preserve`
   - `out-of-scope`
3. Add a spec-fixture manifest format:
   - schema anchor
   - source XML part/path
   - expected semantic model
   - expected render primitive
   - expected unsupported records
4. Add perceptual metric calculations for slide and object crops.
5. Add a full clean-fixture suite runner instead of one-manifest-at-a-time only.
6. Ensure every failure prints artifact paths and source attribution.

## Acceptance Criteria

- One command lists current matrix counts by queue and status.
- One command runs all tracked clean object fixtures.
- One command runs perceptual metrics for the real-world corpus.
- Fixture failures identify schema rows and source XML.
- Existing exact-pixel diff output remains available.

## Verification

```text
go test ./internal/render -run 'TestMicroFixture|TestRendererProductionFailureScoreboard' -count=1
PUPPT_RUN_REALWORLD_RENDER_TESTS=1 go test ./internal/render -run TestRealWorldGoldenComparison -count=1
git diff --check
```

If the real-world gate is expected to fail, the milestone must document the
expected failure shape and prove the metric/artifact output was generated.

## Closeout

Update:

- coverage matrix with fixture/queue metadata
- renderer checklist with commands and results
- experiment log if any diagnostic gate rejects an approach
