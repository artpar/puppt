# Puppt Commands

All command JSON uses the `puppt.v1` envelope with stable top-level fields: `schema_version`, `command`, `status`, `input`, `output`, `warnings`, `errors`, and `summary`.

## Inspect

```sh
puppt inspect input.pptx --json
```

Returns slide order, titles, visible text, notes, metadata, image/media references, layouts, repeated text, and warnings.

## Plan

```sh
puppt plan input.pptx --edit edit.json --json
```

Resolves targets and reports whether a requested edit is ready, ambiguous, unsupported, or has no match. It does not write a deck.

## Edit

```sh
puppt edit input.pptx --edit edit.json --out output.pptx --json
```

Applies supported edits after planning, writes a new `.pptx`, validates it, and reports changes.

## Create

```sh
puppt create --input deck.json --out output.pptx --json
```

Builds a deterministic editable `.pptx` from structured JSON, then validates the output.

## Validate

```sh
puppt validate input.pptx --json
```

Checks package structure, required parts, slide reachability, and supported relationship targets.

## Review

```sh
puppt review input.pptx --changes changes.json --json
```

Reads a prior command result or a `changes` array, inspects and validates the deck, and emits an agent-readable plus human-readable change summary.

## Render

```sh
puppt render input.pptx --slide 1 --out slide-1.png --json
```

Renders one 1-based slide to PNG using Puppt-owned `.pptx` package interpretation and Go image output. The default output resolution is 72 DPI, matching the checked-in Apple Notes reference export dimensions for the real-world fixtures; use `--dpi 96` for 1280x720 alternate comparison output. The renderer does not shell out to LibreOffice, PowerPoint, Keynote, browser renderers, or external office tools. Unsupported visible objects are reported in `unsupported`; they must not be silently omitted.

Set `PUPPT_FONT_MAP` to pin exact renderer font files for environments that provide Office-compatible fonts. Entries are semicolon-separated `family=/path/to/font.ttf` values; style-specific entries use `family:bold=...`, `family:italic=...`, or `family:bolditalic=...`. When no exact font is available, Puppt uses deterministic configured substitutes for known Office theme fonts and reports the substitution in JSON.
