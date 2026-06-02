# ECMA-376 Reference Bundle

This directory keeps local copies of the Office Open XML specification material
that renderer and package work must cite before changing behavior.

Source:

- ECMA-376, 5th edition, December 2016
- https://ecma-international.org/publications-and-standards/standards/ecma-376/
- Downloaded archive: `ECMA-376-1_5th_edition_december_2016.zip`

Stored files:

- `part1/Ecma Office Open XML Part 1 - Fundamentals And Markup Language Reference.pdf`
- `part1/schema/strict/*.xsd`
- `SHA256SUMS`

Important local anchors for current renderer work:

- DrawingML text anchoring enum:
  `part1/schema/strict/dml-main.xsd`, lines 2547-2555.
- DrawingML text body properties:
  `part1/schema/strict/dml-main.xsd`, lines 2625-2652.
- DrawingML text body content model:
  `part1/schema/strict/dml-main.xsd`, lines 2653-2659.
- DrawingML picture content model:
  `part1/schema/strict/dml-picture.xsd`, lines 14-21.
- DrawingML relative crop/fill rectangles:
  `part1/schema/strict/dml-main.xsd`, lines 648-652.
- DrawingML stretch/fill mode and blip fill:
  `part1/schema/strict/dml-main.xsd`, lines 1455-1464 and 1502-1509.
- DrawingML color choice and fill properties:
  `part1/schema/strict/dml-main.xsd`, lines 667-680 and 1577-1590.
- DrawingML text autofit:
  `part1/schema/strict/dml-main.xsd`, lines 2610-2624.

Maintenance rules:

- Do not edit vendored spec files.
- If the ECMA source archive is refreshed, replace the affected files together
  and update `SHA256SUMS`.
- Cite this directory, section numbers from the PDF, or exact schema paths and
  lines in renderer experiment logs before implementing behavior that depends
  on OOXML semantics.
- Use Office implementation notes only as compatibility evidence after the ECMA
  schema/reference has been checked.
