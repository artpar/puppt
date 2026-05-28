# Technical KT

## Architecture

Puppt is a Go CLI with business logic in internal packages. The command layer is intentionally thin and delegates to workflow packages.

| Package | Responsibility |
|---|---|
| `cmd/puppt` | CLI process entrypoint |
| `internal/cli` | Cobra command wiring and stream handling |
| `internal/pptx` | Owned Open Packaging Convention reader/writer |
| `internal/inspect` | Structured deck inspection |
| `internal/target` | Target resolution and ambiguity detection |
| `internal/edit` | Edit planning and mutation workflows |
| `internal/create` | Structured JSON deck creation |
| `internal/validate` | Package structure and relationship validation |
| `internal/review` | Change review summaries |
| `internal/report` | Stable JSON output |
| `internal/model` | Shared `puppt.v1` result and domain structs |
| `internal/fixtures` | Deterministic test deck builders |

## Data Flow

Inspection:

```text
CLI -> internal/inspect -> internal/pptx.Open -> model.CommandResult
```

Edit:

```text
CLI -> internal/edit.Plan -> internal/inspect -> internal/target
CLI -> internal/edit.Apply -> internal/pptx.Open -> mutate package parts -> internal/pptx.Write -> internal/validate
```

Create:

```text
CLI -> internal/create -> build package parts -> internal/pptx.Write -> internal/validate -> internal/inspect content check
```

Review:

```text
CLI -> internal/review -> read changes JSON -> internal/inspect + internal/validate -> model.CommandResult
```

## Core Invariants

- Only modern `.pptx` ZIP/Open XML packages are supported.
- The authoritative package reader/writer and mutation path are owned by Puppt.
- Every edit must plan and resolve targets before mutation.
- Ambiguous and no-match targets must not mutate output.
- Unsupported operations must be rejected or reported explicitly.
- JSON output uses the additive `puppt.v1` envelope.
- Unknown package parts should be preserved where feasible.

## JSON Compatibility

The stable envelope is:

```json
{
  "schema_version": "puppt.v1",
  "command": "inspect",
  "status": "ok",
  "input": "input.pptx",
  "output": null,
  "warnings": [],
  "errors": [],
  "summary": {
    "human": "..."
  }
}
```

Fields may be added, but existing v1 fields should not be removed or repurposed without a compatibility decision.

## Validation Boundary

`internal/validate` checks core package readability and relationship target reachability. Edit and create workflows add expected-content checks after writing. General-purpose expected-content assertions are not yet part of `puppt validate`.

## Dependency Boundary

Cobra owns CLI routing only. PPTX reading, writing, inspection, mutation, creation, validation, and review logic remain in Puppt-owned Go code using standard library ZIP/XML/JSON primitives.

## Known Gaps

- Real-world deck fixture breadth.
- Advanced object extraction.
- Rich media metadata.
- Notes part creation for existing slides.
- Rendered visual validation.
- Release packaging and CI.
