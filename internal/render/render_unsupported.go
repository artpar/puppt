package render

import (
	"fmt"
	"strings"

	"github.com/artpar/puppt/internal/model"
)

func unsupportedItems(slidePart string, elements []slideElement) []model.SkipItem {
	items := make([]model.SkipItem, 0, len(elements))
	for _, element := range elements {
		if element.Rendered {
			continue
		}
		if element.UnsupportedNote != "" {
			continue
		}
		if element.IsPlaceholder && element.Text == "" && element.EmbedID == "" {
			continue
		}
		if strings.Contains(strings.ToLower(element.Name), "placeholder") && element.Text == "" && element.EmbedID == "" {
			continue
		}
		if element.ID == "" && element.Name == "" && element.Text == "" {
			continue
		}
		message := fmt.Sprintf("%s object %q was detected but is not rendered yet", objectKindLabel(element.Kind), elementLabel(element))
		if element.Text != "" {
			message = fmt.Sprintf("%s object %q contains text and is not rendered yet", objectKindLabel(element.Kind), elementLabel(element))
		}
		items = append(items, unsupportedItem(slidePart, unsupportedCode, message))
	}
	return items
}

func timingUnsupportedItems(slidePart string, data []byte, elements []slideElement) []model.SkipItem {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil
	}
	timing := firstDescendant(root, "timing")
	if timing == nil || !timingHasAnimationBehavior(timing) || timingHasSupportedStaticVisibilityBuilds(root, timing, elements) {
		return nil
	}
	return []model.SkipItem{unsupportedItem(slidePart, partialUnsupportedCode, "slide animation timing was not evaluated for static rendering")}
}

func timingHasSupportedStaticVisibilityBuilds(root *xmlNode, timing *xmlNode, elements []slideElement) bool {
	targetIDs := partObjectIDs(root, elements)
	if len(targetIDs) == 0 {
		return false
	}
	seenVisibilityBuild := false
	for _, node := range timingBehaviorNodes(timing) {
		switch node.Name {
		case "set":
			targetID, ok := timingSetVisibilityTarget(node)
			if !ok || !targetIDs[targetID] {
				return false
			}
			seenVisibilityBuild = true
		case "animEffect":
			targetID, ok := timingStaticEntranceEffectTarget(node)
			if !ok || !targetIDs[targetID] {
				return false
			}
		case "cTn":
			if !timingContainerIsSupportedStaticEntrance(node) {
				return false
			}
		default:
			return false
		}
	}
	return seenVisibilityBuild
}

func partObjectIDs(root *xmlNode, elements []slideElement) map[string]bool {
	ids := map[string]bool{}
	for _, property := range descendantsByName(root, "cNvPr") {
		if id := attrValue(property.Attrs, "id"); id != "" {
			ids[id] = true
		}
	}
	for _, element := range elements {
		if element.ID != "" {
			ids[element.ID] = true
		}
	}
	return ids
}

func timingBehaviorNodes(node *xmlNode) []*xmlNode {
	var nodes []*xmlNode
	if timingNodeIsBehavior(node) {
		nodes = append(nodes, node)
	}
	for _, child := range node.Children {
		nodes = append(nodes, timingBehaviorNodes(child)...)
	}
	return nodes
}

func timingNodeIsBehavior(node *xmlNode) bool {
	switch node.Name {
	case "anim", "animClr", "animEffect", "animMotion", "animRot", "animScale", "cmd", "set":
		return true
	case "cTn":
		return attrValue(node.Attrs, "presetClass") != "" || attrValue(node.Attrs, "presetID") != ""
	default:
		return false
	}
}

func timingSetVisibilityTarget(node *xmlNode) (string, bool) {
	if !timingSetWritesVisibleStyle(node) {
		return "", false
	}
	target := firstDescendant(node, "spTgt")
	if target == nil {
		return "", false
	}
	return attrValue(target.Attrs, "spid"), true
}

func timingStaticEntranceEffectTarget(node *xmlNode) (string, bool) {
	if attrValue(node.Attrs, "transition") != "in" {
		return "", false
	}
	target := firstDescendant(node, "spTgt")
	if target == nil {
		return "", false
	}
	return attrValue(target.Attrs, "spid"), true
}

func timingSetWritesVisibleStyle(node *xmlNode) bool {
	attrNames := descendantsByName(node, "attrName")
	if len(attrNames) != 1 || strings.TrimSpace(attrNames[0].Text) != "style.visibility" {
		return false
	}
	value := firstDescendant(node, "strVal")
	return value != nil && strings.EqualFold(attrValue(value.Attrs, "val"), "visible")
}

func timingContainerIsSupportedStaticEntrance(node *xmlNode) bool {
	presetClass := attrValue(node.Attrs, "presetClass")
	return presetClass == "" || presetClass == "entr"
}

func timingHasAnimationBehavior(timing *xmlNode) bool {
	for _, child := range timing.Children {
		if timingNodeHasAnimationBehavior(child) {
			return true
		}
	}
	return false
}

func timingNodeHasAnimationBehavior(node *xmlNode) bool {
	if timingNodeIsBehavior(node) {
		return true
	}
	for _, child := range node.Children {
		if timingNodeHasAnimationBehavior(child) {
			return true
		}
	}
	return false
}

func unsupportedItem(slidePart string, code string, message string) model.SkipItem {
	return model.SkipItem{
		Code:    code,
		Message: message,
		Part:    slidePart,
	}
}

func elementLabel(element slideElement) string {
	label := strings.TrimSpace(element.Name)
	if label == "" {
		label = element.ID
	}
	if label == "" {
		label = element.Kind
	}
	return label
}

func objectKindLabel(kind string) string {
	switch kind {
	case "sp":
		return "shape"
	case "cxnSp":
		return "connector"
	case "pic":
		return "picture"
	case "graphicFrame":
		return "graphic frame"
	case "grpSp":
		return "group"
	default:
		return kind
	}
}
