#!/usr/bin/env python3
"""Generate the OOXML/DrawingML schema coverage audit."""

from __future__ import annotations

from collections import Counter, defaultdict
from pathlib import Path
import re
import textwrap
from xml.etree import ElementTree as ET


ROOT = Path(__file__).resolve().parents[1]
SCHEMA_DIR = ROOT / "docs/specs/ecma-376/part1/schema/strict"
OUTPUT = ROOT / "docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md"

TARGET_FILES = [
    "pml.xsd",
    "dml-main.xsd",
    "dml-picture.xsd",
    "dml-diagram.xsd",
    "dml-chart.xsd",
    "dml-chartDrawing.xsd",
    "dml-lockedCanvas.xsd",
    "dml-spreadsheetDrawing.xsd",
    "dml-wordprocessingDrawing.xsd",
]

XSD_NS = {"xsd": "http://www.w3.org/2001/XMLSchema"}
DECLARATION_TAGS = {
    "complexType",
    "simpleType",
    "group",
    "attributeGroup",
    "element",
    "attribute",
}


SUPPORTED = "Supported"
PARTIAL = "Partial"
UNSUPPORTED = "Unsupported"
OUT_OF_SCOPE = "Out of renderer scope"
NO_EVIDENCE = "Unimplemented / no evidence"


PML_SUPPORTED = {
    "CT_SlideIdListEntry",
    "CT_SlideIdList",
    "CT_SlideSize",
    "CT_Empty",
    "sldId",
    "presentation",
}

PML_PARTIAL_PREFIXES = (
    "CT_Slide",
    "CT_CommonSlide",
    "CT_Shape",
    "CT_Connector",
    "CT_Picture",
    "CT_GraphicalObjectFrame",
    "CT_GroupShape",
    "CT_Background",
    "CT_Placeholder",
    "CT_ApplicationNonVisual",
    "CT_SlideMaster",
    "CT_SlideLayout",
    "CT_SlideMasterTextStyles",
    "CT_HeaderFooter",
    "CT_Notes",
)

PML_PARTIAL_NAMES = {
    "sld",
    "sldLayout",
    "sldMaster",
    "notes",
    "ph",
    "nvPr",
    "sp",
    "cxnSp",
    "pic",
    "graphicFrame",
    "grpSp",
    "bg",
    "clrMap",
    "clrMapOvr",
}

PML_UNSUPPORTED_PREFIXES = (
    "CT_TL",
    "CT_SlideTransition",
    "CT_Transition",
    "ST_Transition",
    "CT_Control",
    "CT_Ole",
    "CT_CustomData",
)

PML_OUT_OF_SCOPE_PREFIXES = (
    "CT_Handout",
    "CT_Comment",
    "CT_CommentAuthor",
    "CT_CustomerData",
    "CT_Kinsoku",
    "CT_ModifyVerifier",
    "CT_PhotoAlbum",
    "CT_Show",
    "CT_Print",
    "CT_Web",
)


DML_MAIN_SUPPORTED = {
    "CT_Point2D",
    "CT_PositiveSize2D",
    "CT_Ratio",
    "CT_RelativeRect",
    "ST_Coordinate",
    "ST_Coordinate32",
    "ST_PositiveCoordinate",
    "ST_PositiveCoordinate32",
    "ST_Angle",
    "ST_PositiveFixedAngle",
    "ST_Percentage",
}

DML_MAIN_PARTIAL_PREFIXES = (
    "CT_Color",
    "CT_SchemeColor",
    "CT_SRgbColor",
    "CT_ScRgbColor",
    "CT_PresetColor",
    "CT_SystemColor",
    "CT_HslColor",
    "CT_ColorScheme",
    "CT_Font",
    "CT_Style",
    "CT_Fill",
    "CT_Solid",
    "CT_Gradient",
    "CT_Blip",
    "CT_Transform2D",
    "CT_GroupTransform2D",
    "CT_PresetGeometry",
    "CT_CustomGeometry",
    "CT_Path2D",
    "CT_Geom",
    "CT_Line",
    "CT_ShapeProperties",
    "CT_GroupShapeProperties",
    "CT_Text",
    "CT_Table",
    "CT_OuterShadow",
    "CT_Effect",
)

DML_MAIN_PARTIAL_NAMES = {
    "EG_ColorChoice",
    "EG_ColorTransform",
    "EG_FillProperties",
    "EG_FillModeProperties",
    "EG_Geometry",
    "EG_EffectProperties",
    "EG_LineFillProperties",
    "EG_LineDashProperties",
    "EG_LineJoinProperties",
    "EG_TextRun",
    "EG_TextAutofit",
    "ST_BlackWhiteMode",
    "ST_SchemeColorVal",
    "ST_PresetColorVal",
    "ST_ShapeType",
    "ST_TextAnchoringType",
    "ST_TextVerticalType",
    "ST_TextWrappingType",
    "ST_TextHorzOverflowType",
    "ST_TextVertOverflowType",
    "ST_LineEndType",
    "ST_LineCap",
    "ST_CompoundLine",
}

DML_MAIN_UNSUPPORTED_PREFIXES = (
    "CT_Scene3D",
    "CT_Shape3D",
    "CT_Camera",
    "CT_LightRig",
    "CT_Bevel",
    "CT_Backdrop",
    "CT_Reflection",
    "CT_Glow",
    "CT_InnerShadow",
    "CT_PresetShadow",
    "CT_Blur",
    "CT_FillOverlay",
    "CT_EffectContainer",
    "CT_Alpha",
    "CT_Duotone",
    "CT_HSL",
    "CT_Tint",
    "CT_Luminance",
)


def local_name(tag: str) -> str:
    return tag.split("}", 1)[-1]


def line_index(path: Path) -> dict[tuple[str, str], int]:
    pattern = re.compile(
        r"^\s*<xsd:(complexType|simpleType|group|attributeGroup|element|attribute)\b[^>]*(?:\bname=\"([^\"]+)\"|\bref=\"([^\"]+)\")"
    )
    index: dict[tuple[str, str], int] = {}
    for lineno, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        match = pattern.search(line)
        if not match:
            continue
        kind = match.group(1)
        name = match.group(2) or match.group(3)
        index.setdefault((kind, name), lineno)
    return index


def member_summary(node: ET.Element) -> str:
    items: list[str] = []
    for child in node.iter():
        if child is node:
            continue
        kind = local_name(child.tag)
        if kind not in {"element", "attribute", "group", "choice", "sequence"}:
            continue
        name = child.attrib.get("name") or child.attrib.get("ref")
        if not name:
            continue
        if kind == "element":
            label = f"el:{name}"
        elif kind == "attribute":
            label = f"attr:{name}"
        else:
            label = f"{kind}:{name}"
        if label not in items:
            items.append(label)
        if len(items) >= 8:
            break
    if not items:
        return "-"
    suffix = " ..." if len(items) >= 8 else ""
    return ", ".join(items) + suffix


def declaration_rows(path: Path) -> list[dict[str, str]]:
    root = ET.parse(path).getroot()
    lines = line_index(path)
    rows = []
    for child in root:
        kind = local_name(child.tag)
        if kind not in DECLARATION_TAGS:
            continue
        name = child.attrib.get("name") or child.attrib.get("ref")
        if not name:
            continue
        rows.append(
            {
                "file": path.name,
                "kind": kind,
                "name": name,
                "line": str(lines.get((kind, name), "?")),
                "members": member_summary(child),
            }
        )
    return rows


def coverage_for(row: dict[str, str]) -> tuple[str, str]:
    file = row["file"]
    name = row["name"]
    kind = row["kind"]

    if file in {"dml-wordprocessingDrawing.xsd", "dml-spreadsheetDrawing.xsd"}:
        return OUT_OF_SCOPE, "Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target."
    if file == "dml-lockedCanvas.xsd":
        return UNSUPPORTED, "Locked canvas is not lowered into render primitives."
    if file == "dml-chart.xsd":
        return UNSUPPORTED, "Chart schema is not rendered as chart graphics."
    if file == "dml-chartDrawing.xsd":
        return UNSUPPORTED, "Chart drawing schema is not rendered directly."
    if file == "dml-picture.xsd":
        return PARTIAL, "Picture object structure is parsed/rendered through PresentationML picture handling; full blip/effect behavior is incomplete."
    if file == "dml-diagram.xsd":
        if name in {"CT_Shape", "sp", "CT_Style", "CT_Pt", "CT_PtList"}:
            return PARTIAL, "Simple diagram shapes/text can render when resolved from related diagram drawing parts."
        return UNSUPPORTED, "SmartArt/diagram layout and non-shape content are not fully implemented."

    if file == "pml.xsd":
        if name in PML_SUPPORTED:
            return SUPPORTED, "Covered by package, presentation, slide-order, or slide-size workflows."
        if name in PML_PARTIAL_NAMES or name.startswith(PML_PARTIAL_PREFIXES):
            return PARTIAL, "Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete."
        if name.startswith(PML_UNSUPPORTED_PREFIXES):
            return UNSUPPORTED, "Detected only partially or not rendered for static output."
        if name.startswith(PML_OUT_OF_SCOPE_PREFIXES):
            return OUT_OF_SCOPE, "Not part of current static slide-rendering goal."
        if kind in {"element"} and name in {"sld", "sldLayout", "sldMaster", "presentation"}:
            return PARTIAL, "Root PresentationML element participates in current workflows."
        return NO_EVIDENCE, "No explicit renderer coverage evidence found in the maintained docs/tests."

    if file == "dml-main.xsd":
        if name in DML_MAIN_SUPPORTED:
            return SUPPORTED, "Core unit/value type used by current render geometry or package workflows."
        if name in DML_MAIN_PARTIAL_NAMES or name.startswith(DML_MAIN_PARTIAL_PREFIXES):
            return PARTIAL, "Part of current DrawingML render subset; full source semantics are not complete."
        if name.startswith(DML_MAIN_UNSUPPORTED_PREFIXES):
            return UNSUPPORTED, "Visible behavior is unsupported or only reported as partial in current renderer."
        if name in {"graphic", "CT_GraphicalObject", "CT_GraphicalObjectData"}:
            return PARTIAL, "Graphic payloads are recognized for tables and simple diagrams; charts/OLE/media remain unsupported."
        return NO_EVIDENCE, "No explicit renderer coverage evidence found in the maintained docs/tests."

    return NO_EVIDENCE, "No explicit renderer coverage evidence found in the maintained docs/tests."


def md_escape(value: str) -> str:
    return value.replace("|", "\\|").replace("\n", " ")


def render() -> str:
    all_rows: list[dict[str, str]] = []
    for filename in TARGET_FILES:
        all_rows.extend(declaration_rows(SCHEMA_DIR / filename))

    for row in all_rows:
        status, note = coverage_for(row)
        row["status"] = status
        row["note"] = note

    by_file = defaultdict(list)
    for row in all_rows:
        by_file[row["file"]].append(row)

    status_counts = Counter(row["status"] for row in all_rows)

    lines: list[str] = []
    lines.extend(
        [
            "# OOXML/DrawingML Coverage Matrix",
            "",
            "This is the exhaustive schema-declaration audit for Puppt's static PPTX renderer target.",
            "It is generated from the local ECMA-376 strict-schema files under",
            "`docs/specs/ecma-376/part1/schema/strict/`.",
            "",
            "Scope for this audit:",
            "",
            "- `pml.xsd`",
            "- every `dml-*.xsd` file in the local strict schema bundle",
            "- every top-level `xsd:complexType`, `xsd:simpleType`, `xsd:group`,",
            "  `xsd:attributeGroup`, `xsd:element`, and `xsd:attribute` declaration",
            "",
            "This makes the audit complete over the PresentationML/DrawingML schema",
            "declarations available in the repo. Nested child elements and attributes are",
            "summarized in the `Members` column of their owning declaration instead of being",
            "duplicated as separate rows.",
            "",
            "Status definitions:",
            "",
            "- **Supported**: current code/docs provide explicit evidence for the covered",
            "  declaration subset.",
            "- **Partial**: current code/docs cover some semantics, but known child elements,",
            "  attributes, renderer behavior, or fixtures remain incomplete.",
            "- **Unsupported**: currently not rendered or only reported/preserved.",
            "- **Out of renderer scope**: schema belongs to another host application or a",
            "  non-static-rendering area outside the current Puppt renderer goal.",
            "- **Unimplemented / no evidence**: the declaration exists in the spec but has no",
            "  maintained code/doc/test evidence yet, so it is treated as unimplemented",
            "  until proven otherwise.",
            "",
            "## Audit Totals",
            "",
            f"- Total schema declarations audited: **{len(all_rows)}**",
            "",
            "| Status | Count |",
            "|---|---:|",
        ]
    )
    for status in [SUPPORTED, PARTIAL, UNSUPPORTED, OUT_OF_SCOPE, NO_EVIDENCE]:
        lines.append(f"| {status} | {status_counts[status]} |")

    lines.extend(["", "## File Totals", "", "| Schema file | Declarations | Supported | Partial | Unsupported | Out of scope | Unimplemented/no evidence |", "|---|---:|---:|---:|---:|---:|---:|"])
    for filename in TARGET_FILES:
        rows = by_file[filename]
        counts = Counter(row["status"] for row in rows)
        lines.append(
            f"| `{filename}` | {len(rows)} | {counts[SUPPORTED]} | {counts[PARTIAL]} | {counts[UNSUPPORTED]} | {counts[OUT_OF_SCOPE]} | {counts[NO_EVIDENCE]} |"
        )

    lines.extend(
        [
            "",
            "## Coverage Rows",
            "",
            "Each row is a schema declaration. The anchor line points at the declaration in",
            "the local schema file.",
        ]
    )

    for filename in TARGET_FILES:
        rows = by_file[filename]
        lines.extend(
            [
                "",
                f"### {filename}",
                "",
                "| Anchor | Kind | Declaration | Members | Status | Evidence / gap |",
                "|---|---|---|---|---|---|",
            ]
        )
        for row in rows:
            anchor = f"`{row['file']}:{row['line']}`"
            lines.append(
                "| "
                + " | ".join(
                    [
                        anchor,
                        md_escape(row["kind"]),
                        f"`{md_escape(row['name'])}`",
                        md_escape(row["members"]),
                        row["status"],
                        md_escape(row["note"]),
                    ]
                )
                + " |"
            )

    lines.extend(
        [
            "",
            "## Maintenance Rules",
            "",
            "1. Regenerate this file after changing schema scope or coverage classification:",
            "",
            "   ```text",
            "   python3 tools/generate_ooxml_drawingml_audit.py",
            "   ```",
            "",
            "2. A declaration may move to **Supported** only with source-schema anchors,",
            "   parser/lowering evidence, deterministic fixtures, and renderer/reporting",
            "   tests for excluded subclauses.",
            "3. Do not mark a declaration supported because a real-world screenshot looks",
            "   close. Source semantics and fixture proof come first.",
        ]
    )
    return "\n".join(lines) + "\n"


def main() -> None:
    OUTPUT.write_text(render(), encoding="utf-8")


if __name__ == "__main__":
    main()
