# Puppt Reference Map

Puppt's `.pptx` reader/writer is product-core code. It MUST be implemented against public Office Open XML and PresentationML references, plus real fixture behavior, rather than undocumented assumptions.

## Normative and Primary References

### ECMA-376: Office Open XML File Formats

Source: https://ecma-international.org/publications-and-standards/standards/ecma-376/

Local reference bundle: `docs/specs/ecma-376/`

Use:

- Office Open XML vocabulary and document representation.
- Package structure and producer/consumer requirements.
- Part 1: Fundamentals and Markup Language Reference.
- Part 2: Open Packaging Conventions.
- Part 3: Markup Compatibility and Extensibility.
- Part 4: Transitional Migration Features.

Puppt interpretation:

- Use ECMA-376 as the primary freely accessible standard reference.
- Treat Open Packaging Conventions as foundational for ZIP parts, relationships, content types, and target resolution.

### ISO/IEC 29500: Office Open XML File Formats

Source: https://www.iso.org/standard/71691.html

Use:

- XML vocabularies for word-processing documents, spreadsheets, and presentations.
- Requirements for Office Open XML consumers and producers.
- Strict and transitional conformance concepts.

Puppt interpretation:

- Use ISO/IEC 29500 terminology and conformance concepts when defining validation, compatibility, and support boundaries.
- Prefer explicit warnings where real-world transitional content exceeds current v1 support.

### Microsoft Learn: Structure of a PresentationML Document

Source: https://learn.microsoft.com/en-us/office/open-xml/presentation/structure-of-a-presentationml-document

Use:

- Practical PresentationML package structure.
- Important presentation parts.
- Slide, layout, master, theme, notes, comments, media, and relationship behavior.
- Minimum presentation package examples.

Puppt interpretation:

- A presentation is not one large XML document; it is a package of related parts.
- Each slide has its own part.
- Slide order comes from the presentation part and relationship IDs.
- Slide parts are explicit relationship targets from the presentation part.

### Microsoft Learn: Office Implementation Information for ISO/IEC 29500

Source: https://learn.microsoft.com/en-us/openspecs/office_standards/ms-oi29500/bd9e8289-844a-42e2-9809-66c7005bd9e2

Use:

- Microsoft Office implementation notes for ISO/IEC 29500.
- Compatibility details where Office behavior varies from or extends the standard.

Puppt interpretation:

- Use this as a compatibility reference when PowerPoint behavior matters more than a minimal standard reading.
- Do not assume standards-only output opens correctly in PowerPoint without fixture tests.

### Microsoft Learn: Office Drawing Extensions to Office Open XML

Source: https://learn.microsoft.com/en-us/openspecs/office_standards/ms-odrawxml/

Local notes: `docs/specs/ms-odrawxml/`

Use:

- Microsoft `a14` and related Office Drawing extension elements that appear in
  real `.pptx` source XML.
- Compatibility interpretation after the ECMA-376 DrawingML model has already
  been checked.

Puppt interpretation:

- Preserve unknown or unsupported extension XML where possible.
- Do not infer visible renderer behavior from an extension element unless the
  official extension documentation and an attributed object fixture support it.

## Renderer Primitive Dependency References

These are not `.pptx` authorities. They are candidate primitive libraries for
graphics/text/color work behind Puppt-owned DrawingML interpretation.

### go-text/typesetting

Source:

- https://pkg.go.dev/github.com/go-text/typesetting/harfbuzz
- https://pkg.go.dev/github.com/go-text/typesetting/shaping

Use:

- Candidate pure-Go text shaping and OpenType positioning backend.

Puppt interpretation:

- May be evaluated behind a text-shaping adapter that consumes Puppt-resolved
  text runs and font files.
- Must not read, write, render, mutate, validate, or interpret `.pptx` packages.
- Added as a controlled primitive dependency on 2026-06-01 for the test-only
  text shaping diagnostic; production text rendering is not replaced yet.

### draw2d

Source: https://pkg.go.dev/github.com/llgcode/draw2d

Use:

- Candidate pure-Go vector path/stroke/rasterization backend under the current
  Go 1.24 toolchain.

Puppt interpretation:

- May be evaluated behind a shape-rasterization adapter that consumes
  Puppt-parsed DrawingML geometry.
- Must first prove focused object fixtures before replacing production shape
  painting.
- Added as a controlled primitive dependency on 2026-06-01 for the
  test-only rectangle backend diagnostic; production shape rendering is not
  replaced yet.

### tdewolff/canvas

Source: https://pkg.go.dev/github.com/tdewolff/canvas

Use:

- Candidate future vector/text/raster backend.

Puppt interpretation:

- Not an immediate dependency while Puppt remains on Go 1.24.3 because the
  current latest module advertises Go 1.25.0.

## Implementation Rule

Every `.pptx` parser, writer, mutator, and validator change SHOULD cite the relevant section or concept from these references in its decision record, code comment, fixture name, or test name when the behavior is not obvious.

When a reference and real PowerPoint behavior disagree, preserve the input, report the discrepancy, add a fixture, and document the chosen behavior before broadening mutation support.
