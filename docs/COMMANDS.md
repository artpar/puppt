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
