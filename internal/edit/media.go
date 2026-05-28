package edit

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

func applyImageReplacement(pkg *pptx.Package, spec model.EditSpec, matches []model.TargetMatch) ([]model.ChangeItem, error) {
	if len(matches) != 1 {
		return nil, fmt.Errorf("image replacement requires one resolved image target")
	}
	replacement, err := os.ReadFile(spec.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("read replacement image: %w", err)
	}
	targetPart, err := imageTargetPart(pkg, matches[0])
	if err != nil {
		return nil, err
	}
	if _, ok := pkg.Parts[targetPart]; !ok {
		return nil, fmt.Errorf("image target part missing: %s", targetPart)
	}
	pkg.Parts[targetPart] = replacement
	return []model.ChangeItem{{
		SlideNumber: matches[0].SlideNumber,
		ObjectID:    matches[0].ObjectID,
		Message:     fmt.Sprintf("Replaced image target %s.", targetPart),
	}}, nil
}

func verifyImageReplacement(ctx context.Context, outputPath string, spec model.EditSpec, matches []model.TargetMatch) error {
	replacement, err := os.ReadFile(spec.ImagePath)
	if err != nil {
		return err
	}
	pkg, err := pptx.Open(ctx, outputPath)
	if err != nil {
		return err
	}
	targetPart, err := imageTargetPart(pkg, matches[0])
	if err != nil {
		return err
	}
	if !bytes.Equal(pkg.Parts[targetPart], replacement) {
		return fmt.Errorf("image replacement bytes not found in %s", targetPart)
	}
	return nil
}

func imageTargetPart(pkg *pptx.Package, match model.TargetMatch) (string, error) {
	relationshipID, ok := strings.CutPrefix(match.ObjectID, match.SlideID+"#")
	if !ok || relationshipID == "" {
		return "", fmt.Errorf("image object id does not include slide relationship: %s", match.ObjectID)
	}
	relationships, err := pkg.RelationshipsForPart(match.SlideID)
	if err != nil {
		return "", err
	}
	for _, relationship := range relationships {
		if relationship.ID == relationshipID && relationship.Type == pptx.ImageRelType {
			return pptx.ResolveTargetPart(match.SlideID, relationship.Target), nil
		}
	}
	return "", fmt.Errorf("image relationship %s not found on %s", relationshipID, match.SlideID)
}

func applySimpleAddition(pkg *pptx.Package, spec model.EditSpec, matches []model.TargetMatch) ([]model.ChangeItem, error) {
	if len(matches) != 1 {
		return nil, fmt.Errorf("simple addition requires one resolved slide target")
	}
	match := matches[0]
	data, ok := pkg.Parts[match.SlideID]
	if !ok {
		return nil, fmt.Errorf("slide part missing: %s", match.SlideID)
	}
	nextID, err := nextShapeID(data)
	if err != nil {
		return nil, err
	}
	shapeName := "Puppt Text Box"
	withShape := false
	if spec.Operation == "add_shape" {
		shapeName = "Puppt Rectangle"
		withShape = true
	}
	updated, err := appendShapeToSlide(data, simpleEditableShapeXML(nextID, shapeName, spec.Replacement, withShape))
	if err != nil {
		return nil, err
	}
	pkg.Parts[match.SlideID] = updated
	return []model.ChangeItem{{
		SlideNumber: match.SlideNumber,
		ObjectID:    fmt.Sprintf("%s#shape-%d", match.SlideID, nextID),
		Message:     fmt.Sprintf("Added editable object to slide %d.", match.SlideNumber),
	}}, nil
}

func nextShapeID(data []byte) (int, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	maxID := 1
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return maxID + 1, nil
			}
			return 0, err
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "cNvPr" {
			continue
		}
		for _, attr := range start.Attr {
			if attr.Name.Local != "id" {
				continue
			}
			id, err := strconv.Atoi(attr.Value)
			if err == nil && id > maxID {
				maxID = id
			}
		}
	}
}

func appendShapeToSlide(data []byte, shapeXML []byte) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	inserted := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if end, ok := token.(xml.EndElement); ok && end.Name.Local == "spTree" {
			if err := encoder.Flush(); err != nil {
				return nil, err
			}
			output.Write(shapeXML)
			inserted = true
		}
		if err := encoder.EncodeToken(token); err != nil {
			return nil, err
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, err
	}
	if !inserted {
		return nil, fmt.Errorf("slide shape tree not found")
	}
	return output.Bytes(), nil
}

func simpleEditableShapeXML(id int, name string, text string, withShape bool) []byte {
	shapeProperties := ""
	if withShape {
		shapeProperties = `
        <p:spPr>
          <a:prstGeom xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" prst="rect">
            <a:avLst/>
          </a:prstGeom>
        </p:spPr>`
	}
	return []byte(fmt.Sprintf(`
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="%s"/>
        </p:nvSpPr>%s
        <p:txBody>
          <a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
            <a:r><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>`, id, name, shapeProperties, escapeXMLText(text)))
}
