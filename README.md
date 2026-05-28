# Puppt

Puppt is an agent-first Go tool for inspecting, editing, creating, validating, and reviewing modern PowerPoint `.pptx` files while preserving editability and unrelated deck content.

The binding product and engineering documents are:

- `goal.md`
- `PRODUCT_VISION.md`
- `USER_EXPERIENCE.md`
- `project-ops.md`
- `swe_skill.md`

## Current State

Puppt is at Checkpoint 0: Repository Foundation. The CLI skeleton and package layout exist, but `.pptx` workflows are not implemented yet.

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
