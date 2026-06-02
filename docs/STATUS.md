# Puppt Status

## Current State

Puppt `v0.1.0` is published as a versioned CLI release with GitHub Actions CI,
GoReleaser packaging, cross-platform archives, and `SHA256SUMS`.

The command surface is implemented for fixture-backed v1 workflows:

- `inspect`
- `plan`
- `edit`
- `create`
- `validate`
- `review`
- `render`
- `version`

Puppt is useful today for structured `.pptx` inspection, edit planning,
supported mutations, editable deck creation, validation, review summaries, and
diagnostic rendering. Full production-grade behavior for arbitrary real-world
PowerPoint decks is not claimed.

## Implemented

- Puppt-owned Go `.pptx` package reader/writer for core Open Packaging
  Convention structure, content types, relationships, presentation roots, slide
  order, and package preservation.
- JSON command results use the `puppt.v1` envelope.
- Edit workflows plan before mutation and reject ambiguous, no-match, invalid,
  or unsupported requests before writing.
- Supported edits include text, existing speaker notes, metadata, slide
  add/delete/move/duplicate operations, explicit image replacement, and simple
  editable shape additions.
- Creation supports structured JSON decks with title, section, title/body,
  bullet-list slides, notes, metadata, and provided images.
- Validation checks package structure and relationship reachability; edit and
  create workflows add expected-content checks after writing.
- Review combines command results, inspection facts, skipped/ambiguous/
  unsupported items, and validation status.
- Rendering is Puppt-owned Go code and does not shell out to Office,
  LibreOffice, Keynote, browser renderers, SaaS renderers, or image-conversion
  tools.
- CI runs `make verify` on `main` and pull requests.
- Release packaging uses GoReleaser and publishes platform archives plus
  checksums from version tags.

## Known Gaps

- Notes updates require an existing notes relationship; creating new notes parts
  for slides without notes is not implemented.
- General-purpose expected-content assertions are not yet part of
  `puppt validate`.
- Advanced non-text extraction and richer media metadata remain limited.
- Real-world deck fixture breadth remains limited.
- Renderer parity is incomplete. See `docs/RENDERING.md`,
  `docs/SUPPORT_MATRIX.md`, and `docs/RENDERER_COMPLETION_CHECKLIST.md` for
  the current renderer boundary and evidence.

## Verification

```sh
make verify
```

This runs the test suite, whitespace diff check, and CLI help smoke test.
