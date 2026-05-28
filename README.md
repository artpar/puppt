# Puppt

Puppt is an agent-first Go tool for inspecting, editing, creating, validating, and reviewing modern PowerPoint `.pptx` files while preserving editability and unrelated deck content.

The binding product and engineering documents are:

- `goal.md`
- `PRODUCT_VISION.md`
- `USER_EXPERIENCE.md`
- `project-ops.md`
- `swe_skill.md`

## Current State

Puppt has fixture-backed v1 checkpoint workflows for inspection, edit planning, supported mutations, image replacement, simple editable additions, structured deck creation, validation, and review. Full production-grade compliance is not claimed yet; see `docs/HANDOFF.md` and `docs/COMPLIANCE_AUDIT.md`.

## Development

Run the baseline test suite:

```sh
go test ./...
```

Build the local binary:

```sh
make build
```

Run the repository verification handoff:

```sh
make verify
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
- [Build and release](docs/BUILD_RELEASE.md)
- [State handoff](docs/HANDOFF.md)
- [Technical KT](docs/TECHNICAL_KT.md)
- [Doctrine compliance audit](docs/COMPLIANCE_AUDIT.md)

## Implementation Language

The product core, CLI, public API surface, tests, and fixtures are implemented in Go.
