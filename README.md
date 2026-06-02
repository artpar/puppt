# Puppt

[![Release](https://img.shields.io/github/v/release/artpar/puppt?include_prereleases&sort=semver)](https://github.com/artpar/puppt/releases)
[![CI](https://github.com/artpar/puppt/actions/workflows/ci.yml/badge.svg)](https://github.com/artpar/puppt/actions/workflows/ci.yml)
[![Release workflow](https://github.com/artpar/puppt/actions/workflows/release.yml/badge.svg)](https://github.com/artpar/puppt/actions/workflows/release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/artpar/puppt.svg)](https://pkg.go.dev/github.com/artpar/puppt)
[![Go Report Card](https://goreportcard.com/badge/github.com/artpar/puppt)](https://goreportcard.com/report/github.com/artpar/puppt)

Puppt is a Go CLI for inspecting, editing, creating, validating, reviewing, and
rendering editable PowerPoint `.pptx` files.

It is built for agent and automation workflows where a deck must stay editable:
Puppt reads the PowerPoint Open XML package, plans mutations before writing,
rejects ambiguous or unsupported edits before mutation, and preserves unrelated
deck content wherever the package structure allows it.

## What You Can Do

- Inspect decks and return structured facts as JSON: slide order, titles,
  visible text, notes, media, layouts, masters, metadata, and unsupported
  content signals.
- Plan targeted edits without writing output, so agents and humans can review
  what will change before mutation.
- Edit supported content, including text, notes, metadata, slide order, slide
  add/delete/move/duplicate operations, image replacement, and simple editable
  shape additions.
- Create editable `.pptx` decks from structured input, including title slides,
  section slides, title/body slides, bullet lists, speaker notes, metadata, and
  provided images.
- Validate package structure and expected content after creation or edits.
- Review changes by combining prior command results, inspection facts, skipped
  items, unsupported items, and validation status.
- Render a slide to PNG through Puppt-owned Go code for visual review and
  diagnostics.

## Quick Start

Download release artifacts and checksums from
[GitHub Releases](https://github.com/artpar/puppt/releases).

Install from source:

```sh
go install github.com/artpar/puppt/cmd/puppt@latest
```

Build the local binary:

```sh
make build
```

Inspect a deck:

```sh
puppt inspect deck.pptx --json
```

Plan an edit before writing:

```sh
puppt plan deck.pptx --edit edit.json --json
```

Apply a supported edit:

```sh
puppt edit deck.pptx --edit edit.json --out edited.pptx --json
```

Render a slide:

```sh
puppt render deck.pptx --slide 1 --out slide-1.png --json
```

## Commands

| Command | Use |
| --- | --- |
| `inspect` | Read a `.pptx` deck and return structured facts. |
| `plan` | Resolve targets and validate an edit request without writing output. |
| `edit` | Apply supported targeted edits and write a new `.pptx`. |
| `create` | Create an editable `.pptx` deck from structured input. |
| `validate` | Check package structure and expected content. |
| `review` | Summarize deck changes for agents and human reviewers. |
| `render` | Render one slide to a PNG image. |
| `version` | Print Puppt version information. |

Run command help for exact flags:

```sh
puppt --help
puppt <command> --help
```

During development, use:

```sh
go run ./cmd/puppt --help
```

## Editing Model

Puppt treats `.pptx` files as structured Open XML packages, not as screenshots.
The normal edit flow is:

1. Inspect the deck to find stable targets.
2. Plan the edit and check whether the target is ready, ambiguous, missing, or
   unsupported.
3. Apply the edit only when the plan is supported.
4. Validate the written deck.
5. Review the result as JSON for downstream agents or human reviewers.

Ambiguous targets and unsupported advanced visual edits are rejected before the
deck is mutated. Supported edits are written through Puppt-owned package
handling so unrelated parts of the deck stay intact.

## Rendering

`puppt render` is a Puppt-owned Go renderer. It does not shell out to
LibreOffice, PowerPoint, Keynote, browser renderers, SaaS renderers, or
image-conversion tools.

The renderer currently covers practical static PPTX content including slide
dimensions, backgrounds, themes, layouts and masters, placeholders, pictures,
common image metadata, shape fills and outlines, connectors, text, bullets,
tables, selected shadows/effects, simple diagram fallback drawings, and explicit
JSON reports for content that is not painted or only partially painted.

Renderer parity is still in progress. Puppt is useful for visual review and
diagnostics today, but final renderer conformance is not claimed yet. See
`docs/RENDERING.md`, `docs/RENDERER_COMPLETION_GOAL.md`, and
`docs/RENDERER_COMPLETION_CHECKLIST.md` for the current renderer status and
completion path.

## Current State

Puppt has fixture-backed v1 workflows for inspection, edit planning, supported
mutations, image replacement, simple editable additions, structured deck
creation, validation, review, and rendering. Full production-grade compliance is
not claimed yet.

All required v1 command names are implemented: `inspect`, `plan`, `edit`,
`create`, `validate`, `review`, `render`, and `version`.

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

## Docs

User workflows:

- [Commands](docs/COMMANDS.md)
- [Create examples](docs/CREATE_EXAMPLES.md)
- [Plan examples](docs/PLAN_EXAMPLES.md)
- [Failure modes](docs/FAILURE_MODES.md)
- [Acceptance workflow](docs/ACCEPTANCE.md)

Capability and status:

- [Status](docs/STATUS.md)
- [Support matrix](docs/SUPPORT_MATRIX.md)
- [Rendering](docs/RENDERING.md)

Engineering and completion:

- [Renderer completion goal](docs/RENDERER_COMPLETION_GOAL.md)
- [Renderer milestones](docs/renderer-milestones/00-INDEX.md)
- [Build and release](docs/BUILD_RELEASE.md)
- [Technical KT](docs/TECHNICAL_KT.md)

## Implementation Language

The product core, CLI, public API surface, tests, and fixtures are implemented
in Go.
