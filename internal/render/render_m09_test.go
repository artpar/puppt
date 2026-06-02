package render

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/artpar/puppt/internal/pptx"
)

func TestM09ParseTableModelPreservesDiagonalCellBorders(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
	<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
	<a:tr h="914400">
		<a:tc>
			<a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
			<a:tcPr>
				<a:lnTlToBr w="38100" cap="flat"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:lnTlToBr>
				<a:lnBlToTr w="38100" cap="flat"><a:solidFill><a:srgbClr val="0000FF"/></a:solidFill></a:lnBlToTr>
			</a:tcPr>
		</a:tc>
	</a:tr>
</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	cell := table.Rows[0].Cells[0]
	if !cell.BorderTopLeftToBottomRight.Specified || !cell.BorderTopLeftToBottomRight.HasLine || cell.BorderTopLeftToBottomRight.Color != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("expected parsed top-left to bottom-right border, got %+v", cell.BorderTopLeftToBottomRight)
	}
	if !cell.BorderBottomLeftToTopRight.Specified || !cell.BorderBottomLeftToTopRight.HasLine || cell.BorderBottomLeftToTopRight.Color != (color.RGBA{B: 0xff, A: 0xff}) {
		t.Fatalf("expected parsed bottom-left to top-right border, got %+v", cell.BorderBottomLeftToTopRight)
	}
	if len(table.UnsupportedFeatures) != 0 {
		t.Fatalf("solid diagonal borders are supported and should not be reported partial: %+v", table.UnsupportedFeatures)
	}
}

func TestM09RenderGraphicFramePaintsDiagonalCellBorders(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Diagonal Table",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns: []int64{1},
			Rows: []tableRow{{
				Height: 1,
				Cells: []tableCell{{
					NoFill:       true,
					BorderTop:    tableCellBorder{Specified: true, NoLine: true},
					BorderBottom: tableCellBorder{Specified: true, NoLine: true},
					BorderLeft:   tableCellBorder{Specified: true, NoLine: true},
					BorderRight:  tableCellBorder{Specified: true, NoLine: true},
					BorderTopLeftToBottomRight: tableCellBorder{
						Specified: true,
						HasLine:   true,
						Color:     color.RGBA{R: 0xff, A: 0xff},
						Width:     emuPerInch / 24,
						Cap:       "flat",
					},
					BorderBottomLeftToTopRight: tableCellBorder{
						Specified: true,
						HasLine:   true,
						Color:     color.RGBA{B: 0xff, A: 0xff},
						Width:     emuPerInch / 24,
						Cap:       "flat",
					},
				}},
			}},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected diagonal table result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(24, 24); got.R == 0 {
		t.Fatalf("expected top-left to bottom-right diagonal to paint red coverage, got %#v", got)
	}
	if got := img.RGBAAt(24, 72); got.B == 0 {
		t.Fatalf("expected bottom-left to top-right diagonal to paint blue coverage, got %#v", got)
	}
	if _, _, _, a := img.At(48, 8).RGBA(); a != 0 {
		t.Fatalf("suppressed outer table borders should remain transparent away from diagonals, alpha=%04x", a)
	}
}

func TestM09TableStyleDiagonalBordersApplyThroughResolvedCellStyle(t *testing.T) {
	styles := parseTableStyles([]byte(`<a:tblStyleLst xmlns:a="a">
	<a:tblStyle styleId="{DIAGONAL}" styleName="Diagonal">
		<a:wholeTbl>
			<a:tcStyle><a:tcBdr>
				<a:tl2br><a:ln w="38100" cap="flat"><a:solidFill><a:srgbClr val="00AA00"/></a:solidFill></a:ln></a:tl2br>
				<a:tr2bl><a:ln w="38100" cap="flat"><a:solidFill><a:srgbClr val="AA00AA"/></a:solidFill></a:ln></a:tr2bl>
			</a:tcBdr></a:tcStyle>
		</a:wholeTbl>
	</a:tblStyle>
</a:tblStyleLst>`), defaultThemeColors(), themeFonts{}, themeFillStyles{}, themeLineStyles{}, themeEffectStyles{})
	table := tableModel{
		StyleID: "{DIAGONAL}",
		Columns: []int64{1},
		Rows: []tableRow{{
			Height: 1,
			Cells:  []tableCell{{NoFill: true}},
		}},
	}
	resolved := resolvedTableCellStyle(table, styles, 0, 0)
	if !resolved.Borders.TopLeftToBottomRight.Specified || !resolved.Borders.TopLeftToBottomRight.HasLine || resolved.Borders.TopLeftToBottomRight.Color != (color.RGBA{G: 0xaa, A: 0xff}) {
		t.Fatalf("expected resolved style top-left to bottom-right border, got %+v", resolved.Borders.TopLeftToBottomRight)
	}
	if !resolved.Borders.BottomLeftToTopRight.Specified || !resolved.Borders.BottomLeftToTopRight.HasLine || resolved.Borders.BottomLeftToTopRight.Color != (color.RGBA{R: 0xaa, B: 0xaa, A: 0xff}) {
		t.Fatalf("expected resolved style bottom-left to top-right border, got %+v", resolved.Borders.BottomLeftToTopRight)
	}

	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Styled Diagonal Table",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table:        table,
	}
	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, styles)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected styled diagonal table result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(24, 24); got.G == 0 {
		t.Fatalf("expected styled top-left to bottom-right diagonal to paint green coverage, got %#v", got)
	}
	if got := img.RGBAAt(24, 72); got.R == 0 || got.B == 0 {
		t.Fatalf("expected styled bottom-left to top-right diagonal to paint magenta coverage, got %#v", got)
	}
}

func TestM09TableStyleRegionBoundaryBorderOverridesInsideBorder(t *testing.T) {
	styles := parseTableStyles([]byte(`<a:tblStyleLst xmlns:a="a">
	<a:tblStyle styleId="{FIRSTROW}" styleName="First Row Boundary">
		<a:wholeTbl>
			<a:tcStyle><a:tcBdr>
				<a:insideH><a:ln w="12700"><a:solidFill><a:srgbClr val="000000"/></a:solidFill></a:ln></a:insideH>
			</a:tcBdr></a:tcStyle>
		</a:wholeTbl>
		<a:firstRow>
			<a:tcStyle><a:tcBdr>
				<a:bottom><a:ln w="38100"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:ln></a:bottom>
			</a:tcBdr></a:tcStyle>
		</a:firstRow>
	</a:tblStyle>
</a:tblStyleLst>`), defaultThemeColors(), themeFonts{}, themeFillStyles{}, themeLineStyles{}, themeEffectStyles{})
	table := tableModel{
		StyleID:  "{FIRSTROW}",
		FirstRow: true,
		Columns:  []int64{1},
		Rows: []tableRow{
			{Height: 1, Cells: []tableCell{{}}},
			{Height: 1, Cells: []tableCell{{}}},
		},
	}

	firstRow := resolvedTableCellStyle(table, styles, 0, 0)
	border := tableEdgeBorder(firstRow.Borders, tableEdgeBottom, 0, 0, 2, 1)
	if !border.Specified || !border.HasLine || border.Width != 38100 || border.Color != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("expected firstRow bottom border to override inherited insideH, got %+v", border)
	}
	bodyRow := resolvedTableCellStyle(table, styles, 1, 0)
	border = tableEdgeBorder(bodyRow.Borders, tableEdgeTop, 1, 0, 2, 1)
	if !border.Specified || !border.HasLine || border.Width != 12700 || border.Color != (color.RGBA{A: 0xff}) {
		t.Fatalf("expected body top border to use wholeTbl insideH, got %+v", border)
	}

	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Styled First Row Table",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table:        table,
	}
	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, styles)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected styled first-row table result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(48, 48); got.R == 0 || got.G != 0 || got.B != 0 {
		t.Fatalf("expected explicit firstRow bottom border to repaint over inherited insideH, got %#v", got)
	}
}

func TestM09DiagonalBorderKnownLineEndDecorationsAreRenderedScope(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
	<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
	<a:tr h="914400">
		<a:tc>
			<a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
			<a:tcPr>
				<a:lnTlToBr cap="unsupportedCap"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:headEnd type="triangle"/></a:lnTlToBr>
			</a:tcPr>
		</a:tc>
	</a:tr>
</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	if !slices.Contains(table.UnsupportedFeatures, "uses border line caps that were not rendered") {
		t.Fatalf("expected diagonal border cap unsupported feature in %+v", table.UnsupportedFeatures)
	}
	if slices.Contains(table.UnsupportedFeatures, "uses border line end decorations that were not rendered") {
		t.Fatalf("known line-end marker should be rendered, not reported unsupported: %+v", table.UnsupportedFeatures)
	}
	if table.Rows[0].Cells[0].BorderTopLeftToBottomRight.HeadMarker != "triangle" {
		t.Fatalf("expected parsed diagonal border marker, got %+v", table.Rows[0].Cells[0].BorderTopLeftToBottomRight)
	}
}

func TestM09UnknownDiagonalBorderLineEndDecorationIsReported(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
	<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
	<a:tr h="914400">
		<a:tc>
			<a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
			<a:tcPr>
				<a:lnTlToBr><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:headEnd type="futureMarker"/></a:lnTlToBr>
			</a:tcPr>
		</a:tc>
	</a:tr>
</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	if !slices.Contains(table.UnsupportedFeatures, "uses border line end decorations that were not rendered") {
		t.Fatalf("expected unknown diagonal border marker unsupported feature in %+v", table.UnsupportedFeatures)
	}
}
