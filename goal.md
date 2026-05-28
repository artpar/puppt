/goal Complete Puppt as a production-grade v1 agent-first PowerPoint inspection, editing, creation, validation, and review tool, using PRODUCT_VISION.md, USER_EXPERIENCE.md, swe_skill.md, and project-ops.md as binding source documents.

These documents are binding project law. Treat their requirements as mandatory, not inspirational. When implementation choices conflict, the controlling order is:

1. `goal.md` for completion scope and acceptance.
2. `PRODUCT_VISION.md` for product identity and boundaries.
3. `USER_EXPERIENCE.md` for agent and human workflow behavior.
4. `project-ops.md` for checkpoint execution, module plan, gates, and evidence.
5. `swe_skill.md` for engineering doctrine, safety, verification, and operating discipline.

The implementation language for Puppt v1 is **Go**. Do not implement the product core in another language unless this document is explicitly updated. Small helper scripts may be used only when they do not become product runtime dependencies and do not weaken the Go-first architecture.

Stop only when Puppt can reliably:
1. Inspect modern .pptx files and return structured facts about slides, titles, visible text, speaker notes, images/media references, layouts, repeated content, and warnings.
2. Apply targeted edits without disturbing unrelated deck content, including text replacement, deck-wide text replacement, slide add/delete/move/duplicate, speaker-note updates, image replacement, metadata updates, and simple text/shape additions where safely supported.
3. Create a clean editable .pptx from structured instructions.
4. Validate output files for structural usability and expected content.
5. Produce a structured and human-readable change summary naming touched slides, changed objects/content, skipped edits, ambiguous matches, unsupported features, and validation status.
6. Preserve unsupported or untargeted content wherever possible, and fail explicitly instead of silently flattening, dropping, or corrupting deck content.

Operate in checkpoints. At each checkpoint:
- Read the relevant docs and current code before editing.
- Make the smallest coherent implementation step.
- Add or update focused tests and fixtures.
- Run the narrowest meaningful validation commands.
- Record progress, changed files, verification results, remaining risks, and next checkpoint.

Use the doctrine from swe_skill.md and project-ops.md as the quality bar:
- correctness before speed
- inspect before acting
- surgical edits over regeneration
- editable PowerPoint output only
- explicit errors and warnings
- stable interfaces
- bounded dependencies
- traceable changes
- no secrets in code/logs
- deterministic tests where feasible
- documentation required for operation

Do not stop for ordinary partial completion. Stop only when the v1 acceptance suite passes, docs describe supported and unsupported behavior, sample workflows are demonstrably complete, and remaining limitations are explicitly listed as known non-v1 gaps.

## Project Operation

Run the project as sequential Codex goal checkpoints, not one vague “make it perfect” task.

1. **Discovery and Architecture**
   - Inspect repo shape, dependency candidates, current implementation if any, and `.pptx` library constraints.
   - Produce or update an implementation plan for core modules: deck parser, object model, edit planner, mutation engine, validator, reporter, CLI/API surface, fixtures.
   - Decide supported v1 `.pptx` features and unsupported-preserve behavior.

2. **Inspection Core**
   - Build structured deck inspection first.
   - Output stable slide/object identifiers, text runs, images, notes, metadata, warnings.
   - Add fixture decks and golden inspection tests.

3. **Targeting and Edit Planner**
   - Implement targeting by slide number, title, visible text, object id, match scope, and deck property.
   - Detect ambiguity before mutation.
   - Return planned edits and skipped/ambiguous matches in structured form.

4. **Mutation Workflows**
   - Implement common edits in priority order: text replace, notes update, metadata update, slide operations, image replacement, simple additions.
   - Preserve styling and unrelated XML/package parts where possible.
   - Add round-trip tests proving unaffected content remains intact.

5. **Creation Workflow**
   - Create editable decks from structured instructions: title slides, section slides, bullet/body content, notes, images when provided.
   - Keep design simple, editable, and deterministic.

6. **Validation and Review**
   - Validate generated/edited `.pptx` structure.
   - Verify expected content exists after edits.
   - Produce machine-readable JSON plus concise human summaries.

7. **Operational Hardening**
   - Add docs for CLI/API usage, supported operations, unsupported boundaries, failure modes, and examples.
   - Add regression fixtures for real-world deck patterns.
   - Ensure tests run from a clean checkout with documented commands.

## Acceptance Criteria

Puppt v1 is complete when Codex can demonstrate these workflows end to end on sample `.pptx` fixtures:

- Inspect a deck and identify slide order, titles, visible text, notes, images, metadata, and warnings.
- Replace one title on one slide without changing unrelated slides.
- Replace a phrase across the deck and report exact match count.
- Add, delete, move, and duplicate slides while preserving editability.
- Add or update speaker notes.
- Replace an image or explicitly report why the target is ambiguous/unsupported.
- Create a new editable deck from structured input.
- Validate the resulting file and report success/warnings/failures.
- Return a clear change summary suitable for an AI agent and a human reviewer.

## Test Plan

- Unit tests for parsing, targeting, ambiguity detection, edit planning, validation, and report generation.
- Round-trip fixture tests for existing decks, including notes, images, metadata, and repeated text.
- Negative tests for unsupported file types, missing slide numbers, no-match text, ambiguous matches, invalid input, and corrupted output.
- Contract tests for CLI/API output shape so agents can rely on stable fields.
- Manual verification command that opens or validates generated `.pptx` files with available local tooling.

## Assumptions

- First completion target is **planning-only now**, but the eventual implementation target is **production-grade v1**.
- v1 focuses on modern `.pptx`; legacy binary PowerPoint formats are out of scope.
- Preview is useful but not a blocker for v1 unless the repo already contains preview infrastructure.
- Human UI is not required for the first implementation goal; the primary surface is agent-friendly CLI/API behavior.
- “Perfection” means doctrine-compliant, well-tested, predictable, editable, and honest about unsupported cases, not unlimited feature coverage.
- Go is the required implementation language for the product core, CLI, public API surface, test suite, and package-level fixtures.
