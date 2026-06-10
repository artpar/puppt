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

## Usage Examples

Puppt works best as a tight loop: inspect the real `.pptx`, plan a targeted
change, write a new editable deck, review the result, and render the slides that
need a visual check.

The screenshots below are committed render outputs from the local renderer
corpus; use the same commands with any media-heavy `.pptx`.

### Inspect

Use `inspect` to turn a slide into stable JSON targets. Here slide 2 has nine
image references and two editable text objects.

```sh
./bin/puppt inspect testdata/realworld-ppts/EPA-generate-2021-presentation.pptx --json |
  jq '{status, slide: (
    .inspection.slides[] |
    select(.number == 2) |
    {
      number,
      title,
      images: (.images | length),
      text_objects: (.visible_text | length),
      title_object: .visible_text[0].object_id
    }
  )}'
```

```json
{
  "status": "ok",
  "slide": {
    "number": 2,
    "title": "Energy 101: The big picture",
    "images": 9,
    "text_objects": 2,
    "title_object": "ppt/slides/slide2.xml#shape-2"
  }
}
```

<img src="docs/assets/readme/epa-generate-slide-2.png" alt="Inspected slide 2 render" width="520">

### Render

Use `render` to produce PNGs from the `.pptx` itself. The JSON tells you which
slide part was painted and where the images were written.

```sh
./bin/puppt render \
  testdata/realworld-ppts/EPA-generate-2021-presentation.pptx \
  --slides 1-3 \
  --dpi 72 \
  --out 'docs/assets/readme/epa-generate-slide-{slide}.png' \
  --json |
  jq '{status, outputs, renders, unsupported_count:(.unsupported|length)}'
```

```json
{
  "status": "ok",
  "outputs": [
    "docs/assets/readme/epa-generate-slide-1.png",
    "docs/assets/readme/epa-generate-slide-2.png",
    "docs/assets/readme/epa-generate-slide-3.png"
  ],
  "renders": [
    {"slide_number":1,"slide_part":"ppt/slides/slide1.xml","width":960,"height":540},
    {"slide_number":2,"slide_part":"ppt/slides/slide2.xml","width":960,"height":540},
    {"slide_number":3,"slide_part":"ppt/slides/slide3.xml","width":960,"height":540}
  ],
  "unsupported_count": 0
}
```

<p>
  <img src="docs/assets/readme/epa-generate-slide-1.png" alt="Rendered slide 1" width="260">
  <img src="docs/assets/readme/epa-generate-slide-2.png" alt="Rendered slide 2" width="260">
  <img src="docs/assets/readme/epa-generate-slide-3.png" alt="Rendered slide 3" width="260">
</p>

### Edit

Use the object id from `inspect` to mutate one editable object without touching
unrelated slide content.

Save this as `.tmp/readme-edit-visual/replace-title.json`:

```json
{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide2.xml#shape-2"
  },
  "replacement": "Energy 101: Edited with Puppt"
}
```

```sh
./bin/puppt edit \
  testdata/realworld-ppts/EPA-generate-2021-presentation.pptx \
  --edit .tmp/readme-edit-visual/replace-title.json \
  --out .tmp/readme-edit-visual/epa-generate-edited.pptx \
  --json |
  jq '{status, summary, changes, validation}'
```

```json
{
  "status": "ok",
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

<table>
  <tr>
    <th>Before</th>
    <th>After</th>
  </tr>
  <tr>
    <td><img src="docs/assets/readme/epa-generate-slide-2.png" alt="Before text edit" width="390"></td>
    <td><img src="docs/assets/readme/epa-generate-slide-2-after-edit.png" alt="After text edit" width="390"></td>
  </tr>
</table>

### Create And Review

Puppt can also create editable decks from JSON and then run the same inspection,
planning, review, and rendering steps on the generated `.pptx`.

```sh
./bin/puppt create --input deck.json --out deck.pptx --json
./bin/puppt plan deck.pptx --edit rename.json --json
./bin/puppt edit deck.pptx --edit rename.json --out edited.pptx --json > edit-result.json
./bin/puppt review edited.pptx --changes edit-result.json --json
./bin/puppt render edited.pptx --slides 1-3 --out renders --json
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
