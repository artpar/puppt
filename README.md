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

## Examples

The examples below use the local binary and show compact excerpts of real JSON
output. `jq` is only used to keep the displayed output short.

Create a small editable deck specification:

```sh
mkdir -p .tmp/readme-examples
```

```json
{
  "metadata": {
    "title": "Quarterly Review",
    "author": "Puppt",
    "subject": "Q4"
  },
  "slides": [
    {
      "layout": "title",
      "title": "Quarterly Review"
    },
    {
      "layout": "title_body",
      "title": "Metrics",
      "body": "Revenue and retention moved together.",
      "bullets": [
        "Revenue up",
        "Retention stable"
      ],
      "notes": "Pause before the metrics."
    }
  ]
}
```

Save that as `.tmp/readme-examples/deck.json`, then create the `.pptx`:

```sh
./bin/puppt create \
  --input .tmp/readme-examples/deck.json \
  --out .tmp/readme-examples/quarterly.pptx \
  --json |
  jq '{schema_version, command, status, output, summary, validation}'
```

Output:

```json
{
  "schema_version": "puppt.v1",
  "command": "create",
  "status": "ok",
  "output": ".tmp/readme-examples/quarterly.pptx",
  "summary": {
    "human": "Created 2 slide deck."
  },
  "validation": {
    "valid": true,
    "warnings": [],
    "errors": []
  }
}
```

Inspect the deck:

```sh
./bin/puppt inspect .tmp/readme-examples/quarterly.pptx --json |
  jq '{
    schema_version,
    command,
    status,
    summary,
    inspection: {
      slide_count: .inspection.slide_count,
      metadata: .inspection.metadata,
      slides: [
        .inspection.slides[] |
        {number, title, visible_text: [.visible_text[].text]}
      ]
    }
  }'
```

Output:

```json
{
  "schema_version": "puppt.v1",
  "command": "inspect",
  "status": "ok",
  "summary": {
    "human": "Found 2 slides."
  },
  "inspection": {
    "slide_count": 2,
    "metadata": {
      "title": "Quarterly Review",
      "author": "Puppt",
      "subject": "Q4"
    },
    "slides": [
      {
        "number": 1,
        "title": "Quarterly Review",
        "visible_text": [
          "Quarterly Review"
        ]
      },
      {
        "number": 2,
        "title": "Metrics",
        "visible_text": [
          "Metrics",
          "Revenue and retention moved together.Revenue upRetention stable"
        ]
      }
    ]
  }
}
```

Plan an edit before writing. Save this as `.tmp/readme-examples/edit.json`:

```json
{
  "operation": "replace_text",
  "target": {
    "type": "visible_text",
    "text": "Metrics"
  },
  "replacement": "Q4 Metrics"
}
```

```sh
./bin/puppt plan \
  .tmp/readme-examples/quarterly.pptx \
  --edit .tmp/readme-examples/edit.json \
  --json |
  jq '{
    schema_version,
    command,
    status,
    summary,
    plan: {
      operation: .plan.operation,
      status: .plan.status,
      message: .plan.message,
      matches: .plan.matches,
      replacement: .plan.replacement
    }
  }'
```

Output:

```json
{
  "schema_version": "puppt.v1",
  "command": "plan",
  "status": "ok",
  "summary": {
    "human": "Planned replace_text for 1 target(s)."
  },
  "plan": {
    "operation": "replace_text",
    "status": "ready",
    "message": "matched 1 target",
    "matches": [
      {
        "slide_number": 2,
        "slide_id": "ppt/slides/slide2.xml",
        "object_id": "ppt/slides/slide2.xml#shape-2",
        "kind": "visible_text",
        "text": "Metrics"
      }
    ],
    "replacement": "Q4 Metrics"
  }
}
```

Apply the edit and keep the JSON result as a review artifact:

```sh
./bin/puppt edit \
  .tmp/readme-examples/quarterly.pptx \
  --edit .tmp/readme-examples/edit.json \
  --out .tmp/readme-examples/quarterly-edited.pptx \
  --json |
  tee .tmp/readme-examples/edit-result.json |
  jq '{schema_version, command, status, output, summary, changes, validation}'
```

Output:

```json
{
  "schema_version": "puppt.v1",
  "command": "edit",
  "status": "ok",
  "output": ".tmp/readme-examples/quarterly-edited.pptx",
  "summary": {
    "human": "Applied replace_text with 1 change(s)."
  },
  "changes": [
    {
      "slide_number": 2,
      "object_id": "ppt/slides/slide2.xml#shape-2",
      "message": "Replaced 1 text match(es)."
    }
  ],
  "validation": {
    "valid": true,
    "warnings": [],
    "errors": []
  }
}
```

Validate the edited deck:

```sh
./bin/puppt validate .tmp/readme-examples/quarterly-edited.pptx --json |
  jq '{schema_version, command, status, summary, validation}'
```

Output:

```json
{
  "schema_version": "puppt.v1",
  "command": "validate",
  "status": "ok",
  "summary": {
    "human": "Validation passed."
  },
  "validation": {
    "valid": true,
    "warnings": [],
    "errors": []
  }
}
```

Review the edited deck against the saved edit result:

```sh
./bin/puppt review \
  .tmp/readme-examples/quarterly-edited.pptx \
  --changes .tmp/readme-examples/edit-result.json \
  --json |
  jq '{schema_version, command, status, summary, changes, validation}'
```

Output:

```json
{
  "schema_version": "puppt.v1",
  "command": "review",
  "status": "ok",
  "summary": {
    "human": "Reviewed 2 slide deck with 1 reported change(s) on slide 2; skipped 0, ambiguous 0, unsupported 0; validation passed."
  },
  "changes": [
    {
      "slide_number": 2,
      "object_id": "ppt/slides/slide2.xml#shape-2",
      "message": "Replaced 1 text match(es)."
    }
  ],
  "validation": {
    "valid": true,
    "warnings": [],
    "errors": []
  }
}
```

Render a slide to PNG. Rendering reports a `partial` status when visible
objects are preserved in the deck but not painted by the current renderer:

```sh
./bin/puppt render \
  .tmp/readme-examples/quarterly-edited.pptx \
  --slide 2 \
  --out .tmp/readme-examples/slide-2.png \
  --json |
  jq '{schema_version, command, status, output, summary, render, unsupported}'
```

Output:

```json
{
  "schema_version": "puppt.v1",
  "command": "render",
  "status": "partial",
  "output": ".tmp/readme-examples/slide-2.png",
  "summary": {
    "human": "Rendered slide 2 with 2 unsupported object(s)."
  },
  "render": {
    "slide_number": 2,
    "slide_part": "ppt/slides/slide2.xml",
    "width": 960,
    "height": 540
  },
  "unsupported": [
    {
      "code": "render_unsupported_object",
      "message": "shape object \"Title 1\" contains text and is not rendered yet",
      "part": "ppt/slides/slide2.xml"
    },
    {
      "code": "render_unsupported_object",
      "message": "shape object \"Body 1\" contains text and is not rendered yet",
      "part": "ppt/slides/slide2.xml"
    }
  ]
}
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
