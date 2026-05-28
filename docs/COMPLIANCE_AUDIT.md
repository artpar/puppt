# Doctrine Compliance Audit

This audit records current alignment with `swe_skill.md`. It is not a waiver. It is the handoff point for remaining compliance work.

## Satisfied Or Substantially Satisfied

- Product core is Go.
- CLI entrypoint is thin.
- Business logic lives under `internal`.
- PPTX reader/writer and mutation path are Puppt-owned.
- Core package handling uses Go standard library ZIP/XML/JSON/filesystem primitives.
- Cobra is isolated to CLI routing and documented in a dependency decision.
- Edit workflows plan before mutation.
- Ambiguity, no-match, unsupported operations, and validation failures are explicit.
- Deterministic fixture generation and golden inspection output exist.
- Round-trip tests cover requested edits and selected preservation cases.

## Previously Identified Gaps Addressed In This Pass

- Added direct tests for `cmd/puppt`, `internal/fixtures`, `internal/model`, and `internal/report`.
- Added `Makefile` build and verification entrypoints.
- Added build/release, state handoff, and technical KT docs.
- Downgraded status wording from production-complete to fixture-backed checkpoint implementation.

## Remaining Gaps

- Real-world deck coverage is still limited.
- General validation does not yet accept expected-content assertions.
- Visual fidelity is not verified by rendering.
- Advanced non-text object extraction and rich media metadata remain incomplete.
- Release/rollback policy exists only as a local-build handoff, not a full production release process.

## Required Verification

```sh
make verify
```
