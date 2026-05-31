package render

import (
	"bytes"
	"compress/zlib"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/vector"
)

func renderPicture(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship) []model.SkipItem {
	if element.EmbedID == "" {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q has no embedded image relationship", elementLabel(*element)))}
	}
	relationship, ok := relationships[element.EmbedID]
	if !ok {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q references missing relationship %q", elementLabel(*element), element.EmbedID))}
	}
	if relationship.Type != pptx.ImageRelType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q uses unsupported relationship %q", elementLabel(*element), relationship.Type))}
	}
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q has no renderable transform", elementLabel(*element)))}
	}

	source, targetPart, partialUnsupported := pictureSourceImage(pkg, slidePart, element, relationships, relationship)
	if source == nil {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q uses unsupported image data %q: %v", elementLabel(*element), targetPart, partialUnsupported))}
	}

	target := image.Rect(
		scaleEMU(element.OffX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY, size.CY, img.Bounds().Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	)
	var unsupported []model.SkipItem
	if element.HasShadow {
		for _, message := range shadowTransformUnsupportedMessages(*element) {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", elementLabel(*element), message)))
		}
		if drawPictureShadow(img, target, *element, size) {
			// Supported picture shadows are painted before the image so the image occludes the inner shadow area.
		} else if element.ShadowColor.A != 0 {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q outer shadow geometry was not rendered", elementLabel(*element))))
		}
	}
	for _, message := range shape3DUnsupportedMessages(*element) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", elementLabel(*element), message)))
	}
	pictureImage, pictureBounds := pictureSourceForElement(source, *element)
	softEdgeRendered := drawPictureRaster(img, target, pictureImage, pictureBounds, *element, size)
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		if normalizedRotationDegrees(element.Rotation) == 0 {
			drawPictureOutline(img, target, *element, lineWidth)
		}
	}
	element.Rendered = true

	if len(element.CustomPath) >= 3 && len(element.CustomPathUnsupported) > 0 {
		for _, message := range element.CustomPathUnsupported {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", elementLabel(*element), message)))
		}
	}
	if element.HasSoftEdge {
		if !softEdgeRendered {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q soft edge was not rendered", elementLabel(*element))))
		}
	}
	if partialUnsupported != nil {
		if strings.EqualFold(path.Ext(targetPart), ".svg") {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q rendered from SVG because fallback raster could not be decoded: %v", elementLabel(*element), partialUnsupported)))
		} else {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q rendered from fallback raster because SVG image could not be decoded: %v", elementLabel(*element), partialUnsupported)))
		}
	}
	return unsupported
}

func drawPictureRaster(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, element slideElement, size slideSize) bool {
	rotation := normalizedRotationDegrees(element.Rotation)
	if !pictureRotatesWithShape(element) {
		rotation = 0
	}
	if rotation == 0 {
		return drawPictureRasterLayer(img, target, pictureImage, pictureBounds, element, size, img.Bounds().Dx())
	}
	if target.Empty() {
		return false
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	layerTarget := layer.Bounds()
	softEdgeRendered := drawPictureRasterLayer(layer, layerTarget, pictureImage, pictureBounds, element, size, img.Bounds().Dx())
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		drawPictureOutline(layer, layerTarget, element, lineWidth)
	}
	rotated := rotateRGBA(layer, rotation)
	center := image.Point{X: target.Min.X + target.Dx()/2, Y: target.Min.Y + target.Dy()/2}
	dst := image.Rect(center.X-rotated.Bounds().Dx()/2, center.Y-rotated.Bounds().Dy()/2, center.X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(), center.Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy())
	drawRGBAAt(img, dst, rotated)
	return softEdgeRendered
}

func pictureRotatesWithShape(element slideElement) bool {
	return !element.HasBlipRotWithShape || element.BlipRotWithShape
}

func drawPictureRasterLayer(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, element slideElement, size slideSize, outputWidth int) bool {
	if element.HasSoftEdge && len(element.CustomPath) < 3 {
		scaleImageWithSoftEdge(img, target, pictureImage, pictureBounds, softEdgeRadiusPixels(element, size, outputWidth))
		return true
	}
	if len(element.CustomPath) >= 3 {
		scaleImageWithCustomMask(img, target, pictureImage, pictureBounds, element.CustomPath, element.CustomPathCommands)
		return false
	}
	scaleImage(img, target, pictureImage, pictureBounds)
	return false
}

func drawPictureOutline(img *image.RGBA, target image.Rectangle, element slideElement, lineWidth int) {
	drawStyledRectOutlineAlignedWithCap(img, target, element.LineColor, lineWidth, element.LineDash, element.LineAlign, element.LineCap)
}

func drawPictureShadow(img *image.RGBA, target image.Rectangle, element slideElement, size slideSize) bool {
	if element.ShadowColor.A == 0 {
		return false
	}
	offset := shadowOffset(element, size, img.Bounds().Dx())
	shadowBounds := target.Add(offset)
	blur := shadowBlurPixels(element, size, img.Bounds().Dx())
	if !shadowIntersectsCanvas(shadowBounds, blur, img.Bounds()) {
		return false
	}
	if len(element.CustomPath) >= 3 {
		drawSoftPolygon(img, shadowBounds, element.CustomPath, element.ShadowColor, blur)
	} else {
		drawSoftRect(img, shadowBounds, element.ShadowColor, blur)
	}
	return true
}

func pictureSourceImage(pkg *pptx.Package, slidePart string, element *slideElement, relationships map[string]pptx.Relationship, fallbackRelationship pptx.Relationship) (image.Image, string, error) {
	fallback, fallbackPart, fallbackErr := fallbackPictureSourceImage(pkg, slidePart, fallbackRelationship)
	if fallbackErr == nil {
		return fallback, fallbackPart, nil
	}
	if element.SVGEmbedID == "" {
		return nil, fallbackPart, fallbackErr
	}
	relationship, ok := relationships[element.SVGEmbedID]
	if !ok || relationship.Type != pptx.ImageRelType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
		return nil, fallbackPart, fallbackErr
	}
	targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
	data, ok := pkg.Parts[targetPart]
	if !ok {
		return nil, targetPart, fallbackErr
	}
	source, err := decodeImage(targetPart, pkg.ContentTypes.ForPart(targetPart), data)
	if err != nil {
		return nil, targetPart, fallbackErr
	}
	return source, targetPart, fallbackErr
}

func fallbackPictureSourceImage(pkg *pptx.Package, slidePart string, relationship pptx.Relationship) (image.Image, string, error) {
	targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
	data, ok := pkg.Parts[targetPart]
	if !ok {
		return nil, targetPart, fmt.Errorf("missing image part")
	}
	source, err := decodeImage(targetPart, pkg.ContentTypes.ForPart(targetPart), data)
	if err != nil {
		return nil, targetPart, err
	}
	return source, targetPart, nil
}

func pictureUnsupported(slidePart string, element *slideElement, message string) model.SkipItem {
	element.UnsupportedNote = message
	return unsupportedItem(slidePart, unsupportedCode, message)
}

func decodeImage(partName string, contentType string, data []byte) (image.Image, error) {
	extension := strings.ToLower(path.Ext(partName))
	switch {
	case contentType == "image/png" || extension == ".png":
		return decodePNGImage(data)
	case contentType == "image/jpeg" || contentType == "image/jpg" || extension == ".jpg" || extension == ".jpeg":
		return decodeJPEGImage(data)
	case contentType == "image/gif" || extension == ".gif":
		return gif.Decode(bytes.NewReader(data))
	case contentType == "image/svg+xml" || extension == ".svg":
		return decodeSVGImage(data)
	default:
		return nil, fmt.Errorf("unsupported image content type %q", contentType)
	}
}

func decodePNGImage(data []byte) (image.Image, error) {
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	profileData, ok := pngICCProfile(data)
	if !ok {
		return source, nil
	}
	profile, ok := parseICCRGBToSRGBProfile(profileData)
	if !ok {
		return source, nil
	}
	return convertICCImageToSRGB(source, profile), nil
}

func decodeJPEGImage(data []byte) (image.Image, error) {
	source, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if profileData, ok := jpegICCProfile(data); ok {
		if bytes.Contains(profileData, []byte("Adobe RGB (1998)")) || bytes.Contains(profileData, []byte("Adobe RGB")) {
			return convertAdobeRGBImageToSRGB(source), nil
		}
		if profile, ok := parseICCRGBToSRGBProfile(profileData); ok {
			return convertICCImageToSRGB(source, profile), nil
		}
	}
	return source, nil
}

func jpegHasAdobeRGBProfile(data []byte) bool {
	profileData, ok := jpegICCProfile(data)
	return ok && (bytes.Contains(profileData, []byte("Adobe RGB (1998)")) || bytes.Contains(profileData, []byte("Adobe RGB")))
}

func jpegICCProfile(data []byte) ([]byte, bool) {
	const markerPrefix = "ICC_PROFILE\x00"
	chunks := map[int][]byte{}
	totalChunks := 0
	for offset := 0; offset+4 <= len(data); {
		if data[offset] != 0xFF {
			offset++
			continue
		}
		for offset < len(data) && data[offset] == 0xFF {
			offset++
		}
		if offset >= len(data) {
			break
		}
		marker := data[offset]
		offset++
		if marker == 0xDA || marker == 0xD9 {
			break
		}
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}
		if offset+2 > len(data) {
			return nil, false
		}
		length := int(data[offset])<<8 | int(data[offset+1])
		offset += 2
		if length < 2 || offset+length-2 > len(data) {
			return nil, false
		}
		segment := data[offset : offset+length-2]
		offset += length - 2
		if marker != 0xE2 || !bytes.HasPrefix(segment, []byte(markerPrefix)) {
			continue
		}
		if len(segment) < len(markerPrefix)+2 {
			return nil, false
		}
		sequenceNumber := int(segment[len(markerPrefix)])
		sequenceTotal := int(segment[len(markerPrefix)+1])
		if sequenceNumber == 0 || sequenceTotal == 0 || sequenceNumber > sequenceTotal {
			return nil, false
		}
		if totalChunks == 0 {
			totalChunks = sequenceTotal
		} else if totalChunks != sequenceTotal {
			return nil, false
		}
		if _, exists := chunks[sequenceNumber]; exists {
			return nil, false
		}
		chunks[sequenceNumber] = segment[len(markerPrefix)+2:]
	}
	if totalChunks == 0 || len(chunks) != totalChunks {
		return nil, false
	}
	var profile []byte
	for index := 1; index <= totalChunks; index++ {
		chunk, ok := chunks[index]
		if !ok {
			return nil, false
		}
		profile = append(profile, chunk...)
	}
	return profile, true
}

func pngICCProfile(data []byte) ([]byte, bool) {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return nil, false
	}
	offset := 8
	for offset+8 <= len(data) {
		length := int(readUint32BE(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || offset+length+4 > len(data) {
			return nil, false
		}
		chunk := data[offset : offset+length]
		offset += length + 4
		if chunkType == "IEND" {
			return nil, false
		}
		if chunkType != "iCCP" {
			continue
		}
		nameEnd := bytes.IndexByte(chunk, 0)
		if nameEnd < 0 || nameEnd+2 > len(chunk) || chunk[nameEnd+1] != 0 {
			return nil, false
		}
		reader, err := zlib.NewReader(bytes.NewReader(chunk[nameEnd+2:]))
		if err != nil {
			return nil, false
		}
		defer reader.Close()
		profile, err := io.ReadAll(reader)
		if err != nil {
			return nil, false
		}
		return profile, true
	}
	return nil, false
}

func parseICCRGBToSRGBProfile(data []byte) (iccRGBToSRGBProfile, bool) {
	if len(data) < 132 || string(data[16:20]) != "RGB " || string(data[20:24]) != "XYZ " {
		return iccRGBToSRGBProfile{}, false
	}
	tagCount := int(readUint32BE(data[128:132]))
	if tagCount < 0 || 132+tagCount*12 > len(data) {
		return iccRGBToSRGBProfile{}, false
	}
	tags := map[string][]byte{}
	for index := 0; index < tagCount; index++ {
		entry := 132 + index*12
		signature := string(data[entry : entry+4])
		offset := int(readUint32BE(data[entry+4 : entry+8]))
		size := int(readUint32BE(data[entry+8 : entry+12]))
		if offset < 0 || size < 0 || offset+size > len(data) {
			continue
		}
		tags[signature] = data[offset : offset+size]
	}
	profile := iccRGBToSRGBProfile{}
	var ok bool
	if profile.rXYZ, ok = parseICCXYZTag(tags["rXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.gXYZ, ok = parseICCXYZTag(tags["gXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.bXYZ, ok = parseICCXYZTag(tags["bXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.rTRC, ok = parseICCCurveTag(tags["rTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.gTRC, ok = parseICCCurveTag(tags["gTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.bTRC, ok = parseICCCurveTag(tags["bTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	return profile, true
}

func parseICCXYZTag(data []byte) ([3]float64, bool) {
	if len(data) < 20 || string(data[:4]) != "XYZ " {
		return [3]float64{}, false
	}
	return [3]float64{
		s15Fixed16(data[8:12]),
		s15Fixed16(data[12:16]),
		s15Fixed16(data[16:20]),
	}, true
}

func parseICCCurveTag(data []byte) (iccCurve, bool) {
	if len(data) < 12 || string(data[:4]) != "curv" {
		return iccCurve{}, false
	}
	count := int(readUint32BE(data[8:12]))
	if count == 0 {
		return iccCurve{gamma: 1}, true
	}
	if len(data) < 12+count*2 {
		return iccCurve{}, false
	}
	if count == 1 {
		return iccCurve{gamma: float64(readUint16BE(data[12:14])) / 256}, true
	}
	table := make([]uint16, count)
	for index := range table {
		table[index] = readUint16BE(data[12+index*2 : 14+index*2])
	}
	return iccCurve{table: table}, true
}

func convertICCImageToSRGB(source image.Image, profile iccRGBToSRGBProfile) *image.RGBA {
	bounds := source.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			r, g, b := profile.iccRGBToSRGB(pixel.R, pixel.G, pixel.B)
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: pixel.A})
		}
	}
	return dst
}

func (profile iccRGBToSRGBProfile) iccRGBToSRGB(r uint8, g uint8, b uint8) (uint8, uint8, uint8) {
	linearR := profile.rTRC.linearize(r)
	linearG := profile.gTRC.linearize(g)
	linearB := profile.bTRC.linearize(b)

	xD50 := profile.rXYZ[0]*linearR + profile.gXYZ[0]*linearG + profile.bXYZ[0]*linearB
	yD50 := profile.rXYZ[1]*linearR + profile.gXYZ[1]*linearG + profile.bXYZ[1]*linearB
	zD50 := profile.rXYZ[2]*linearR + profile.gXYZ[2]*linearG + profile.bXYZ[2]*linearB

	// ICC matrix profiles encode PCS XYZ relative to D50. Adapt to D65 before
	// applying the sRGB output matrix.
	xD65 := 0.9555766*xD50 - 0.0230393*yD50 + 0.0631636*zD50
	yD65 := -0.0282895*xD50 + 1.0099416*yD50 + 0.0210077*zD50
	zD65 := 0.0122982*xD50 - 0.0204830*yD50 + 1.3299098*zD50

	srgbR := 3.2404542*xD65 - 1.5371385*yD65 - 0.4985314*zD65
	srgbG := -0.9692660*xD65 + 1.8760108*yD65 + 0.0415560*zD65
	srgbB := 0.0556434*xD65 - 0.2040259*yD65 + 1.0572252*zD65
	return linearToSRGBByte(srgbR), linearToSRGBByte(srgbG), linearToSRGBByte(srgbB)
}

func (curve iccCurve) linearize(value uint8) float64 {
	encoded := float64(value) / 255
	if len(curve.table) == 0 {
		gamma := curve.gamma
		if gamma == 0 {
			gamma = 1
		}
		return math.Pow(encoded, gamma)
	}
	position := encoded * float64(len(curve.table)-1)
	index := int(math.Floor(position))
	if index >= len(curve.table)-1 {
		return float64(curve.table[len(curve.table)-1]) / 65535
	}
	fraction := position - float64(index)
	a := float64(curve.table[index]) / 65535
	b := float64(curve.table[index+1]) / 65535
	return a + (b-a)*fraction
}

func readUint32BE(data []byte) uint32 {
	if len(data) < 4 {
		return 0
	}
	return uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
}

func readUint16BE(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return uint16(data[0])<<8 | uint16(data[1])
}

func s15Fixed16(data []byte) float64 {
	if len(data) < 4 {
		return 0
	}
	value := int32(readUint32BE(data))
	return float64(value) / 65536
}

func convertAdobeRGBImageToSRGB(source image.Image) *image.RGBA {
	bounds := source.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.RGBAModel.Convert(source.At(x, y)).(color.RGBA)
			pixel.R, pixel.G, pixel.B = adobeRGBToSRGB(pixel.R, pixel.G, pixel.B)
			dst.SetRGBA(x, y, pixel)
		}
	}
	return dst
}

func adobeRGBToSRGB(r uint8, g uint8, b uint8) (uint8, uint8, uint8) {
	linearR := adobeRGBByteToLinear(r)
	linearG := adobeRGBByteToLinear(g)
	linearB := adobeRGBByteToLinear(b)

	x := 0.5767309*linearR + 0.1855540*linearG + 0.1881852*linearB
	y := 0.2973769*linearR + 0.6273491*linearG + 0.0752741*linearB
	z := 0.0270343*linearR + 0.0706872*linearG + 0.9911085*linearB

	srgbR := 3.2404542*x - 1.5371385*y - 0.4985314*z
	srgbG := -0.9692660*x + 1.8760108*y + 0.0415560*z
	srgbB := 0.0556434*x - 0.2040259*y + 1.0572252*z
	return linearToSRGBByte(srgbR), linearToSRGBByte(srgbG), linearToSRGBByte(srgbB)
}

func adobeRGBByteToLinear(value uint8) float64 {
	if value == 0 {
		return 0
	}
	return math.Pow(float64(value)/255, 2.19921875)
}

func decodeSVGImage(data []byte) (image.Image, error) {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil, err
	}
	if root.Name != "svg" {
		return nil, fmt.Errorf("expected svg root, got %q", root.Name)
	}
	viewBox, err := parseSVGViewBox(root)
	if err != nil {
		return nil, err
	}
	width := svgRasterDimension(viewBox.Width)
	height := svgRasterDimension(viewBox.Height)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := drawSVGNode(img, img.Bounds(), viewBox, root, parseSVGStyleRules(root), svgPaintStyle{}); err != nil {
		return nil, err
	}
	return img, nil
}

func parseSVGViewBox(root *xmlNode) (svgViewBox, error) {
	raw := attrValue(root.Attrs, "viewBox")
	values := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(values) == 4 {
		var parsed [4]float64
		for index, value := range values {
			number, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return svgViewBox{}, fmt.Errorf("invalid svg viewBox %q", raw)
			}
			parsed[index] = number
		}
		if parsed[2] > 0 && parsed[3] > 0 {
			return svgViewBox{MinX: parsed[0], MinY: parsed[1], Width: parsed[2], Height: parsed[3]}, nil
		}
	}
	width, widthOK := svgLengthAttr(root.Attrs, "width")
	height, heightOK := svgLengthAttr(root.Attrs, "height")
	if widthOK && heightOK && width > 0 && height > 0 {
		return svgViewBox{Width: width, Height: height}, nil
	}
	return svgViewBox{}, fmt.Errorf("svg viewBox is missing or invalid")
}

func svgLengthAttr(attrs []xml.Attr, name string) (float64, bool) {
	value := attrValue(attrs, name)
	value = strings.TrimSuffix(strings.TrimSpace(value), "px")
	if value == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(value, 64)
	return number, err == nil
}

func svgRasterDimension(value float64) int {
	dimension := int(math.Round(value))
	if dimension < 1 {
		return 1
	}
	if dimension > 2048 {
		return 2048
	}
	return dimension
}

func drawSVGNode(img *image.RGBA, bounds image.Rectangle, viewBox svgViewBox, node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle) error {
	inherited = resolveSVGPaintStyle(node, styles, inherited, true)
	for _, child := range node.Children {
		switch child.Name {
		case "g", "svg":
			if err := drawSVGNode(img, bounds, viewBox, child, styles, inherited); err != nil {
				return err
			}
		case "path":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			paths, err := parseSVGPath(attrValue(child.Attrs, "d"), viewBox)
			if err != nil {
				return err
			}
			for _, points := range paths {
				drawPolygon(img, bounds, points, c)
			}
		case "rect":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			rect, ok := svgRectBounds(child, bounds, viewBox)
			if ok {
				draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
			}
		case "circle", "ellipse":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			ellipse, ok := svgEllipseBounds(child, bounds, viewBox)
			if ok {
				drawEllipse(img, ellipse, c)
			}
		}
	}
	return nil
}

func svgNodeFill(node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle) (color.RGBA, bool) {
	style := resolveSVGPaintStyle(node, styles, inherited, true)
	if style.NoFill {
		return color.RGBA{}, false
	}
	if !style.HasFill {
		style.Fill = color.RGBA{A: 255}
	}
	if style.HasOpacity {
		opacity := style.FillOpacity
		if opacity < 0 {
			opacity = 0
		}
		if opacity > 1 {
			opacity = 1
		}
		style.Fill.A = uint8(math.Round(float64(style.Fill.A) * opacity))
	}
	return style.Fill, true
}

func resolveSVGPaintStyle(node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle, includePresentationAttrs bool) svgPaintStyle {
	resolved := inherited
	if includePresentationAttrs {
		mergeSVGPaintStyle(&resolved, parseSVGPaintDeclarations("fill:"+attrValue(node.Attrs, "fill")+";fill-opacity:"+attrValue(node.Attrs, "fill-opacity")))
	}
	for _, className := range strings.Fields(attrValue(node.Attrs, "class")) {
		if style, ok := styles[className]; ok {
			mergeSVGPaintStyle(&resolved, style)
		}
	}
	mergeSVGPaintStyle(&resolved, parseSVGPaintDeclarations(attrValue(node.Attrs, "style")))
	return resolved
}

func mergeSVGPaintStyle(base *svgPaintStyle, override svgPaintStyle) {
	if override.HasFill || override.NoFill {
		base.Fill = override.Fill
		base.HasFill = override.HasFill
		base.NoFill = override.NoFill
	}
	if override.HasOpacity {
		base.FillOpacity = override.FillOpacity
		base.HasOpacity = true
	}
}

func parseSVGStyleRules(root *xmlNode) map[string]svgPaintStyle {
	styles := map[string]svgPaintStyle{}
	for _, node := range descendantsByName(root, "style") {
		for _, block := range strings.Split(node.Text, "}") {
			selectorText, declarationText, ok := strings.Cut(block, "{")
			if !ok {
				continue
			}
			style := parseSVGPaintDeclarations(declarationText)
			if !style.HasFill && !style.NoFill && !style.HasOpacity {
				continue
			}
			for _, selector := range strings.Split(selectorText, ",") {
				selector = strings.TrimSpace(selector)
				if !strings.HasPrefix(selector, ".") {
					continue
				}
				className := strings.TrimSpace(strings.TrimPrefix(selector, "."))
				if className != "" {
					styles[className] = style
				}
			}
		}
	}
	return styles
}

func parseSVGPaintDeclarations(raw string) svgPaintStyle {
	var style svgPaintStyle
	for _, declaration := range strings.Split(raw, ";") {
		name, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		switch name {
		case "fill":
			c, hasFill, noFill := parseSVGFillValue(value)
			style.Fill = c
			style.HasFill = hasFill
			style.NoFill = noFill
		case "fill-opacity":
			if opacity, ok := parseSVGOpacity(value); ok {
				style.FillOpacity = opacity
				style.HasOpacity = true
			}
		}
	}
	return style
}

func parseSVGFillValue(raw string) (color.RGBA, bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return color.RGBA{}, false, false
	}
	if strings.EqualFold(raw, "none") {
		return color.RGBA{}, false, true
	}
	var c color.RGBA
	var ok bool
	switch strings.ToLower(raw) {
	case "black":
		c, ok = color.RGBA{A: 255}, true
	case "white":
		c, ok = color.RGBA{R: 255, G: 255, B: 255, A: 255}, true
	default:
		c, ok = parseHexColor(raw)
	}
	if !ok {
		return color.RGBA{}, false, false
	}
	return c, true, false
}

func parseSVGOpacity(raw string) (float64, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return value, err == nil
}

func svgRectBounds(node *xmlNode, bounds image.Rectangle, viewBox svgViewBox) (image.Rectangle, bool) {
	x, xOK := svgFloatAttr(node.Attrs, "x")
	y, yOK := svgFloatAttr(node.Attrs, "y")
	width, widthOK := svgFloatAttr(node.Attrs, "width")
	height, heightOK := svgFloatAttr(node.Attrs, "height")
	if !xOK {
		x = 0
	}
	if !yOK {
		y = 0
	}
	if !widthOK || !heightOK || width <= 0 || height <= 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(
		svgCoordToPixel(x, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(y, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
		svgCoordToPixel(x+width, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(y+height, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
	).Intersect(bounds), true
}

func svgEllipseBounds(node *xmlNode, bounds image.Rectangle, viewBox svgViewBox) (image.Rectangle, bool) {
	cx, cxOK := svgFloatAttr(node.Attrs, "cx")
	cy, cyOK := svgFloatAttr(node.Attrs, "cy")
	if !cxOK || !cyOK {
		return image.Rectangle{}, false
	}
	rx, rxOK := svgFloatAttr(node.Attrs, "rx")
	ry, ryOK := svgFloatAttr(node.Attrs, "ry")
	if node.Name == "circle" {
		r, rOK := svgFloatAttr(node.Attrs, "r")
		rx, ry, rxOK, ryOK = r, r, rOK, rOK
	}
	if !rxOK || !ryOK || rx <= 0 || ry <= 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(
		svgCoordToPixel(cx-rx, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(cy-ry, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
		svgCoordToPixel(cx+rx, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(cy+ry, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
	).Intersect(bounds), true
}

func svgFloatAttr(attrs []xml.Attr, name string) (float64, bool) {
	value := strings.TrimSpace(attrValue(attrs, name))
	if value == "" {
		return 0, false
	}
	value = strings.TrimSuffix(value, "px")
	number, err := strconv.ParseFloat(value, 64)
	return number, err == nil
}

func svgCoordToPixel(value float64, min float64, span float64, pixelMin int, pixelSpan int) int {
	if span == 0 {
		return pixelMin
	}
	return pixelMin + int(math.Round((value-min)/span*float64(pixelSpan)))
}

func svgPointToPathPoint(x float64, y float64, viewBox svgViewBox) pathPoint {
	return pathPoint{
		X: (x - viewBox.MinX) / viewBox.Width,
		Y: (y - viewBox.MinY) / viewBox.Height,
	}
}

func parseSVGPath(data string, viewBox svgViewBox) ([][]pathPoint, error) {
	tokens, err := tokenizeSVGPath(data)
	if err != nil {
		return nil, err
	}
	var paths [][]pathPoint
	var points []pathPoint
	var currentCommand byte
	var currentX float64
	var currentY float64
	var startX float64
	var startY float64
	index := 0
	for index < len(tokens) {
		if !tokens[index].IsNumber {
			currentCommand = tokens[index].Command
			index++
		} else if currentCommand == 0 {
			return nil, fmt.Errorf("svg path data starts with a number")
		}
		switch currentCommand {
		case 'M', 'm':
			first := true
			for index < len(tokens) && tokens[index].IsNumber {
				x, y, next, ok := readSVGPathPair(tokens, index)
				if !ok {
					return nil, fmt.Errorf("svg path move command has incomplete coordinates")
				}
				index = next
				if currentCommand == 'm' {
					x += currentX
					y += currentY
				}
				if first {
					if len(points) >= 3 {
						paths = append(paths, points)
					}
					points = []pathPoint{svgPointToPathPoint(x, y, viewBox)}
					startX, startY = x, y
					first = false
				} else {
					points = append(points, svgPointToPathPoint(x, y, viewBox))
				}
				currentX, currentY = x, y
			}
		case 'L', 'l':
			for index < len(tokens) && tokens[index].IsNumber {
				x, y, next, ok := readSVGPathPair(tokens, index)
				if !ok {
					return nil, fmt.Errorf("svg path line command has incomplete coordinates")
				}
				index = next
				if currentCommand == 'l' {
					x += currentX
					y += currentY
				}
				points = append(points, svgPointToPathPoint(x, y, viewBox))
				currentX, currentY = x, y
			}
		case 'H', 'h':
			for index < len(tokens) && tokens[index].IsNumber {
				x := tokens[index].Number
				index++
				if currentCommand == 'h' {
					x += currentX
				}
				points = append(points, svgPointToPathPoint(x, currentY, viewBox))
				currentX = x
			}
		case 'V', 'v':
			for index < len(tokens) && tokens[index].IsNumber {
				y := tokens[index].Number
				index++
				if currentCommand == 'v' {
					y += currentY
				}
				points = append(points, svgPointToPathPoint(currentX, y, viewBox))
				currentY = y
			}
		case 'C', 'c':
			for index < len(tokens) && tokens[index].IsNumber {
				values, next, ok := readSVGPathNumbers(tokens, index, 6)
				if !ok {
					return nil, fmt.Errorf("svg path cubic command has incomplete coordinates")
				}
				index = next
				x1, y1, x2, y2, x, y := values[0], values[1], values[2], values[3], values[4], values[5]
				if currentCommand == 'c' {
					x1 += currentX
					y1 += currentY
					x2 += currentX
					y2 += currentY
					x += currentX
					y += currentY
				}
				points = append(points, flattenSVGCubic(currentX, currentY, x1, y1, x2, y2, x, y, viewBox)...)
				currentX, currentY = x, y
			}
		case 'Z', 'z':
			if len(points) >= 3 {
				paths = append(paths, points)
			}
			points = nil
			currentX, currentY = startX, startY
			currentCommand = 0
		default:
			return nil, fmt.Errorf("unsupported svg path command %q", string(currentCommand))
		}
	}
	if len(points) >= 3 {
		paths = append(paths, points)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("svg path has no closed paintable subpaths")
	}
	return paths, nil
}

func tokenizeSVGPath(data string) ([]svgPathToken, error) {
	var tokens []svgPathToken
	for index := 0; index < len(data); {
		ch := data[index]
		switch {
		case isSVGPathSeparator(ch):
			index++
		case isSVGPathCommand(ch):
			tokens = append(tokens, svgPathToken{Command: ch})
			index++
		case isSVGPathNumberStart(ch):
			start := index
			index++
			for index < len(data) && isSVGPathNumberByte(data[index], data[index-1]) {
				index++
			}
			number, err := strconv.ParseFloat(data[start:index], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid svg path number %q", data[start:index])
			}
			tokens = append(tokens, svgPathToken{Number: number, IsNumber: true})
		default:
			return nil, fmt.Errorf("invalid svg path token %q", string(ch))
		}
	}
	return tokens, nil
}

func isSVGPathSeparator(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ','
}

func isSVGPathCommand(ch byte) bool {
	return strings.ContainsRune("MmLlHhVvCcZz", rune(ch))
}

func isSVGPathNumberStart(ch byte) bool {
	return (ch >= '0' && ch <= '9') || ch == '-' || ch == '+' || ch == '.'
}

func isSVGPathNumberByte(ch byte, previous byte) bool {
	if ch >= '0' && ch <= '9' {
		return true
	}
	if ch == '.' {
		return true
	}
	if ch == '-' || ch == '+' {
		return previous == 'e' || previous == 'E'
	}
	return ch == 'e' || ch == 'E'
}

func readSVGPathPair(tokens []svgPathToken, index int) (float64, float64, int, bool) {
	values, next, ok := readSVGPathNumbers(tokens, index, 2)
	if !ok {
		return 0, 0, index, false
	}
	return values[0], values[1], next, true
}

func readSVGPathNumbers(tokens []svgPathToken, index int, count int) ([]float64, int, bool) {
	if index+count > len(tokens) {
		return nil, index, false
	}
	values := make([]float64, 0, count)
	for offset := 0; offset < count; offset++ {
		token := tokens[index+offset]
		if !token.IsNumber {
			return nil, index, false
		}
		values = append(values, token.Number)
	}
	return values, index + count, true
}

func flattenSVGCubic(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, viewBox svgViewBox) []pathPoint {
	const segments = 12
	points := make([]pathPoint, 0, segments)
	for step := 1; step <= segments; step++ {
		t := float64(step) / segments
		inv := 1 - t
		x := inv*inv*inv*x0 + 3*inv*inv*t*x1 + 3*inv*t*t*x2 + t*t*t*x3
		y := inv*inv*inv*y0 + 3*inv*inv*t*y1 + 3*inv*t*t*y2 + t*t*t*y3
		points = append(points, svgPointToPathPoint(x, y, viewBox))
	}
	return points
}

func scaleEMU(value int64, totalEMU int64, totalPixels int) int {
	if totalEMU == 0 {
		return 0
	}
	return int(math.Round(float64(value) / float64(totalEMU) * float64(totalPixels)))
}

func scaleEMUFloat(value int64, totalEMU int64, totalPixels int) float64 {
	if totalEMU == 0 {
		return 0
	}
	return float64(value) / float64(totalEMU) * float64(totalPixels)
}

func sourceCropRect(bounds image.Rectangle, element slideElement) image.Rectangle {
	if !element.HasCrop {
		return bounds
	}
	width := bounds.Dx()
	height := bounds.Dy()
	left := bounds.Min.X + cropPixels(width, element.CropLeft)
	top := bounds.Min.Y + cropPixels(height, element.CropTop)
	right := bounds.Max.X - cropPixels(width, element.CropRight)
	bottom := bounds.Max.Y - cropPixels(height, element.CropBottom)
	cropped := image.Rect(left, top, right, bottom)
	if cropped.Empty() || cropped.Intersect(bounds).Empty() {
		return bounds
	}
	return cropped
}

func cropPixels(total int, percentage int64) int {
	if percentage == 0 || total == 0 {
		return 0
	}
	return int(math.Round(float64(total) * float64(percentage) / 100000))
}

func pictureSourceForElement(src image.Image, element slideElement) (image.Image, image.Rectangle) {
	srcBounds := sourceCropRect(src.Bounds(), element)
	if !element.FlipH && !element.FlipV && !shouldApplyImageAlphaModFix(element) {
		return src, srcBounds
	}
	return transformedPictureImage(src, srcBounds, element), image.Rect(0, 0, srcBounds.Dx(), srcBounds.Dy())
}

func shouldApplyImageAlphaModFix(element slideElement) bool {
	return element.HasImageAlphaModFix && element.ImageAlphaModFixPct > 0 && element.ImageAlphaModFixPct != 100000
}

func transformedPictureImage(src image.Image, srcBounds image.Rectangle, element slideElement) *image.RGBA {
	width := srcBounds.Dx()
	height := srcBounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + y
		if element.FlipV {
			srcY = srcBounds.Max.Y - 1 - y
		}
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x
			if element.FlipH {
				srcX = srcBounds.Max.X - 1 - x
			}
			pixel := color.RGBAModel.Convert(src.At(srcX, srcY)).(color.RGBA)
			pixel = applyImageAlphaModFix(pixel, element)
			dst.SetRGBA(x, y, pixel)
		}
	}
	return dst
}

func applyImageAlphaModFix(c color.RGBA, element slideElement) color.RGBA {
	if shouldApplyImageAlphaModFix(element) {
		c.A = scaleColorChannel(c.A, element.ImageAlphaModFixPct)
	}
	return c
}

func scaleImage(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() {
		return
	}
	if srcBounds.Empty() {
		return
	}
	pictureScaler(src, srcBounds).Scale(dst, target, src, srcBounds, xdraw.Over, nil)
}

func pictureScaler(src image.Image, srcBounds image.Rectangle) xdraw.Scaler {
	if _, ok := src.(*image.YCbCr); ok && srcBounds.In(src.Bounds()) {
		return xdraw.CatmullRom
	}
	return xdraw.ApproxBiLinear
}

func scaleImageWithSoftEdge(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle, radius int) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() {
		return
	}
	if srcBounds.Empty() {
		return
	}
	if radius <= 0 {
		scaleImage(dst, target, src, srcBounds)
		return
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	pictureScaler(src, srcBounds).Scale(layer, layer.Bounds(), src, srcBounds, xdraw.Over, nil)
	applySoftEdgeAlpha(layer, radius)
	for y := 0; y < layer.Bounds().Dy(); y++ {
		for x := 0; x < layer.Bounds().Dx(); x++ {
			blendPixel(dst, target.Min.X+x, target.Min.Y+y, layer.RGBAAt(x, y))
		}
	}
}

func applySoftEdgeAlpha(img *image.RGBA, radius int) {
	bounds := img.Bounds()
	if radius <= 0 || bounds.Empty() {
		return
	}
	maxRadius := min(radius, min(bounds.Dx(), bounds.Dy())/2)
	if maxRadius <= 0 {
		return
	}
	padding := maxRadius * 3
	maskWidth := bounds.Dx() + padding*2
	maskHeight := bounds.Dy() + padding*2
	mask := make([]uint8, maskWidth*maskHeight)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			mask[(y+padding)*maskWidth+x+padding] = img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y).A
		}
	}
	blurred := gaussianBlurAlpha(mask, maskWidth, maskHeight, maxRadius)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			pixel := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			pixel.A = blurred[(y+padding)*maskWidth+x+padding]
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, pixel)
		}
	}
}

func softEdgeRadiusPixels(element slideElement, size slideSize, outputWidth int) int {
	radius := scaleEMU(element.SoftEdgeRadius, size.CX, outputWidth)
	if radius < 0 {
		return 0
	}
	return radius
}

func scaleImageWithCustomMask(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle, points []pathPoint, commands []pathCommand) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() || len(points) < 3 {
		return
	}
	if srcBounds.Empty() {
		return
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	pictureScaler(src, srcBounds).Scale(layer, layer.Bounds(), src, srcBounds, xdraw.Over, nil)
	mask := rasterizePathMaskWithCommands(layer.Bounds(), points, commands)
	draw.DrawMask(dst, target, layer, image.Point{}, mask, image.Point{}, draw.Over)
}

func rasterizePathMask(bounds image.Rectangle, points []pathPoint) *image.Alpha {
	return rasterizePathMaskWithCommands(bounds, points, nil)
}

func rasterizePathMaskWithCommands(bounds image.Rectangle, points []pathPoint, commands []pathCommand) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if bounds.Empty() || len(points) < 3 {
		return mask
	}
	rasterizer := vector.NewRasterizer(bounds.Dx(), bounds.Dy())
	if len(commands) > 0 {
		for _, command := range commands {
			switch command.Kind {
			case "moveTo":
				if len(command.Points) == 1 {
					rasterizer.MoveTo(maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds))
				}
			case "lnTo":
				if len(command.Points) == 1 {
					rasterizer.LineTo(maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds))
				}
			case "cubicBezTo":
				if len(command.Points) == 3 {
					rasterizer.CubeTo(
						maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds),
						maskPathX(command.Points[1], bounds), maskPathY(command.Points[1], bounds),
						maskPathX(command.Points[2], bounds), maskPathY(command.Points[2], bounds),
					)
				}
			case "close":
				rasterizer.ClosePath()
			}
		}
		rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
		return mask
	}
	for index, point := range points {
		x := maskPathX(point, bounds)
		y := maskPathY(point, bounds)
		if index == 0 {
			rasterizer.MoveTo(x, y)
		} else {
			rasterizer.LineTo(x, y)
		}
	}
	rasterizer.ClosePath()
	rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	return mask
}

func maskPathX(point pathPoint, bounds image.Rectangle) float32 {
	return float32(point.X * float64(bounds.Dx()))
}

func maskPathY(point pathPoint, bounds image.Rectangle) float32 {
	return float32(point.Y * float64(bounds.Dy()))
}
