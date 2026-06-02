package render

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"
)

func firstDescendant(node *xmlNode, name string) *xmlNode {
	if node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if found := firstDescendant(child, name); found != nil {
			return found
		}
	}
	return nil
}

func descendantsByName(node *xmlNode, name string) []*xmlNode {
	var output []*xmlNode
	if node.Name == name {
		output = append(output, node)
	}
	for _, child := range node.Children {
		output = append(output, descendantsByName(child, name)...)
	}
	return output
}

func firstChild(node *xmlNode, name string) *xmlNode {
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func childrenByName(node *xmlNode, name string) []*xmlNode {
	var output []*xmlNode
	for _, child := range node.Children {
		if child.Name == name {
			output = append(output, child)
		}
	}
	return output
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func parseIntAttr(attrs []xml.Attr, name string) int64 {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			value, _ := strconv.ParseInt(attr.Value, 10, 64)
			return value
		}
	}
	return 0
}

func parseIntAttrDefault(attrs []xml.Attr, name string, fallback int64) int64 {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			value, _ := strconv.ParseInt(attr.Value, 10, 64)
			return value
		}
	}
	return fallback
}

func parsePercentAttr(attrs []xml.Attr, name string) int64 {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return parsePercentValue(attr.Value)
		}
	}
	return 0
}

func parsePercentAttrDefault(attrs []xml.Attr, name string, fallback int64) int64 {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return parsePercentValue(attr.Value)
		}
	}
	return fallback
}

func parsePercentValue(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "%") {
		parsed, _ := strconv.ParseInt(trimmed, 10, 64)
		return parsed
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(trimmed, "%")), 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(parsed * 1000))
}

func parentElement(stack []string) string {
	if len(stack) < 2 {
		return ""
	}
	return stack[len(stack)-2]
}
