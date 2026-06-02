package render

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

const (
	commandName            = "render"
	emuPerInch             = 914400
	defaultOutputDPI       = 72
	defaultSlideCX         = 12192000
	defaultSlideCY         = 6858000
	defaultTextInsetXEMU   = 91440
	defaultTextInsetYEMU   = 45720
	defaultTextTabPixels   = defaultOutputDPI
	customBezierSegments   = 48
	unsupportedCode        = "render_unsupported_object"
	partialUnsupportedCode = "render_partial_object"
	diagramDataRelType     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/diagramData"
	diagramDrawingRelType  = "http://schemas.microsoft.com/office/2007/relationships/diagramDrawing"
	chartRelType           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart"
	themeRelType           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"
)

func Render(ctx context.Context, inputPath string, options Options) (model.CommandResult, error) {
	result := model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       commandName,
		Status:        "error",
		Input:         inputPath,
		Warnings:      []model.Warning{},
		Errors:        []model.ErrorItem{},
		Unsupported:   []model.SkipItem{},
		Summary:       model.Summary{Human: "Render failed."},
	}
	if options.OutputPath != "" {
		result.Output = &options.OutputPath
	}
	if options.OutputPath == "" {
		return result, errors.New("render output path is required")
	}

	pkg, err := pptx.Open(ctx, inputPath)
	if err != nil {
		return result, err
	}
	if options.SlideNumber < 1 || options.SlideNumber > len(pkg.SlideParts) {
		return result, fmt.Errorf("slide %d out of range 1..%d", options.SlideNumber, len(pkg.SlideParts))
	}

	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	dpi := normalizeOutputDPI(options.DPI)
	width := emuToPixelsAtDPI(size.CX, dpi)
	height := emuToPixelsAtDPI(size.CY, dpi)
	if width <= 0 || height <= 0 {
		return result, fmt.Errorf("invalid slide size %dx%d EMU", size.CX, size.CY)
	}

	slidePart := pkg.SlideParts[options.SlideNumber-1]
	theme := packageThemeColors(pkg)
	fonts := packageThemeFonts(pkg)
	themeForPart := func(part string) themeColors {
		return themeColorsForPart(pkg, part, theme)
	}
	fontsForPart := func(part string) themeFonts {
		return themeFontsForPart(pkg, part, fonts)
	}
	renderParts := inheritedRenderParts(pkg, slidePart)
	paintParts := visibleRenderParts(pkg, slidePart, renderParts)
	placeholderSources := inheritedPlaceholderSourcesWithThemeResolver(pkg, renderParts, slidePart, themeForPart)
	textStyles := inheritedTextStylesWithThemeResolver(pkg, renderParts, slidePart, themeForPart)
	background := inheritedBackgroundWithThemeResolver(pkg, renderParts, themeForPart)
	headerFooter := inheritedHeaderFooterSettings(pkg, renderParts)
	if !presentationShowsSpecialPlaceholdersOnTitleSlide(pkg) && slideUsesTitleLayout(pkg, slidePart) {
		headerFooter = headerFooterSettings{}
	}
	inheritedHeaderFooterPart := inheritedHeaderFooterRenderPart(pkg, paintParts, slidePart, headerFooter)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var unsupported []model.SkipItem
	unsupported = append(unsupported, presentationUnsupportedItems(pkg.PresentationPath, pkg.Parts[pkg.PresentationPath])...)
	if objectDebugUsesOwnBackground(options.ObjectDebug) {
		if color, ok := objectDebugBackgroundColor(options.ObjectDebug); ok {
			draw.Draw(img, img.Bounds(), &image.Uniform{C: color}, image.Point{}, draw.Src)
		}
	} else if background.HasGradient {
		drawGradientBackground(img, background.Gradient)
		if !background.Gradient.FullySupported {
			unsupported = append(unsupported, unsupportedItem(background.Part, partialUnsupportedCode, "slide background gradient was rendered with simplified layout"))
		}
	} else if background.HasPattern {
		drawPatternRect(img, img.Bounds(), background.Pattern)
	} else {
		draw.Draw(img, img.Bounds(), &image.Uniform{C: background.Color}, image.Point{}, draw.Src)
	}
	for _, message := range background.Unsupported {
		unsupported = append(unsupported, unsupportedItem(background.Part, partialUnsupportedCode, message))
	}

	for _, renderPart := range paintParts {
		partTheme := themeForPart(renderPart)
		partFonts := fontsForPart(renderPart)
		partLineStyles := themeLineStylesForPart(pkg, renderPart)
		effectStyles := themeEffectStylesForPart(pkg, renderPart)
		fillStyles := themeFillStylesForPart(pkg, renderPart)
		tableStyles := packageTableStyles(pkg, partTheme, partFonts, fillStyles, partLineStyles, effectStyles)
		elements := collectSlideElementsWithThemeEffectsAndFills(pkg.Parts[renderPart], partTheme, effectStyles, fillStyles, partLineStyles)
		if renderPart != slidePart {
			elements = filterInheritedPlaceholdersForRender(elements, placeholderSources, headerFooter, renderPart == inheritedHeaderFooterPart)
		} else {
			elements = resolveSlidePlaceholders(elements, placeholderSources)
			elements = applyInheritedTextStyles(elements, textStyles)
		}
		elements = applyInheritedTableTextStyles(elements, textStyles)
		elements = applyThemeFontFamilies(elements, partFonts)
		elements = resolveTextFields(elements, options.SlideNumber)
		unsupported = append(unsupported, renderElementsWithDebug(pkg, slidePart, renderPart, size, img, elements, tableStyles, options.ObjectDebug)...)
		unsupported = append(unsupported, unsupportedItems(renderPart, elements)...)
		unsupported = append(unsupported, timingUnsupportedItems(renderPart, pkg.Parts[renderPart], elements)...)
	}
	applyDisplayP3OutputTransform(img)
	if err := writePNGWithDPI(options.OutputPath, img, dpi); err != nil {
		return result, err
	}

	result.Status = "ok"
	result.Summary = model.Summary{Human: fmt.Sprintf("Rendered slide %d to %s.", options.SlideNumber, options.OutputPath)}
	if len(unsupported) > 0 {
		result.Status = "partial"
		result.Summary = model.Summary{Human: fmt.Sprintf("Rendered slide %d with %d unsupported object(s).", options.SlideNumber, len(unsupported))}
		result.Unsupported = unsupported
	}
	result.Render = &model.Render{
		SlideNumber: options.SlideNumber,
		SlidePart:   slidePart,
		Width:       width,
		Height:      height,
	}
	return result, nil
}
