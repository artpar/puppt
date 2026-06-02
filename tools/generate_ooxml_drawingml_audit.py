#!/usr/bin/env python3
"""Generate the OOXML/DrawingML schema coverage audit."""

from __future__ import annotations

import argparse
from collections import Counter, defaultdict
import json
from pathlib import Path
import re
import textwrap
from xml.etree import ElementTree as ET


ROOT = Path(__file__).resolve().parents[1]
SCHEMA_DIR = ROOT / "docs/specs/ecma-376/part1/schema/strict"
OUTPUT = ROOT / "docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md"
SUMMARY_OUTPUT = ROOT / "docs/renderer-coverage-summary.json"

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

QUEUE_CORE_STATIC = "core-static"
QUEUE_COMMON_PARTIAL = "common-partial"
QUEUE_HARD_RENDERING = "hard-rendering"
QUEUE_UNSUPPORTED_PRESERVE = "unsupported-preserve"
QUEUE_OUT_OF_SCOPE = "out-of-scope"

QUEUES = [
    QUEUE_CORE_STATIC,
    QUEUE_COMMON_PARTIAL,
    QUEUE_HARD_RENDERING,
    QUEUE_UNSUPPORTED_PRESERVE,
    QUEUE_OUT_OF_SCOPE,
]


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

PML_TRANSITION_PLAYBACK_NAMES = {
    "CT_SideDirectionTransition",
    "CT_CornerDirectionTransition",
    "CT_EightDirectionTransition",
    "CT_OrientationTransition",
    "CT_InOutTransition",
    "CT_OptionalBlackTransition",
    "CT_SplitTransition",
    "CT_WheelTransition",
    "CT_TransitionStartSoundAction",
    "CT_TransitionSoundAction",
}

PML_TIMING_PARTIAL_PREFIXES = (
    "CT_TL",
    "ST_TL",
    "AG_TL",
)

PML_TIMING_PARTIAL_NAMES = {
    "CT_TimeNodeList",
    "ST_IterateType",
    "CT_BuildList",
    "ST_ChartBuildStep",
    "ST_DgmBuildStep",
    "ST_AnimationBuildType",
    "ST_AnimationDgmOnlyBuildType",
    "ST_AnimationDgmBuildType",
    "ST_AnimationChartOnlyBuildType",
    "ST_AnimationChartBuildType",
    "CT_AnimationDgmElement",
    "CT_AnimationChartElement",
    "CT_AnimationElementChoice",
    "CT_AnimationDgmBuildProperties",
    "CT_AnimationChartBuildProperties",
    "CT_AnimationGraphicalObjectBuildProperties",
}

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

PML_OUT_OF_SCOPE_NAMES = {
    "ST_Name",
    "ST_Direction",
    "ST_Index",
    "CT_IndexRange",
    "CT_CustomShowId",
    "EG_SlideListChoice",
    "CT_TagsData",
    "cmAuthorLst",
    "cmLst",
    "CT_SmartTags",
    "CT_CustomShow",
    "CT_CustomShowList",
    "CT_HtmlPublishProperties",
    "ST_PrintWhat",
    "ST_PrintColorMode",
    "EG_ShowType",
    "CT_PresentationProperties",
    "presentationPr",
    "ST_PhotoAlbumLayout",
    "ST_PhotoAlbumFrameShape",
    "handoutMaster",
    "notesMaster",
    "sldSyncPr",
    "ST_SplitterBarState",
    "ST_ViewType",
    "CT_NormalViewPortion",
    "CT_NormalViewProperties",
    "CT_CommonViewProperties",
    "CT_OutlineViewSlideEntry",
    "CT_OutlineViewSlideList",
    "CT_OutlineViewProperties",
    "CT_Guide",
    "CT_GuideList",
    "CT_ViewProperties",
    "viewPr",
    "CT_StringTag",
    "CT_TagList",
    "tagLst",
}

PML_EMBEDDED_FONT_PARTIAL_NAMES = {
    "CT_EmbeddedFontDataId",
    "CT_EmbeddedFontListEntry",
    "CT_EmbeddedFontList",
}

PML_EXTENSION_PARTIAL_NAMES = {
    "CT_Extension",
    "EG_ExtensionList",
    "CT_ExtensionList",
    "CT_ExtensionListModify",
}

PML_PRESENTATION_PARTIAL_NAMES = {
    "ST_SlideId",
    "ST_SlideMasterId",
    "ST_SlideSizeCoordinate",
    "ST_SlideSizeType",
    "ST_BookmarkIdSeed",
    "CT_Presentation",
    "ST_PlaceholderType",
    "ST_PlaceholderSize",
    "EG_TopLevelSlide",
    "EG_ChildSlide",
    "AG_ChildSlide",
    "EG_Background",
    "ST_SlideLayoutType",
    "ST_SlideLayoutId",
}


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

DML_MAIN_MEDIA_UNSUPPORTED_NAMES = {
    "CT_AudioFile",
    "CT_VideoFile",
    "CT_QuickTimeFile",
    "CT_AudioCDTime",
    "CT_AudioCD",
    "CT_EmbeddedWAVAudioFile",
    "EG_Media",
    "videoFile",
}

DML_MAIN_ANIMATION_PARTIAL_NAMES = {
    "ST_ChartBuildStep",
    "ST_DgmBuildStep",
    "ST_AnimationBuildType",
    "ST_AnimationDgmOnlyBuildType",
    "ST_AnimationDgmBuildType",
    "ST_AnimationChartOnlyBuildType",
    "ST_AnimationChartBuildType",
    "CT_AnimationDgmElement",
    "CT_AnimationChartElement",
    "CT_AnimationElementChoice",
    "CT_AnimationDgmBuildProperties",
    "CT_AnimationChartBuildProperties",
    "CT_AnimationGraphicalObjectBuildProperties",
}

DML_MAIN_NONVISUAL_PARTIAL_NAMES = {
    "ST_DrawingElementId",
    "AG_Locking",
    "CT_ConnectorLocking",
    "CT_ShapeLocking",
    "CT_PictureLocking",
    "CT_GroupLocking",
    "CT_GraphicalObjectFrameLocking",
    "CT_ContentPartLocking",
    "CT_NonVisualDrawingProps",
    "CT_NonVisualDrawingShapeProps",
    "CT_NonVisualConnectorProperties",
    "CT_NonVisualPictureProperties",
    "CT_NonVisualGroupDrawingShapeProps",
    "CT_NonVisualGraphicFrameProperties",
    "CT_NonVisualContentPartProperties",
}

DML_MAIN_3D_UNSUPPORTED_NAMES = set()

DML_MAIN_3D_PARTIAL_NAMES = {
    "CT_Cell3D",
    "CT_Point3D",
    "CT_Vector3D",
    "CT_SphereCoords",
    "ST_PresetCameraType",
    "ST_FOVAngle",
    "CT_Camera",
    "ST_LightRigDirection",
    "ST_LightRigType",
    "CT_LightRig",
    "CT_Scene3D",
    "CT_Backdrop",
    "ST_BevelPresetType",
    "CT_Bevel",
    "ST_PresetMaterialType",
    "CT_Shape3D",
    "CT_FlatText",
    "EG_Text3D",
}

DML_MAIN_THEME_PARTIAL_NAMES = {
    "ST_StyleMatrixColumnIndex",
    "ST_FontCollectionIndex",
    "ST_ColorSchemeIndex",
    "CT_CustomColor",
    "CT_SupplementalFont",
    "CT_CustomColorList",
    "CT_BaseStyles",
    "CT_BaseStylesOverride",
    "CT_ClipboardStyleSheet",
    "CT_OfficeStyleSheet",
    "theme",
    "themeOverride",
    "themeManager",
    "CT_ObjectStyleDefaults",
    "CT_DefaultShapeDefinition",
    "CT_Headers",
    "EG_ThemeableFillStyle",
    "CT_ThemeableLineStyle",
    "EG_ThemeableEffectStyle",
    "EG_ThemeableFontStyles",
    "ST_OnOffStyleType",
    "tblStyleLst",
}

DML_MAIN_VALUE_PARTIAL_NAMES = {
    "ST_CoordinateUnqualified",
    "ST_Coordinate32Unqualified",
    "CT_Angle",
    "ST_FixedAngle",
    "CT_PositiveFixedAngle",
    "CT_Percentage",
    "ST_PositivePercentage",
    "CT_PositivePercentage",
    "ST_FixedPercentage",
    "CT_FixedPercentage",
    "ST_PositiveFixedPercentage",
    "CT_PositiveFixedPercentage",
    "ST_RectAlignment",
    "CT_Scale2D",
}

DML_MAIN_COLOR_TRANSFORM_PARTIAL_NAMES = {
    "CT_ComplementTransform",
    "CT_InverseTransform",
    "CT_GrayscaleTransform",
    "CT_GammaTransform",
    "CT_InverseGammaTransform",
    "ST_SystemColorVal",
}

DML_MAIN_EXTENSION_PARTIAL_NAMES = {
    "CT_OfficeArtExtension",
    "EG_OfficeArtExtensionList",
    "CT_OfficeArtExtensionList",
    "AG_Blob",
}

DML_MAIN_OUT_OF_SCOPE_NAMES = {
    "CT_Hyperlink",
}

DML_MAIN_GVML_PARTIAL_NAMES = {
    "CT_GvmlUseShapeRectangle",
    "CT_GvmlTextShape",
    "CT_GvmlShapeNonVisual",
    "CT_GvmlShape",
    "CT_GvmlConnectorNonVisual",
    "CT_GvmlConnector",
    "CT_GvmlPictureNonVisual",
    "CT_GvmlPicture",
    "CT_GvmlGraphicFrameNonVisual",
    "CT_GvmlGraphicalObjectFrame",
    "CT_GvmlGroupShapeNonVisual",
    "CT_GvmlGroupShape",
}

DML_MAIN_FILL_EFFECT_PARTIAL_NAMES = {
    "CT_BackgroundFormatting",
    "CT_WholeE2oFormatting",
    "CT_NoFillProperties",
    "ST_PathShadeType",
    "CT_PathShadeProperties",
    "EG_ShadeProperties",
    "ST_TileFlipMode",
    "ST_BlipCompression",
    "ST_PresetPatternVal",
    "blip",
}

DML_MAIN_EFFECT_UNSUPPORTED_NAMES = set()

DML_MAIN_GEOMETRY_PARTIAL_NAMES = {
    "ST_TextShapeType",
    "ST_GeomGuideName",
    "ST_GeomGuideFormula",
    "ST_AdjCoordinate",
    "ST_AdjAngle",
    "CT_AdjPoint2D",
    "CT_XYAdjustHandle",
    "CT_PolarAdjustHandle",
    "CT_ConnectionSite",
    "CT_AdjustHandleList",
    "CT_ConnectionSiteList",
    "CT_Connection",
    "ST_PathFillMode",
    "CT_PresetTextShape",
    "EG_TextGeometry",
    "ST_ShapeID",
    "CT_EmptyElement",
}

DML_MAIN_TEXT_PARTIAL_NAMES = {
    "ST_TextPoint",
    "ST_TextPointUnqualified",
    "ST_TextNonNegativePoint",
    "ST_PitchFamily",
    "EG_TextUnderlineLine",
    "EG_TextUnderlineFill",
    "ST_TextCapsType",
    "CT_Boolean",
    "ST_TextFontAlignType",
    "ST_TextIndentLevelType",
}


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
        return PARTIAL, "M12 lowers lockedCanvas graphicData children through the static GVML group parser for supported shapes and standalone text shapes; full locked canvas parity and unsupported child families remain partial."
    if file == "dml-chart.xsd":
        return PARTIAL, "M12 classifies chart payloads as preserved Partial rendering gaps: chart graphic frames, related chart parts, and chart skip records are detected, while static chart graphics remain implementation work."
    if file == "dml-chartDrawing.xsd":
        return PARTIAL, "M12 classifies chart drawing parts as preserved Partial rendering gaps: related user-shape drawing parts remain package-preserved while static chart drawing graphics remain implementation work."
    if file == "dml-picture.xsd":
        if name in {"CT_PictureNonVisual", "CT_Picture", "pic"}:
            return PARTIAL, "M07 render-scene lowering preserves picture provenance plus embedded/linked blip relationships, crop/fill/effect metadata, and current static picture backend semantics; full media and sampling parity remains incomplete."
        return PARTIAL, "Picture object structure is parsed/rendered through PresentationML picture handling; full static media parity remains incomplete."
    if file == "dml-diagram.xsd":
        if name in {"CT_Shape", "sp", "CT_Style", "CT_Pt", "CT_PtList"}:
            return PARTIAL, "M11 keeps the supported diagram subset to related diagram drawing parts that lower into static shape/text primitives; full SmartArt data/layout semantics remain static rendering implementation work."
        if name in {"CT_DataModel", "dataModel", "CT_RelIds", "relIds"}:
            return PARTIAL, "M11 resolves diagram data relIds to a drawing fallback when present and reports missing/unavailable SmartArt drawing fallbacks explicitly."
        return PARTIAL, "SmartArt/diagram layout, constraints, style, color transforms, and non-shape content are implementable static rendering work; current renderer preserves/reports missing drawing fallbacks and lowers only the supported drawing subset."

    if file == "pml.xsd":
        if name in PML_SUPPORTED:
            return SUPPORTED, "Covered by package, presentation, slide-order, or slide-size workflows."
        if name in PML_TRANSITION_PLAYBACK_NAMES or name.startswith("ST_Transition"):
            return OUT_OF_SCOPE, "Slide transition playback is outside the static slide-rendering target; transition XML is preserved by package workflows and is not applied to still PNG output."
        if name in PML_TIMING_PARTIAL_NAMES or name.startswith(PML_TIMING_PARTIAL_PREFIXES):
            return PARTIAL, "Static renderer handles source timing only for supported visibility entrance builds and reports other animation timing explicitly instead of silently applying slideshow behavior."
        if name in PML_EMBEDDED_FONT_PARTIAL_NAMES:
            return PARTIAL, "M12 reports embeddedFontLst declarations during render and preserves the package data; static output still relies on installed/fallback fonts rather than embedded font binaries."
        if name in PML_OUT_OF_SCOPE_NAMES:
            return OUT_OF_SCOPE, "Presentation playback, print/view, comments, custom shows, or authoring UI metadata are outside the static slide-rendering target."
        if name in PML_EXTENSION_PARTIAL_NAMES:
            return PARTIAL, "Extension-list metadata is preserved in source/package workflows, with known extension-derived render semantics promoted only when parsed and fixture-proven."
        if name in PML_PRESENTATION_PARTIAL_NAMES:
            return PARTIAL, "Presentation, slide-size, slide-id, placeholder, background, layout, master visibility, and color-map structures participate in current static rendering and package workflows; full PresentationML semantics remain incomplete."
        if name in {"CT_Shape", "sp"}:
            return PARTIAL, "M06 renders the common preset geometry subset plus custom move/line/quad/cubic/arc/multi-path geometry and source-derived stroke semantics; full preset catalog and text/image/effect parity remain incomplete."
        if name in {"CT_Connector", "cxnSp"}:
            return PARTIAL, "M06 renders straight connector geometry from source xfrm/flip endpoints with width, cap, dash, compound line, and schema line-end markers; routed connector semantics remain incomplete."
        if name in {"CT_Picture", "pic"}:
            return PARTIAL, "M05 preserves picture blip-fill relationship paint state alongside source-derived line/effect styles; full picture sampling/effect semantics remain incomplete."
        if name in {"CT_GraphicalObjectFrame", "graphicFrame"}:
            return PARTIAL, "M11 classifies graphic-frame payloads: tables and simple diagram drawing fallbacks can render, while charts and unknown graphicData are preserved and reported explicitly."
        if name in {"CT_OleObject", "CT_OleObjectEmbed", "CT_OleObjectLink", "oleObj", "AG_Ole", "ST_OleObjectFollowColorScheme"}:
            return UNSUPPORTED, "M11 detects OLE payloads, preserves embedded/link relationships and preview pictures where present, and reports the embedded application runtime as a source-proven static-renderer impossibility."
        if name in {"CT_Control", "CT_ControlList"}:
            return UNSUPPORTED, "M11 detects controls, preserves related payloads and preview pictures where present, and reports active control execution as a source-proven static-renderer impossibility."
        if name == "CT_Rel":
            return PARTIAL, "M11 parses contentPart relationship ids and reports preserved content-part payloads as unsupported render content."
        if name in {"CT_GroupShape", "grpSp"}:
            return PARTIAL, "M05 resolves group fill paint for child grpFill while M04 composes nested transforms; full group compositing remains incomplete."
        if name in {"CT_Background", "CT_BackgroundProperties", "bg"}:
            return PARTIAL, "M05 resolves bgPr/bgRef solid, gradient, and pattern fills through theme color maps and style matrix references; full background effects remain incomplete."
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
        if name in DML_MAIN_MEDIA_UNSUPPORTED_NAMES:
            return UNSUPPORTED, "M11 detects audio/video media relationships, preserves related parts, and reports time-based media playback as a source-proven static-renderer impossibility."
        if name in DML_MAIN_ANIMATION_PARTIAL_NAMES:
            return PARTIAL, "Animation build metadata participates in slideshow timing; current static renderer handles supported visibility entrance builds and reports other animation timing explicitly."
        if name in DML_MAIN_NONVISUAL_PARTIAL_NAMES:
            return PARTIAL, "Non-visual drawing properties, ids, names, creation ids, descriptions, titles, hidden/decorative flags, enabled lock flags, relationship-bearing metadata, and CT_NonVisualDrawingShapeProps@txBox text-box metadata are preserved in source parsing/lowering for current object primitives."
        if name in DML_MAIN_3D_UNSUPPORTED_NAMES:
            return UNSUPPORTED, "3-D camera, light rig, bevel, material, and text-depth semantics are detected or preserved and reported as unrendered 3-D behavior for static output."
        if name in DML_MAIN_3D_PARTIAL_NAMES:
            return PARTIAL, "M12 detects and reports source-authored DrawingML 3-D scene, camera, light-rig, bevel, shape-depth, text-depth, and table-cell 3-D properties; static 3-D surface rendering remains a feasible effect gap, not an Unsupported shortcut."
        if name in DML_MAIN_THEME_PARTIAL_NAMES:
            return PARTIAL, "Theme/style declarations participate in current theme color, fill, line, font, and default-style resolution; full Office theme/style semantics remain incomplete."
        if name in DML_MAIN_VALUE_PARTIAL_NAMES:
            return PARTIAL, "Core coordinate, angle, percentage, alignment, and scale value forms are consumed by current geometry, transform, fill, effect, and text helpers; full clause evidence remains partial."
        if name in DML_MAIN_COLOR_TRANSFORM_PARTIAL_NAMES:
            return PARTIAL, "M05 resolves common DrawingML color transforms including complement, inverse, grayscale, gamma, and system-color fallback in the shared color pipeline."
        if name in DML_MAIN_EXTENSION_PARTIAL_NAMES:
            return PARTIAL, "OfficeArt extension and blip relationship metadata is preserved by package/source workflows; extension-specific render semantics are parsed only when fixture-proven."
        if name in DML_MAIN_OUT_OF_SCOPE_NAMES:
            return OUT_OF_SCOPE, "Hyperlink interaction is outside the current static PresentationML slide-rendering path."
        if name in DML_MAIN_GVML_PARTIAL_NAMES:
            return PARTIAL, "M12 lowers GVML group children from lockedCanvas graphicData through existing static shape, text, picture, connector, and group primitives where supported; full GVML host-drawing parity remains partial."
        if name in DML_MAIN_FILL_EFFECT_PARTIAL_NAMES:
            return PARTIAL, "Current fill, background, blip, tile, pattern, path-gradient, and no-fill paths parse or render source-backed subsets while preserving/reporting incomplete clauses."
        if name in DML_MAIN_EFFECT_UNSUPPORTED_NAMES:
            return UNSUPPORTED, "This DrawingML effect or blend graph form is detected or preserved and reported as unrendered effect behavior for static output."
        if name in DML_MAIN_GEOMETRY_PARTIAL_NAMES:
            return PARTIAL, "Current geometry and text-geometry paths parse/render source-backed preset, custom, guide, handle, connection, and path-fill subsets; full preset/guide coverage remains incomplete."
        if name == "ST_TextFontAlignType":
            return PARTIAL, "M12 parses paragraph fontAlgn and applies supported top, center, bottom, baseline, and auto font metric alignment for horizontal styled text; full Office text parity remains incomplete."
        if name in DML_MAIN_TEXT_PARTIAL_NAMES:
            return PARTIAL, "M08 text parsing/layout consumes common point, pitch, underline, boolean, font-align, and indent-level value forms for supported horizontal text. M12 parses and renders ST_TextCapsType all-caps and small-caps run text with explicit cap=none override handling; full Office text parity remains incomplete."
        if name in {"CT_ColorScheme", "CT_Color", "CT_ColorMRU", "EG_ColorChoice", "CT_ScRgbColor", "CT_SRgbColor", "CT_HslColor", "CT_SystemColor", "CT_SchemeColor", "CT_PresetColor"}:
            return PARTIAL, "M05 resolves sRGB, scRGB, HSL, system, scheme, preset, phClr, and color-map colors into paint primitives; uncommon preset/system slots remain fallback-limited."
        if name == "EG_ColorTransform":
            return PARTIAL, "M05 applies DrawingML tint, shade, alpha/alphaMod/alphaOff, hue/saturation/luminance, RGB channel, grayscale, inverse, complement, and gamma transforms in source order."
        if name in {"CT_SolidColorFillProperties", "CT_GradientFillProperties", "CT_PatternFillProperties", "CT_GroupFillProperties", "EG_FillProperties", "CT_FillProperties", "CT_FillStyleList", "CT_BackgroundFillStyleList"}:
            return PARTIAL, "M05/M12 resolve noFill, solidFill, gradFill, pattFill, grpFill, direct table-cell fills, and theme fill/background fill style references into shared paint primitives; remaining gaps are image/tile fill details and advanced gradient clauses."
        if name in {"CT_TileInfoProperties", "CT_StretchInfoProperties", "EG_FillModeProperties"}:
            return PARTIAL, "M07 parses DrawingML blip fill stretch/tile mode and renders source-backed stretch/fillRect and default tiling with scale, offset, alignment, and flip metadata; M12 fixture manifests separate supported fill-mode metadata from actual unsupported records. Masked/soft-edge tiling remains partial."
        if name in {"CT_StyleMatrix", "CT_StyleMatrixReference", "CT_ShapeStyle", "CT_FontReference"}:
            return PARTIAL, "M05 resolves fillRef, lnRef, effectRef, fontRef, phClr, and theme style matrix references into fill, stroke, effect, and text-color paint primitives."
        if name in {"CT_FontCollection", "CT_FontScheme", "CT_Font"}:
            return PARTIAL, "M08 resolves theme and requested text font families through deterministic exact/substitute/fallback font sources and reports substitutions; full Office font inventory and script-specific font selection remain incomplete."
        if name in {"CT_ColorMapping", "CT_ColorMappingOverride", "CT_ColorSchemeAndMapping", "CT_ColorSchemeList"}:
            return PARTIAL, "M05 applies slide/layout/master color-map overrides to resolved theme colors; extra color schemes remain limited to current theme selection."
        if name in {"CT_Transform2D", "CT_GroupTransform2D"}:
            return PARTIAL, "M04 shared transform stack derives integer, fractional, clipped, rotated, and flipped primitive/debug bounds from OOXML xfrm EMUs; path/text/sampling semantics remain incomplete."
        if name in {"CT_ShapeProperties", "CT_GroupShapeProperties"}:
            return PARTIAL, "M06 combines resolved M05 paint with common preset/custom geometry, stroke, dash, join, compound line, and marker rendering; full preset catalog/effect/text semantics remain incomplete."
        if name in {"CT_PresetGeometry2D", "CT_CustomGeometry2D", "CT_Path2D"}:
            return PARTIAL, "M06 renders common preset geometry and custom path moveTo, lnTo, quadBezTo, cubicBezTo, arcTo, close, and multiple path entries into path primitives; full preset catalog and guide formulas remain incomplete."
        if name == "CT_LineProperties":
            return PARTIAL, "M06 renders source line width, preset/custom dash, cap, pen alignment for rects, join metadata, compound lines, and schema head/tail marker types; gradient/pattern stroke fills remain incomplete."
        if name in {"CT_LineEndProperties", "ST_LineEndType", "ST_LineEndWidth", "ST_LineEndLength"}:
            return PARTIAL, "M06 renders DrawingML head/tail line-end marker types triangle, stealth, diamond, oval, and arrow with source width/length hints."
        if name in {"EG_LineDashProperties", "CT_PresetLineDashProperties", "CT_DashStopList", "CT_DashStop", "ST_PresetLineDashVal"}:
            return PARTIAL, "M06 renders preset and custom dash stop sequences as source-derived stroke patterns."
        if name in {"EG_LineJoinProperties", "CT_LineJoinRound", "CT_LineJoinBevel", "CT_LineJoinMiterProperties"}:
            return PARTIAL, "M06 preserves line join semantics in stroke primitives and renders round joins for supported path outlines; bevel/miter use the stable segment stroke model."
        if name in {"ST_CompoundLine", "ST_LineCap", "ST_PenAlignment", "ST_LineWidth"}:
            return PARTIAL, "M06 renders DrawingML line width, cap, compound line variants, and rectangular pen alignment from source line properties."
        if name == "CT_TableCell":
            return PARTIAL, "M09 lowers DrawingML tables into table primitives and renders source grid extents, rows, columns, spans/merges, table flags, direct/style fills, and table text inputs. M12 table text-height measurement uses gridSpan when measuring source text minimums for row reflow, distributes rowSpan text minimums across the spanned rows, preserves authored blank paragraph line boxes with endParaRPr metrics for row-height measurement, and uses source text-minimum proportions for over-capacity first-row spanning/header tables; large real-world table layout/text residuals remain partial."
        if name == "CT_TableRow":
            return PARTIAL, "M09 lowers DrawingML tables into table primitives and renders source grid extents, rows, columns, spans/merges, table flags, direct/style fills, and table text inputs. M12 preserves Office row IDs, reflows row offsets when measured table-cell text needs a larger row minimum inside the same frame, derives row proportions from source text heights when all authored row heights are zero, and applies measured source text-minimum proportions when a first-row spanning/header table cannot satisfy all row minimums inside the fixed graphic frame; large real-world table layout/text residuals remain partial."
        if name == "CT_TableProperties":
            return PARTIAL, "M09 lowers DrawingML table properties into table primitives, including style flags and table style IDs. M12 parses direct table-property solidFill/noFill background semantics before style table backgrounds and reports authored table style IDs plus first/last/band row/column flags in object summaries; table-property effects and non-solid fill variants remain partial."
        if name in {"CT_Table", "CT_TableGrid", "CT_TableCol"}:
            return PARTIAL, "M09 lowers DrawingML tables into table primitives and renders source grid extents, rows, columns, spans/merges, table flags, direct/style fills, and table text inputs. M12 preserves Office column IDs from table-grid extensions and reports authored table style IDs plus first/last/band row/column flags in object summaries; large real-world table layout/text residuals remain partial."
        if name == "CT_TableCellProperties":
            return PARTIAL, "M09 parses and renders table cell margins, fills, four edge borders, diagonal borders lnTlToBr/lnBlToTr, line width/dash/cap/join/double compound, anchor/anchorCtr text anchoring, and reports visible non-solid fills/effects/unsupported line decorations. M12 preserves table-cell text overflow and vertical text metadata through existing text rendering/reporting paths; cell 3-D remains partial."
        if name in {"CT_TableStyle", "CT_TableStyleList", "CT_TablePartStyle", "CT_TableStyleCellStyle", "CT_TableCellBorderStyle", "CT_TableBackgroundStyle", "CT_TableStyleTextStyle"}:
            return PARTIAL, "M09 parses ppt/tableStyles.xml conditional regions, background fills/effects, text style, line references, inside/outside borders, and diagonal style borders. M12 resolves table background and cell-style fillRef entries through theme fill styles, lets explicit conditional-region boundary borders override inherited inside borders, and applies table-style text color, bold, italic, and font-family defaults into cell text layout; full Office table-style precedence and all effect/fill variants remain partial."
        if name == "tbl":
            return PARTIAL, "M09 recognizes and renders DrawingML table payloads inside graphic frames with source provenance and table-specific micro-fixture manifests; full table visual parity remains incomplete."
        if name in {"CT_TextBody", "CT_TextBodyProperties"}:
            return PARTIAL, "M08 preserves text body properties in text primitives and renders source-backed insets, wrap/overflow, anchor, normal/shape/no autofit metadata, and reports vertical/column/bidi gaps explicitly. M12 preserves CT_TextBodyProperties@rtlCol through parsing, primitive lowering, and object debug summaries, lowers fontScale/lnSpcReduction/spcFirstLastPara autofit metadata into text primitives, and reports wrap/autofit/spacing body-property summaries; authored right-to-left multi-column order remains pending until implemented. WordArt/vertical/column layout remain incomplete."
        if name in {"CT_TextParagraph", "CT_TextParagraphProperties", "CT_TextListStyle", "CT_TextCharacterProperties", "CT_RegularTextRun", "CT_TextLineBreak", "EG_TextRun"}:
            return PARTIAL, "M08 parses paragraph/run inheritance, margins, hanging indents, tab stops, line spacing, bullets, font/color/style properties, hard breaks, and uses a HarfBuzz-backed LTR shaping advance backend for layout. M12 preserves authored hyphen/slash wrap opportunities, empty paragraphs with endParaRPr metrics as blank layout lines, explicit buChar empty paragraphs as bullet lines, CT_TextCharacterProperties@lang through text layout and object-debug font-family summaries, inherited-size baseline runs through fallback-aware measurement/drawing, and CT_TextLineBreak rPr metrics as hard-break line metric segments; it also preserves CT_TextParagraphProperties rtl/eaLnBrk/latinLnBrk/hangingPunct flags. Authored rtl=1 paragraphs are reported as LTR fallback until bidi layout is implemented."
        if name in {"CT_TextNormalAutofit", "CT_TextShapeAutofit", "CT_TextNoAutofit", "EG_TextAutofit"}:
            return PARTIAL, "M08 preserves DrawingML autofit choice and applies source-backed normal/shape/no-autofit behavior for supported horizontal text; rotated shape-autofit and some normal-autofit edge cases are reported as simplified."
        if name in {"EG_TextBulletColor", "EG_TextBulletSize", "EG_TextBulletTypeface", "EG_TextBullet", "CT_TextBulletColorFollowText", "CT_TextBulletSizeFollowText", "CT_TextBulletSizePercent", "CT_TextBulletSizePoint", "CT_TextBulletTypefaceFollowText", "CT_TextAutonumberBullet", "CT_TextCharBullet", "CT_TextNoBullet", "ST_TextBulletStartAtNum", "ST_TextAutonumberScheme", "ST_TextBulletSize", "ST_TextBulletSizePercent"}:
            return PARTIAL, "M08 parses DrawingML bullet color, size, typeface, character, autonumber, and no-bullet properties into paragraph/list style layout with deterministic symbol fallbacks. M12 renders common alpha, arabic, and Roman ST_TextAutonumberScheme marker formats; picture bullets and locale-specific numbering parity remain incomplete."
        if name in {"CT_TextSpacing", "CT_TextSpacingPercent", "CT_TextSpacingPoint", "CT_TextTabStop", "CT_TextTabStopList", "ST_TextSpacingPoint", "ST_TextSpacingPercentOrPercentString", "ST_TextMargin", "ST_TextIndent", "ST_TextTabAlignType", "ST_TextAlignType", "ST_TextFontSize", "ST_TextTypeface", "ST_TextUnderlineType", "ST_TextStrikeType", "ST_TextColumnCount", "ST_TextFontScalePercentOrPercentString"}:
            return PARTIAL, "M08 parses and applies common DrawingML text sizing, spacing, margin, indent, tab, alignment, underline, strike, and font-size/typeface semantics for supported horizontal text; uncommon variants remain partial."
        if name in {"CT_EffectList", "CT_EffectProperties", "CT_OuterShadowEffect", "EG_EffectProperties"}:
            return PARTIAL, "M10 renders source-backed outer shadows, glow, and soft edges for supported static shape/picture geometry, preserves effect primitives, and reports remaining visible effects explicitly; full effect ordering and host parity remain incomplete."
        if name == "CT_SoftEdgesEffect":
            return PARTIAL, "M10 renders DrawingML softEdge radius as an alpha-mask blur for supported static shapes and pictures; M12 fixture manifests keep softEdge as supported effect metadata instead of expected unsupported content. Full host edge parity remains incomplete."
        if name == "CT_GlowEffect":
            return PARTIAL, "M10 renders DrawingML glow radius/color as a source-backed blurred alpha mask for supported static shapes and pictures; full host glow parity remains incomplete."
        if name == "CT_BlurEffect":
            return PARTIAL, "M12 parses DrawingML blur rad/grow and renders source-backed RGBA blur for supported static shape/picture objects plus simple blip images; combined blip blur with higher-order object effects remains an explicit partial report."
        if name == "CT_AlphaOutsetEffect":
            return PARTIAL, "M12 parses DrawingML alphaOutset radius and renders source-backed alpha-mask expansion for supported static shape and picture objects; full effect ordering and host parity remains incomplete."
        if name == "CT_RelativeOffsetEffect":
            return PARTIAL, "M12 parses DrawingML relOff tx/ty percentages and renders source-backed object-layer translation for supported static shape and picture objects; full effect ordering and host parity remains incomplete."
        if name == "CT_TransformEffect":
            return PARTIAL, "M12 parses DrawingML xfrm tx/ty coordinates and renders source-backed object-layer translation for supported static shape and picture objects; scale/skew transform attributes are explicitly reported as partial."
        if name == "CT_FillOverlayEffect":
            return PARTIAL, "M12 parses DrawingML fillOverlay fill and blend mode, renders source-backed fill overlays for supported static shape/picture objects plus source-space blip images, and keeps complex effect ordering parity partial."
        if name == "ST_BlendMode":
            return PARTIAL, "M12 implements all DrawingML ST_BlendMode enum values over, mult, screen, darken, and lighten for supported fillOverlay rendering; blend effect graph usage remains partial."
        if name == "CT_InnerShadowEffect":
            return PARTIAL, "M12 parses DrawingML innerShdw color, blur, distance, and direction, renders a source-backed inner alpha-mask shadow for supported static shape and picture objects, and keeps full effect ordering parity partial."
        if name == "CT_PresetShadowEffect":
            return PARTIAL, "M10 maps DrawingML preset shadow color, distance, and direction to the static shadow renderer with an explicit simplified-preset diagnostic; full preset style parity remains incomplete."
        if name == "ST_PresetShadowVal":
            return PARTIAL, "M10 preserves preset shadow usage through a simplified-preset diagnostic while rendering source color, distance, and direction."
        if name == "CT_ReflectionEffect":
            return PARTIAL, "M12 parses DrawingML reflection alpha/fade/distance attributes, renders a source-backed bottom mirror reflection for supported static shape and picture objects, and reports non-bottom transform variants as simplified."
        if name == "CT_BlendEffect":
            return PARTIAL, "M12 flattens blend effectDag containers when their child container contains already-supported static effects, while reporting full blend graph compositing as partial."
        if name in {"EG_Effect", "CT_EffectContainer", "ST_EffectContainerType"}:
            return PARTIAL, "M12 flattens simple effectDag containers containing supported static effects into the normal effect renderer and reports unsupported graph ordering or unimplemented graph-only effect nodes explicitly."
        if name == "CT_BlipFillProperties":
            return PARTIAL, "M07 preserves embedded and linked image relationships, signed srcRect crop/padding, rotWithShape, and stretch/tile fill mode for the picture backend; M12 object-debug summaries now report resolved media part, content type, and decoded intrinsic image size for picture sources, and fixture expected unsupported records use explicit image/effect unsupported fields instead of supported image metadata such as `fillMode=stretch`. Full sampling parity remains incomplete."
        if name == "CT_Blip":
            return PARTIAL, "M07 parses visible blip effects, renders alphaModFix, alphaBiLevel, alphaCeiling, alphaFloor, alphaInv, alphaRepl, biLevel, clrChange, clrRepl, duotone, grayscl, lum, hsl, tint, simple blur, fillOverlay, and scalar-container alphaMod in source/render space, and reports remaining visible effects explicitly. M12 preserves CT_Blip@cstate compression metadata in picture primitives and object-debug summaries, records decoded source image dimensions in picture object summaries, and fixture manifests keep supported blip metadata out of expected unsupported records."
        if name in {"CT_AlphaBiLevelEffect", "CT_AlphaCeilingEffect", "CT_AlphaFloorEffect", "CT_AlphaInverseEffect", "CT_AlphaModulateFixedEffect", "CT_AlphaReplaceEffect", "CT_BiLevelEffect", "CT_ColorChangeEffect", "CT_ColorReplaceEffect", "CT_DuotoneEffect", "CT_GrayscaleEffect", "CT_LuminanceEffect", "CT_HSLEffect", "CT_TintEffect"}:
            return PARTIAL, "M07 renders this blip effect for static picture images in source space; complex cross-effect and host-renderer edge parity remains incomplete."
        if name == "CT_AlphaModulateEffect":
            return PARTIAL, "M12 parses DrawingML alphaMod effect containers and renders source-space alpha modulation when the container collapses to supported scalar alphaModFix children; arbitrary effect containers remain explicitly partial."
        if name in DML_MAIN_PARTIAL_NAMES or name.startswith(DML_MAIN_PARTIAL_PREFIXES):
            return PARTIAL, "Part of current DrawingML render subset; full source semantics are not complete."
        if name.startswith(DML_MAIN_UNSUPPORTED_PREFIXES):
            return UNSUPPORTED, "Visible behavior is unsupported or only reported as partial in current renderer."
        if name in {"graphic", "CT_GraphicalObject", "CT_GraphicalObjectData"}:
            return PARTIAL, "Graphic payloads are recognized for tables, simple diagrams, and preserved chart payloads; chart graphics remain hard-rendering implementation work while OLE/media payloads remain preserve/report boundaries."
        return NO_EVIDENCE, "No explicit renderer coverage evidence found in the maintained docs/tests."

    return NO_EVIDENCE, "No explicit renderer coverage evidence found in the maintained docs/tests."


def queue_for(row: dict[str, str]) -> str:
    status = row["status"]
    file = row["file"]
    name = row["name"]

    if status == OUT_OF_SCOPE:
        return QUEUE_OUT_OF_SCOPE
    if status == UNSUPPORTED:
        return QUEUE_UNSUPPORTED_PRESERVE
    if status == SUPPORTED:
        return QUEUE_CORE_STATIC
    if status == NO_EVIDENCE:
        return QUEUE_HARD_RENDERING
    if status == PARTIAL:
        if file in {"dml-chart.xsd", "dml-chartDrawing.xsd", "dml-lockedCanvas.xsd"}:
            return QUEUE_HARD_RENDERING
        if file == "dml-diagram.xsd":
            return QUEUE_HARD_RENDERING
        if file == "pml.xsd" or file == "dml-picture.xsd":
            return QUEUE_COMMON_PARTIAL
        if name.startswith(("CT_Color", "CT_Fill", "CT_Solid", "CT_Gradient", "CT_Blip", "CT_Blur", "CT_Text", "CT_Table", "CT_Line", "CT_Transform", "CT_GroupTransform", "CT_PresetGeometry", "CT_CustomGeometry", "CT_Path2D", "CT_InnerShadow", "CT_OuterShadow", "CT_Reflection", "CT_Effect")):
            return QUEUE_HARD_RENDERING
        return QUEUE_COMMON_PARTIAL
    return QUEUE_HARD_RENDERING


def md_escape(value: str) -> str:
    return value.replace("|", "\\|").replace("\n", " ")


def build_rows() -> list[dict[str, str]]:
    all_rows: list[dict[str, str]] = []
    for filename in TARGET_FILES:
        all_rows.extend(declaration_rows(SCHEMA_DIR / filename))

    for row in all_rows:
        status, note = coverage_for(row)
        row["status"] = status
        row["note"] = note
        row["queue"] = queue_for(row)
    return all_rows


def summary_for(all_rows: list[dict[str, str]]) -> dict[str, object]:
    queue_status_counts: dict[str, dict[str, int]] = {}
    queue_totals = Counter(row["queue"] for row in all_rows)
    status_counts = Counter(row["status"] for row in all_rows)
    for queue in QUEUES:
        rows = [row for row in all_rows if row["queue"] == queue]
        counts = Counter(row["status"] for row in rows)
        queue_status_counts[queue] = {status: counts[status] for status in [SUPPORTED, PARTIAL, UNSUPPORTED, OUT_OF_SCOPE, NO_EVIDENCE]}
    return {
        "source": "docs/OOXML_DRAWINGML_COVERAGE_MATRIX.md",
        "schema_dir": "docs/specs/ecma-376/part1/schema/strict",
        "total_declarations": len(all_rows),
        "statuses": {status: status_counts[status] for status in [SUPPORTED, PARTIAL, UNSUPPORTED, OUT_OF_SCOPE, NO_EVIDENCE]},
        "queues": {queue: queue_totals[queue] for queue in QUEUES},
        "queue_status_counts": queue_status_counts,
        "rows": [
            {
                "anchor": f"{row['file']}:{row['line']}",
                "schema_file": row["file"],
                "kind": row["kind"],
                "declaration": row["name"],
                "members": row["members"],
                "status": row["status"],
                "queue": row["queue"],
                "evidence_gap": row["note"],
            }
            for row in all_rows
        ],
    }


def render_markdown(all_rows: list[dict[str, str]]) -> str:
    by_file = defaultdict(list)
    for row in all_rows:
        by_file[row["file"]].append(row)

    status_counts = Counter(row["status"] for row in all_rows)
    queue_counts = Counter(row["queue"] for row in all_rows)

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
            "- **Unsupported**: source evidence proves the declaration requires",
            "  runtime behavior or external application execution that the static PPTX",
            "  renderer cannot represent as still output.",
            "- **Out of renderer scope**: schema belongs to another host application or a",
            "  non-static-rendering area outside the current Puppt renderer goal.",
            "- **Unimplemented / no evidence**: the declaration exists in the spec but has no",
            "  maintained code/doc/test evidence yet, so it is treated as unimplemented",
            "  until proven otherwise.",
            "",
            "Status promotion rules:",
            "",
            "- **Supported**: requires source-schema anchors, parser/lowering evidence,",
            "  deterministic synthetic fixtures, focused renderer/reporting tests, and",
            "  explicit documentation of implemented semantics, remaining",
            "  supported-scope gaps, and source-proven static-rendering boundaries.",
            "- **Partial**: requires evidence that at least one source-backed semantic subset",
            "  is parsed, lowered, rendered or reported, and tested, plus a maintained gap",
            "  note for incomplete children, attributes, renderer behavior, or fixtures.",
            "- **Unsupported**: requires source evidence that the declaration is",
            "  impossible for the static renderer to represent, plus detection,",
            "  preservation where package semantics allow it, and explicit",
            "  renderer/JSON reporting. Missing implementation, high pixel diff,",
            "  local fixture failure, or difficult static-rendering behavior stays",
            "  Partial or hard-rendering work.",
            "- **Out of renderer scope**: requires a documented reason that the declaration",
            "  belongs outside Puppt's static PresentationML/DrawingML rendering target;",
            "  moving it into scope requires a milestone update before implementation.",
            "- **Unimplemented / no evidence**: may move only after the implementation",
            "  supplies source-schema anchors, semantic/lowering evidence, fixtures or",
            "  explicit source-proven impossibility reporting, and checklist/log",
            "  evidence for the change.",
            "",
            "Work queue definitions:",
            "",
            "- **core-static**: already supported static-rendering/package declarations",
            "  that must remain stable while later milestones change renderer internals.",
            "- **common-partial**: common PresentationML or object declarations with some",
            "  current source-backed behavior and known missing subclauses.",
            "- **hard-rendering**: unimplemented or algorithm-heavy rendering clauses that",
            "  need dedicated primitive milestones and deterministic fixtures.",
            "- **unsupported-preserve**: source-proven static-renderer impossibility",
            "  declarations that must be preserved where possible and reported",
            "  explicitly when encountered.",
            "- **out-of-scope**: host-application, authoring, playback, or non-PresentationML",
            "  declarations with source evidence outside the static slide-rendering target.",
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

    lines.extend(["", "## Queue Totals", "", "| Queue | Count |", "|---|---:|"])
    for queue in QUEUES:
        lines.append(f"| `{queue}` | {queue_counts[queue]} |")

    lines.extend(["", "## Queue By Status", "", "| Queue | Supported | Partial | Unsupported | Out of scope | Unimplemented/no evidence |", "|---|---:|---:|---:|---:|---:|"])
    for queue in QUEUES:
        rows = [row for row in all_rows if row["queue"] == queue]
        counts = Counter(row["status"] for row in rows)
        lines.append(
            f"| `{queue}` | {counts[SUPPORTED]} | {counts[PARTIAL]} | {counts[UNSUPPORTED]} | {counts[OUT_OF_SCOPE]} | {counts[NO_EVIDENCE]} |"
        )

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
                "| Anchor | Kind | Declaration | Members | Status | Queue | Evidence / gap |",
                "|---|---|---|---|---|---|---|",
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
                        f"`{row['queue']}`",
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
            "   The same command also writes the machine-readable summary:",
            "   `docs/renderer-coverage-summary.json`.",
            "",
            "2. List current queue/status counts with:",
            "",
            "   ```text",
            "   python3 tools/generate_ooxml_drawingml_audit.py --print-summary",
            "   ```",
            "",
            "3. Apply the status promotion rules above before changing a row's status.",
            "4. Do not mark a declaration supported because a real-world screenshot looks",
            "   close. Source semantics and fixture proof come first.",
        ]
    )
    return "\n".join(lines) + "\n"


def print_summary(summary: dict[str, object]) -> None:
    queue_status_counts = summary["queue_status_counts"]
    print("Queue | Supported | Partial | Unsupported | Out of scope | Unimplemented/no evidence | Total")
    print("---|---:|---:|---:|---:|---:|---:")
    for queue in QUEUES:
        counts = queue_status_counts[queue]
        total = summary["queues"][queue]
        print(
            f"{queue} | {counts[SUPPORTED]} | {counts[PARTIAL]} | {counts[UNSUPPORTED]} | "
            f"{counts[OUT_OF_SCOPE]} | {counts[NO_EVIDENCE]} | {total}"
        )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--print-summary", action="store_true", help="print queue/status counts after regenerating files")
    args = parser.parse_args()

    rows = build_rows()
    summary = summary_for(rows)
    OUTPUT.write_text(render_markdown(rows), encoding="utf-8")
    SUMMARY_OUTPUT.write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    if args.print_summary:
        print_summary(summary)


if __name__ == "__main__":
    main()
