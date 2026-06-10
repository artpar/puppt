# Acceptance Workflow

The v1 acceptance path is covered by CLI tests and can be run manually with the same commands.

```sh
puppt inspect input.pptx --json
puppt edit input.pptx --edit edit.json --out edited.pptx --json
puppt validate edited.pptx --json
puppt review edited.pptx --changes changes.json --json
```

The acceptance workflow proves:

- Inspection sees slide order and text in the input deck.
- A targeted edit writes a new deck.
- Validation reports the edited deck as structurally valid.
- Review reports touched slides/objects, changes, skipped/ambiguous/unsupported counts, and validation status.

Run the automated acceptance coverage with:

```sh
go test ./internal/cli
```
