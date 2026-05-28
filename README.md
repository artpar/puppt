# Puppt

Puppt is an agent-first Go tool for inspecting, editing, creating, validating, and reviewing modern PowerPoint `.pptx` files while preserving editability and unrelated deck content.

The binding product and engineering documents are:

- `goal.md`
- `PRODUCT_VISION.md`
- `USER_EXPERIENCE.md`
- `project-ops.md`
- `swe_skill.md`

## Current State

Puppt's v1 checkpoint sequence is complete. Inspection, edit planning, supported mutations, image replacement, simple editable additions, structured deck creation, validation, and review are implemented for deterministic `.pptx` fixtures.

## Development

Run the baseline test suite:

```sh
go test ./...
```

Run CLI help:

```sh
go run ./cmd/puppt --help
```

## Operation Docs

- [Commands](docs/COMMANDS.md)
- [Create examples](docs/CREATE_EXAMPLES.md)
- [Plan examples](docs/PLAN_EXAMPLES.md)
- [Support matrix](docs/SUPPORT_MATRIX.md)
- [Failure modes](docs/FAILURE_MODES.md)
- [Acceptance workflow](docs/ACCEPTANCE.md)

## Implementation Language

The product core, CLI, public API surface, tests, and fixtures are implemented in Go.
