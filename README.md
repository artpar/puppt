# Puppt

Puppt is an agent-first Go tool for inspecting, editing, creating, validating, and reviewing modern PowerPoint `.pptx` files while preserving editability and unrelated deck content.

The binding product and engineering documents are:

- `goal.md`
- `PRODUCT_VISION.md`
- `USER_EXPERIENCE.md`
- `project-ops.md`
- `swe_skill.md`

## Current State

Puppt is at Checkpoint 8: Review and v1 Hardening. Inspection, edit planning, supported mutations, image replacement, simple editable additions, and structured deck creation are implemented for deterministic `.pptx` fixtures, with post-edit validation.

## Development

Run the baseline test suite:

```sh
go test ./...
```

Run CLI help:

```sh
go run ./cmd/puppt --help
```

## Implementation Language

The product core, CLI, public API surface, tests, and fixtures are implemented in Go.
