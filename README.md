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
output from this checkout. `jq` is only used to keep the displayed output short.

### Board Deck Automation

This example creates a five-slide launch review deck, plans a deck-wide rename,
applies seven targeted mutations, reviews the combined change artifact, and
renders a slide range.

```sh
mkdir -p .tmp/readme-showoff/specs
```

Create `.tmp/readme-showoff/deck.json`:

```json
{
  "metadata": {
    "title": "Launch Board Review",
    "author": "Puppt Automation",
    "subject": "FY27 GTM",
    "keywords": "draft,launch,board"
  },
  "slides": [
    {
      "layout": "title",
      "title": "Launch Roadmap"
    },
    {
      "layout": "section",
      "title": "Executive Snapshot",
      "notes": "Open with the decision ask."
    },
    {
      "layout": "title_body",
      "title": "Launch Roadmap",
      "body": "North America pilot in Q1. EMEA expansion in Q2. APAC scale-up in Q3.",
      "bullets": [
        "Three-region rollout",
        "Partner channel readiness",
        "Security review complete"
      ],
      "notes": "Call out dependencies by region."
    },
    {
      "layout": "title_body",
      "title": "Risk Register",
      "body": "Top risks: partner onboarding, pricing approvals, and localization.",
      "bullets": [
        "Mitigation owners assigned",
        "Weekly launch room active",
        "Escalation path agreed"
      ],
      "notes": "Do not over-index on low-probability risks."
    },
    {
      "layout": "title_body",
      "title": "Board Ask",
      "body": "Approve phased launch funding and hiring backfill.",
      "bullets": [
        "Approve pilot budget",
        "Confirm executive sponsor",
        "Authorize launch communications"
      ],
      "notes": "Close with the approval sequence."
    }
  ]
}
```

Create the editable `.pptx`:

```sh
./bin/puppt create \
  --input .tmp/readme-showoff/deck.json \
  --out .tmp/readme-showoff/showcase-00.pptx \
  --json |
  jq '{command,status,output,summary,validation}'
```

Output:

```json
{
  "command": "create",
  "status": "ok",
  "output": ".tmp/readme-showoff/showcase-00.pptx",
  "summary": {
    "human": "Created 5 slide deck."
  },
  "validation": {
    "valid": true,
    "warnings": [],
    "errors": []
  }
}
```

Create these edit specs under `.tmp/readme-showoff/specs/`:

```text
01-rename-roadmap.json
{
  "operation": "replace_text",
  "target": {"type": "visible_text", "scope": "deck", "text": "Launch Roadmap"},
  "replacement": "FY27 Launch Roadmap"
}

02-update-notes.json
{
  "operation": "update_notes",
  "target": {"type": "notes", "slide_number": 3},
  "replacement": "Narrate the region sequence: pilot, expand, scale."
}

03-update-subject.json
{
  "operation": "update_metadata",
  "target": {"type": "metadata", "property": "subject"},
  "replacement": "FY27 launch approval packet"
}

04-add-callout.json
{
  "operation": "add_text_box",
  "target": {"type": "slide_number", "slide_number": 4},
  "replacement": "Decision gate: pricing approval before Q2 expansion"
}

05-add-status-shape.json
{
  "operation": "add_shape",
  "target": {"type": "slide_number", "slide_number": 4},
  "replacement": "Owner: Revenue Ops"
}

06-duplicate-board-ask.json
{
  "operation": "slide_duplicate",
  "target": {"type": "slide_number", "slide_number": 5},
  "insert_after_slide": 5
}

07-move-duplicate.json
{
  "operation": "slide_move",
  "target": {"type": "slide_number", "slide_number": 6},
  "destination_slide_number": 2
}
```

Plan the deck-wide rename before writing:

```sh
./bin/puppt plan \
  .tmp/readme-showoff/showcase-00.pptx \
  --edit .tmp/readme-showoff/specs/01-rename-roadmap.json \
  --json |
  jq '{command,status,summary,plan:{
    operation:.plan.operation,
    status:.plan.status,
    message:.plan.message,
    matches:.plan.matches,
    replacement:.plan.replacement
  }}'
```

Output:

```json
{
  "command": "plan",
  "status": "ok",
  "summary": {
    "human": "Planned replace_text for 2 target(s)."
  },
  "plan": {
    "operation": "replace_text",
    "status": "ready",
    "message": "matched 2 targets",
    "matches": [
      {
        "slide_number": 1,
        "slide_id": "ppt/slides/slide1.xml",
        "object_id": "ppt/slides/slide1.xml#shape-2",
        "kind": "visible_text",
        "text": "Launch Roadmap"
      },
      {
        "slide_number": 3,
        "slide_id": "ppt/slides/slide3.xml",
        "object_id": "ppt/slides/slide3.xml#shape-2",
        "kind": "visible_text",
        "text": "Launch Roadmap"
      }
    ],
    "replacement": "FY27 Launch Roadmap"
  }
}
```

Apply the full edit pipeline, writing a validated deck after every step:

```sh
in=.tmp/readme-showoff/showcase-00.pptx
i=1
for spec in .tmp/readme-showoff/specs/*.json; do
  out=$(printf '.tmp/readme-showoff/showcase-%02d.pptx' "$i")
  result=$(printf '.tmp/readme-showoff/edit-%02d.json' "$i")
  ./bin/puppt edit "$in" --edit "$spec" --out "$out" --json > "$result"
  jq -c '{command,status,output,summary,changes,validation}' "$result"
  in="$out"
  i=$((i+1))
done
```

Output:

```json
{"command":"edit","status":"ok","output":".tmp/readme-showoff/showcase-01.pptx","summary":{"human":"Applied replace_text with 2 change(s)."},"changes":[{"slide_number":1,"object_id":"ppt/slides/slide1.xml#shape-2","message":"Replaced 1 text match(es)."},{"slide_number":3,"object_id":"ppt/slides/slide3.xml#shape-2","message":"Replaced 1 text match(es)."}],"validation":{"valid":true,"warnings":[],"errors":[]}}
{"command":"edit","status":"ok","output":".tmp/readme-showoff/showcase-02.pptx","summary":{"human":"Applied update_notes with 1 change(s)."},"changes":[{"slide_number":3,"message":"Updated speaker notes."}],"validation":{"valid":true,"warnings":[],"errors":[]}}
{"command":"edit","status":"ok","output":".tmp/readme-showoff/showcase-03.pptx","summary":{"human":"Applied update_metadata with 1 change(s)."},"changes":[{"message":"Updated metadata property subject."}],"validation":{"valid":true,"warnings":[],"errors":[]}}
{"command":"edit","status":"ok","output":".tmp/readme-showoff/showcase-04.pptx","summary":{"human":"Applied add_text_box with 1 change(s)."},"changes":[{"slide_number":4,"object_id":"ppt/slides/slide4.xml#shape-4","message":"Added editable object to slide 4."}],"validation":{"valid":true,"warnings":[],"errors":[]}}
{"command":"edit","status":"ok","output":".tmp/readme-showoff/showcase-05.pptx","summary":{"human":"Applied add_shape with 1 change(s)."},"changes":[{"slide_number":4,"object_id":"ppt/slides/slide4.xml#shape-5","message":"Added editable object to slide 4."}],"validation":{"valid":true,"warnings":[],"errors":[]}}
{"command":"edit","status":"ok","output":".tmp/readme-showoff/showcase-06.pptx","summary":{"human":"Applied slide_duplicate with 1 change(s)."},"changes":[{"slide_number":6,"object_id":"ppt/slides/slide6.xml","message":"Duplicated slide ppt/slides/slide5.xml from position 5 to position 6 as ppt/slides/slide6.xml."}],"validation":{"valid":true,"warnings":[],"errors":[]}}
{"command":"edit","status":"ok","output":".tmp/readme-showoff/showcase-07.pptx","summary":{"human":"Applied slide_move with 1 change(s)."},"changes":[{"slide_number":2,"object_id":"ppt/slides/slide6.xml","message":"Moved slide ppt/slides/slide6.xml from position 6 to position 2."}],"validation":{"valid":true,"warnings":[],"errors":[]}}
```

Inspect the final deck shape:

```sh
./bin/puppt inspect .tmp/readme-showoff/showcase-07.pptx --json |
  jq '{command,status,summary,inspection:{
    slide_count:.inspection.slide_count,
    subject:.inspection.metadata.subject,
    outline:[.inspection.slides[] | {
      number,title,part,
      text_objects:(.visible_text|length),
      notes:(.notes|length)
    }],
    risk_slide_text:([
      .inspection.slides[] |
      select(.title=="Risk Register") |
      .visible_text[].text
    ])
  }}'
```

Output:

```json
{
  "command": "inspect",
  "status": "ok",
  "summary": {
    "human": "Found 6 slides."
  },
  "inspection": {
    "slide_count": 6,
    "subject": "FY27 launch approval packet",
    "outline": [
      {"number":1,"title":"FY27 Launch Roadmap","part":"ppt/slides/slide1.xml","text_objects":1,"notes":0},
      {"number":2,"title":"Board Ask","part":"ppt/slides/slide6.xml","text_objects":2,"notes":1},
      {"number":3,"title":"Executive Snapshot","part":"ppt/slides/slide2.xml","text_objects":1,"notes":1},
      {"number":4,"title":"FY27 Launch Roadmap","part":"ppt/slides/slide3.xml","text_objects":2,"notes":1},
      {"number":5,"title":"Risk Register","part":"ppt/slides/slide4.xml","text_objects":4,"notes":1},
      {"number":6,"title":"Board Ask","part":"ppt/slides/slide5.xml","text_objects":2,"notes":1}
    ],
    "risk_slide_text": [
      "Risk Register",
      "Top risks: partner onboarding, pricing approvals, and localization.Mitigation owners assignedWeekly launch room activeEscalation path agreed",
      "Decision gate: pricing approval before Q2 expansion",
      "Owner: Revenue Ops"
    ]
  }
}
```

Review the whole mutation set as one artifact:

```sh
jq -s '[.[].changes[]]' .tmp/readme-showoff/edit-*.json \
  > .tmp/readme-showoff/all-changes.json

./bin/puppt review \
  .tmp/readme-showoff/showcase-07.pptx \
  --changes .tmp/readme-showoff/all-changes.json \
  --json |
  jq '{command,status,summary,
    changes_count:(.changes|length),
    touched_slides:([.changes[].slide_number] |
      map(select(. != null and . != 0)) | unique),
    validation
  }'
```

Output:

```json
{
  "command": "review",
  "status": "ok",
  "summary": {
    "human": "Reviewed 6 slide deck with 8 reported change(s) on slide 1, slide 3, slide 4, slide 6, slide 2; skipped 0, ambiguous 0, unsupported 0; validation passed."
  },
  "changes_count": 8,
  "touched_slides": [1, 2, 3, 4, 6],
  "validation": {
    "valid": true,
    "warnings": [],
    "errors": []
  }
}
```

Render the first three slides. Rendering reports `partial` when visible objects
are preserved in the deck but not fully painted by the current renderer:

```sh
./bin/puppt render \
  .tmp/readme-showoff/showcase-07.pptx \
  --slides 1-3 \
  --out .tmp/readme-showoff/renders \
  --json |
  jq '{command,status,summary,outputs,renders,
    unsupported_count:(.unsupported|length)
  }'
```

Output:

```json
{
  "command": "render",
  "status": "partial",
  "summary": {
    "human": "Rendered 3 slides with 4 unsupported object(s)."
  },
  "outputs": [
    ".tmp/readme-showoff/renders/showcase-07/slide-001.png",
    ".tmp/readme-showoff/renders/showcase-07/slide-002.png",
    ".tmp/readme-showoff/renders/showcase-07/slide-003.png"
  ],
  "renders": [
    {"slide_number":1,"slide_part":"ppt/slides/slide1.xml","width":960,"height":540},
    {"slide_number":2,"slide_part":"ppt/slides/slide6.xml","width":960,"height":540},
    {"slide_number":3,"slide_part":"ppt/slides/slide2.xml","width":960,"height":540}
  ],
  "unsupported_count": 4
}
```

For image-heavy decks, the corresponding rendered PNGs are the artifacts you
review next to the JSON. These committed README images were produced by
`puppt render` from real `.pptx` slides in the local render corpus:

```sh
./bin/puppt inspect testdata/realworld-ppts/EPA-generate-2021-presentation.pptx --json |
  jq '{command,status,summary,slides:[
    .inspection.slides[] |
    select(.number >= 1 and .number <= 3) |
    {
      number,
      title,
      images:(.images|length),
      media:(.media|length),
      text_objects:(.visible_text|length)
    }
  ]}'
```

Output:

```json
{
  "command": "inspect",
  "status": "ok",
  "summary": {
    "human": "Found 12 slides."
  },
  "slides": [
    {
      "number": 1,
      "title": " Welcome to GENERATE: The Game of Energy Choices",
      "images": 4,
      "media": 4,
      "text_objects": 2
    },
    {
      "number": 2,
      "title": "Energy 101: The big picture",
      "images": 9,
      "media": 9,
      "text_objects": 2
    },
    {
      "number": 3,
      "title": "Connecting the dots",
      "images": 1,
      "media": 1,
      "text_objects": 21
    }
  ]
}
```

Original inspected slide:

<img src="docs/assets/readme/epa-generate-slide-2.png" alt="Original rendered slide inspected by Puppt" width="520">

```sh
./bin/puppt render \
  testdata/realworld-ppts/EPA-generate-2021-presentation.pptx \
  --slides 1-3 \
  --dpi 72 \
  --out 'docs/assets/readme/epa-generate-slide-{slide}.png' \
  --json |
  jq '{command,status,summary,outputs,renders,unsupported_count:(.unsupported|length)}'
```

Output:

```json
{
  "command": "render",
  "status": "ok",
  "summary": {
    "human": "Rendered 3 slides to docs/assets/readme/epa-generate-slide-{slide}.png."
  },
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
  <img src="docs/assets/readme/epa-generate-slide-1.png" alt="Puppt render of EPA Generate slide 1" width="260">
  <img src="docs/assets/readme/epa-generate-slide-2.png" alt="Puppt render of EPA Generate slide 2" width="260">
  <img src="docs/assets/readme/epa-generate-slide-3.png" alt="Puppt render of EPA Generate slide 3" width="260">
</p>

For an edit workflow, inspect gives the stable object id, `edit` mutates the
deck, and a render makes the before/after visible:

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
  jq '{command,status,output,summary,changes,validation}'
```

Output:

```json
{
  "command": "edit",
  "status": "ok",
  "output": ".tmp/readme-edit-visual/epa-generate-edited.pptx",
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

```sh
./bin/puppt render \
  .tmp/readme-edit-visual/epa-generate-edited.pptx \
  --slide 2 \
  --dpi 72 \
  --out docs/assets/readme/epa-generate-slide-2-after-edit.png \
  --json |
  jq '{command,status,output,summary,render,unsupported_count:(.unsupported|length)}'
```

Output:

```json
{
  "command": "render",
  "status": "ok",
  "output": "docs/assets/readme/epa-generate-slide-2-after-edit.png",
  "summary": {
    "human": "Rendered slide 2 to docs/assets/readme/epa-generate-slide-2-after-edit.png."
  },
  "render": {
    "slide_number": 2,
    "slide_part": "ppt/slides/slide2.xml",
    "width": 960,
    "height": 540
  },
  "unsupported_count": 0
}
```

<table>
  <tr>
    <th>Before edit</th>
    <th>After edit</th>
  </tr>
  <tr>
    <td><img src="docs/assets/readme/epa-generate-slide-2.png" alt="Before text edit" width="390"></td>
    <td><img src="docs/assets/readme/epa-generate-slide-2-after-edit.png" alt="After text edit" width="390"></td>
  </tr>
</table>

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
