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
	if scheme := firstChild(node, "schemeClr"); scheme != nil {
		if c, ok := schemeColorWithTheme(attrValue(scheme.Attrs, "val"), theme); ok {
			return applyColorModifiers(c, scheme), true
		}
	}
	if sys := firstChild(node, "sysClr"); sys != nil {
		if c, ok := parseHexColor(attrValue(sys.Attrs, "lastClr")); ok {
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
	case "black":
		return color.RGBA{A: 255}, true
	case "red":
		return color.RGBA{R: 255, A: 255}, true
	case "white":
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
			c.A = scaleColorChannel(c.A, value)
		case "alphaOff":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c.A = offsetColorChannel(c.A, value)
		case "hueOff":
			flushLuminance()
			value := parseIntAttr(child.Attrs, "val")
			c = applyHueOffset(c, value)
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
		}
	}
	flushLuminance()
	return c
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
