# Decision 0002: Puppt Owns the PPTX Reader and Writer

## Status

Accepted.

## Context

Puppt's core promise is precise, preservation-first `.pptx` inspection and mutation. Third-party libraries are useful for non-core infrastructure, but the authoritative package reader/writer controls the product's main value: inspect before acting, surgical edits, preservation of unsupported content, explicit warnings, and validation.

PowerPoint `.pptx` files are Office Open XML packages governed by ECMA-376 and ISO/IEC 29500. PresentationML documents are packages of related parts, not a single XML body, and each slide has its own part connected through relationships.

## Decision

Implement the authoritative `.pptx` package reader/writer in Puppt-owned Go code.

The core implementation will use:

- `archive/zip` for package container access.
- `encoding/xml` for controlled XML parsing.
- `path` for package-relative target resolution.
- Puppt-owned structs for content types, relationships, presentation roots, slide order, validation, changes, and warnings.

Third-party `.pptx` libraries MAY be studied as references and MAY be used later for optional comparison tests, but they MUST NOT own the v1 authoritative read/write/mutation path.

## Reference Base

- ECMA-376: https://ecma-international.org/publications-and-standards/standards/ecma-376/
- ISO/IEC 29500-1: https://www.iso.org/standard/71691.html
- Microsoft PresentationML structure: https://learn.microsoft.com/en-us/office/open-xml/presentation/structure-of-a-presentationml-document
- Microsoft Office ISO/IEC 29500 implementation notes: https://learn.microsoft.com/en-us/openspecs/office_standards/ms-oi29500/bd9e8289-844a-42e2-9809-66c7005bd9e2

## Consequences

- Puppt can preserve unknown and unsupported package parts intentionally.
- Mutation behavior can be traced to package parts and relationships.
- Tests can assert exact preservation and structural invariants.
- Implementation cost is higher than delegating to a general `.pptx` library.
- Puppt must maintain its own compatibility fixture suite.

## Validation

Checkpoint 1 begins with package-level tests:

- reject unsupported extension
- reject invalid ZIP
- require `[Content_Types].xml`
- require root relationships
- resolve presentation part
- parse presentation relationships
- expose slide part order from presentation slide IDs
