package render

import (
	"image/color"
	"math"
	"strconv"
)

func colorFromSolidFill(node *xmlNode) (color.RGBA, bool) {
	return colorFromSolidFillWithTheme(node, defaultThemeColors())
}

func colorFromSolidFillWithTheme(node *xmlNode, theme themeColors) (color.RGBA, bool) {
	return colorFromColorNodeWithTheme(node, theme)
}

func colorFromColorNode(node *xmlNode) (color.RGBA, bool) {
	return colorFromColorNodeWithTheme(node, defaultThemeColors())
}

func colorFromColorNodeWithTheme(node *xmlNode, theme themeColors) (color.RGBA, bool) {
	if srgb := firstChild(node, "srgbClr"); srgb != nil {
		if c, ok := parseHexColor(attrValue(srgb.Attrs, "val")); ok {
			return applyColorModifiers(c, srgb), true
		}
	}
	if scrgb := firstChild(node, "scrgbClr"); scrgb != nil {
		if c, ok := parseScRGBColor(scrgb); ok {
			return applyColorModifiers(c, scrgb), true
		}
	}
	if hsl := firstChild(node, "hslClr"); hsl != nil {
		if c, ok := parseHSLColor(hsl); ok {
			return applyColorModifiers(c, hsl), true
		}
	}
	if scheme := firstChild(node, "schemeClr"); scheme != nil {
		if c, ok := schemeColorWithTheme(attrValue(scheme.Attrs, "val"), theme); ok {
			return applyColorModifiers(c, scheme), true
		}
	}
	if sys := firstChild(node, "sysClr"); sys != nil {
		if c, ok := parseHexColor(attrValue(sys.Attrs, "lastClr")); ok {
			return applyColorModifiers(c, sys), true
		}
		if c, ok := systemColor(attrValue(sys.Attrs, "val")); ok {
			return applyColorModifiers(c, sys), true
		}
	}
	if preset := firstChild(node, "prstClr"); preset != nil {
		if c, ok := presetColor(attrValue(preset.Attrs, "val")); ok {
			return applyColorModifiers(c, preset), true
		}
	}
	return color.RGBA{}, false
}

func presetColor(value string) (color.RGBA, bool) {
	switch value {
	case "aliceBlue":
		return color.RGBA{R: 0xf0, G: 0xf8, B: 0xff, A: 255}, true
	case "blue":
		return color.RGBA{B: 255, A: 255}, true
	case "black":
		return color.RGBA{A: 255}, true
	case "cyan":
		return color.RGBA{G: 255, B: 255, A: 255}, true
	case "dkBlue":
		return color.RGBA{B: 0x8b, A: 255}, true
	case "dkCyan":
		return color.RGBA{G: 0x8b, B: 0x8b, A: 255}, true
	case "dkGray":
		return color.RGBA{R: 0xa9, G: 0xa9, B: 0xa9, A: 255}, true
	case "dkGreen":
		return color.RGBA{G: 0x64, A: 255}, true
	case "dkMagenta":
		return color.RGBA{R: 0x8b, B: 0x8b, A: 255}, true
	case "dkRed":
		return color.RGBA{R: 0x8b, A: 255}, true
	case "dkYellow":
		return color.RGBA{R: 0x80, G: 0x80, A: 255}, true
	case "gray":
		return color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 255}, true
	case "green":
		return color.RGBA{G: 0x80, A: 255}, true
	case "ltGray":
		return color.RGBA{R: 0xd3, G: 0xd3, B: 0xd3, A: 255}, true
	case "magenta":
		return color.RGBA{R: 255, B: 255, A: 255}, true
	case "red":
		return color.RGBA{R: 255, A: 255}, true
	case "white":
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}, true
	case "yellow":
		return color.RGBA{R: 255, G: 255, A: 255}, true
	default:
		return color.RGBA{}, false
	}
}

func systemColor(value string) (color.RGBA, bool) {
	switch value {
	case "windowText", "menuText", "captionText", "activeCaptionText", "btnText":
		return color.RGBA{A: 255}, true
	case "window", "menu", "activeCaption", "btnFace":
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}, true
	case "grayText", "scrollBar", "inactiveCaption":
		return color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 255}, true
	case "highlight":
		return color.RGBA{R: 0x33, G: 0x99, B: 0xff, A: 255}, true
	case "highlightText":
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}, true
	default:
		return color.RGBA{}, false
	}
}

func parseScRGBColor(node *xmlNode) (color.RGBA, bool) {
	r, okR := parseScRGBLinearAttr(node, "r")
	g, okG := parseScRGBLinearAttr(node, "g")
	b, okB := parseScRGBLinearAttr(node, "b")
	if !okR || !okG || !okB {
		return color.RGBA{}, false
	}
	return color.RGBA{
		R: linearToSRGBByte(r),
		G: linearToSRGBByte(g),
		B: linearToSRGBByte(b),
		A: 255,
	}, true
}

func parseScRGBLinearAttr(node *xmlNode, name string) (float64, bool) {
	raw := attrValue(node.Attrs, name)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return clampFloat(float64(value)/100000, 0, 1), true
}

func parseHSLColor(node *xmlNode) (color.RGBA, bool) {
	if attrValue(node.Attrs, "hue") == "" || attrValue(node.Attrs, "sat") == "" || attrValue(node.Attrs, "lum") == "" {
		return color.RGBA{}, false
	}
	h := math.Mod(float64(parseIntAttr(node.Attrs, "hue"))/60000, 360)
	if h < 0 {
		h += 360
	}
	s := clampFloat(float64(parsePercentAttr(node.Attrs, "sat"))/100000, 0, 1)
	l := clampFloat(float64(parsePercentAttr(node.Attrs, "lum"))/100000, 0, 1)
	r, g, b := hslToRGB(h, s, l)
	return color.RGBA{R: r, G: g, B: b, A: 255}, true
}

func applyColorModifiers(c color.RGBA, node *xmlNode) color.RGBA {
	pendingLumMod := int64(100000)
	pendingLumOff := int64(0)
	hasPendingLuminance := false
	flushLuminance := func() {
		if !hasPendingLuminance {
			return
		}
		c = applyLuminanceModifier(c, pendingLumMod, pendingLumOff)
		pendingLumMod = 100000
		pendingLumOff = 0
		hasPendingLuminance = false
	}
	for _, child := range node.Children {
		switch child.Name {
		case "lumMod":
			if hasPendingLuminance && pendingLumOff != 0 {
				flushLuminance()
			}
			pendingLumMod = pendingLumMod * parsePercentAttr(child.Attrs, "val") / 100000
			hasPendingLuminance = true
		case "shade":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applyShadeModifier(c, value)
		case "lumOff":
			pendingLumOff += parsePercentAttr(child.Attrs, "val")
			hasPendingLuminance = true
		case "alpha":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c.A = colorChannelFromPercent(value)
		case "alphaMod":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c.A = scaleColorChannel(c.A, value)
		case "alphaOff":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c.A = offsetColorChannel(c.A, value)
		case "hue":
			flushLuminance()
			value := parseIntAttr(child.Attrs, "val")
			c = applyHue(c, value)
		case "hueOff":
			flushLuminance()
			value := parseIntAttr(child.Attrs, "val")
			c = applyHueOffset(c, value)
		case "hueMod":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applyHueModifier(c, value)
		case "sat":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applySaturation(c, value)
		case "tint":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applyTintModifier(c, value)
		case "satMod":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applySaturationModifier(c, value)
		case "satOff":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applySaturationOffset(c, value)
		case "lum":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applyLuminance(c, value)
		case "red":
			flushLuminance()
			c.R = colorChannelFromPercent(parsePercentAttr(child.Attrs, "val"))
		case "redMod":
			flushLuminance()
			c.R = scaleColorChannel(c.R, parsePercentAttr(child.Attrs, "val"))
		case "redOff":
			flushLuminance()
			c.R = offsetColorChannel(c.R, parsePercentAttr(child.Attrs, "val"))
		case "green":
			flushLuminance()
			c.G = colorChannelFromPercent(parsePercentAttr(child.Attrs, "val"))
		case "greenMod":
			flushLuminance()
			c.G = scaleColorChannel(c.G, parsePercentAttr(child.Attrs, "val"))
		case "greenOff":
			flushLuminance()
			c.G = offsetColorChannel(c.G, parsePercentAttr(child.Attrs, "val"))
		case "blue":
			flushLuminance()
			c.B = colorChannelFromPercent(parsePercentAttr(child.Attrs, "val"))
		case "blueMod":
			flushLuminance()
			c.B = scaleColorChannel(c.B, parsePercentAttr(child.Attrs, "val"))
		case "blueOff":
			flushLuminance()
			c.B = offsetColorChannel(c.B, parsePercentAttr(child.Attrs, "val"))
		case "gray":
			flushLuminance()
			c = applyGrayscale(c)
		case "inv":
			flushLuminance()
			c.R = 255 - c.R
			c.G = 255 - c.G
			c.B = 255 - c.B
		case "comp":
			flushLuminance()
			c = applyHueOffset(c, 10800000)
		case "gamma":
			flushLuminance()
			c = applyGammaTransform(c)
		case "invGamma":
			flushLuminance()
			c = applyInverseGammaTransform(c)
		}
	}
	flushLuminance()
	return c
}

func colorChannelFromPercent(value int64) uint8 {
	return scaleColorChannel(255, value)
}

func applyLuminanceModifier(c color.RGBA, mod int64, off int64) color.RGBA {
	if mod == 0 && off == 0 {
		c.R = 0
		c.G = 0
		c.B = 0
		return c
	}
	h, s, l := rgbToHSL(c)
	l = l*float64(mod)/100000 + float64(off)/100000
	if l < 0 {
		l = 0
	} else if l > 1 {
		l = 1
	}
	c.R, c.G, c.B = hslToRGB(h, s, l)
	return c
}

func scaleColorChannel(channel uint8, value int64) uint8 {
	return clampColor(int64(channel) * value / 100000)
}

func offsetColorChannel(channel uint8, value int64) uint8 {
	return clampColor(int64(math.Round(float64(channel) + float64(value)*255/100000)))
}

func applyTintModifier(c color.RGBA, value int64) color.RGBA {
	c.R = blendSRGBChannelLinear(c.R, 255, value)
	c.G = blendSRGBChannelLinear(c.G, 255, value)
	c.B = blendSRGBChannelLinear(c.B, 255, value)
	return c
}

func applyLuminance(c color.RGBA, value int64) color.RGBA {
	h, s, _ := rgbToHSL(c)
	l := clampFloat(float64(value)/100000, 0, 1)
	c.R, c.G, c.B = hslToRGB(h, s, l)
	return c
}

func applyShadeModifier(c color.RGBA, value int64) color.RGBA {
	c.R = blendSRGBChannelLinear(c.R, 0, value)
	c.G = blendSRGBChannelLinear(c.G, 0, value)
	c.B = blendSRGBChannelLinear(c.B, 0, value)
	return c
}

func blendSRGBChannelLinear(channel uint8, target uint8, value int64) uint8 {
	if value < 0 {
		value = 0
	} else if value > 100000 {
		value = 100000
	}
	t := float64(value) / 100000
	linear := srgbByteToLinear(channel)*t + srgbByteToLinear(target)*(1-t)
	return linearToSRGBByte(linear)
}

func applySaturationModifier(c color.RGBA, value int64) color.RGBA {
	if value == 100000 {
		return c
	}
	h, s, l := rgbToHSL(c)
	s *= float64(value) / 100000
	if s < 0 {
		s = 0
	} else if s > 1 {
		s = 1
	}
	r, g, b := hslToRGB(h, s, l)
	c.R = r
	c.G = g
	c.B = b
	return c
}

func applySaturation(c color.RGBA, value int64) color.RGBA {
	h, _, l := rgbToHSL(c)
	s := clampFloat(float64(value)/100000, 0, 1)
	c.R, c.G, c.B = hslToRGB(h, s, l)
	return c
}

func applySaturationOffset(c color.RGBA, value int64) color.RGBA {
	if value == 0 {
		return c
	}
	h, s, l := rgbToHSL(c)
	s += float64(value) / 100000
	if s < 0 {
		s = 0
	} else if s > 1 {
		s = 1
	}
	r, g, b := hslToRGB(h, s, l)
	c.R = r
	c.G = g
	c.B = b
	return c
}

func applyHueOffset(c color.RGBA, value int64) color.RGBA {
	if value == 0 {
		return c
	}
	h, s, l := rgbToHSL(c)
	h = math.Mod(h+float64(value)/60000, 360)
	if h < 0 {
		h += 360
	}
	r, g, b := hslToRGB(h, s, l)
	c.R = r
	c.G = g
	c.B = b
	return c
}

func applyHue(c color.RGBA, value int64) color.RGBA {
	_, s, l := rgbToHSL(c)
	h := math.Mod(float64(value)/60000, 360)
	if h < 0 {
		h += 360
	}
	c.R, c.G, c.B = hslToRGB(h, s, l)
	return c
}

func applyHueModifier(c color.RGBA, value int64) color.RGBA {
	h, s, l := rgbToHSL(c)
	h = math.Mod(h*float64(value)/100000, 360)
	if h < 0 {
		h += 360
	}
	c.R, c.G, c.B = hslToRGB(h, s, l)
	return c
}

func applyGrayscale(c color.RGBA) color.RGBA {
	y := clampColor(int64(math.Round(0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B))))
	c.R = y
	c.G = y
	c.B = y
	return c
}

func applyGammaTransform(c color.RGBA) color.RGBA {
	c.R = linearToSRGBByte(float64(c.R) / 255)
	c.G = linearToSRGBByte(float64(c.G) / 255)
	c.B = linearToSRGBByte(float64(c.B) / 255)
	return c
}

func applyInverseGammaTransform(c color.RGBA) color.RGBA {
	c.R = roundUnitColorChannel(srgbByteToLinear(c.R))
	c.G = roundUnitColorChannel(srgbByteToLinear(c.G))
	c.B = roundUnitColorChannel(srgbByteToLinear(c.B))
	return c
}

func rgbToHSL(c color.RGBA) (float64, float64, float64) {
	r := float64(c.R) / 255
	g := float64(c.G) / 255
	b := float64(c.B) / 255
	maxChannel := math.Max(r, math.Max(g, b))
	minChannel := math.Min(r, math.Min(g, b))
	l := (maxChannel + minChannel) / 2
	if maxChannel == minChannel {
		return 0, 0, l
	}
	delta := maxChannel - minChannel
	s := delta / (1 - math.Abs(2*l-1))
	var h float64
	switch maxChannel {
	case r:
		h = math.Mod((g-b)/delta, 6)
	case g:
		h = (b-r)/delta + 2
	default:
		h = (r-g)/delta + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, l
}

func hslToRGB(h float64, s float64, l float64) (uint8, uint8, uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r1, g1, b1 float64
	switch {
	case h < 60:
		r1, g1, b1 = c, x, 0
	case h < 120:
		r1, g1, b1 = x, c, 0
	case h < 180:
		r1, g1, b1 = 0, c, x
	case h < 240:
		r1, g1, b1 = 0, x, c
	case h < 300:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}
	return roundUnitColorChannel(r1 + m),
		roundUnitColorChannel(g1 + m),
		roundUnitColorChannel(b1 + m)
}

func roundUnitColorChannel(value float64) uint8 {
	return clampColor(int64(math.Floor(value*255 + 0.5 + 1e-9)))
}

func clampColor(value int64) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}
