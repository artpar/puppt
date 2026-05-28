package pptx

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	contentTypesPart      = "[Content_Types].xml"
	rootRelationshipsPart = "_rels/.rels"
	officeDocumentRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"
	slideRelType          = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide"
	CorePropertiesRelType = "http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties"
	SlideLayoutRelType    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout"
	NotesSlideRelType     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide"
	ImageRelType          = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
)

// Package is the initial structural view of a .pptx package.
type Package struct {
	Path                      string
	Parts                     map[string][]byte
	ContentTypes              ContentTypes
	RootRelationships         []Relationship
	PresentationPath          string
	PresentationRelationships []Relationship
	SlideParts                []string
}

// PartNames returns package part names in deterministic order.
func (p *Package) PartNames() []string {
	names := make([]string, 0, len(p.Parts))
	for name := range p.Parts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RelationshipsForPart returns explicit relationships for a package part. A
// part without a relationships item returns an empty slice.
func (p *Package) RelationshipsForPart(partName string) ([]Relationship, error) {
	relationshipsPart := RelationshipsPartFor(partName)
	data, ok := p.Parts[relationshipsPart]
	if !ok {
		return []Relationship{}, nil
	}
	relationships, err := parseRelationships(data)
	if err != nil {
		return nil, packageError(ErrorInvalidXML, "parse", p.Path, relationshipsPart, err)
	}
	return relationships, nil
}

// ContentTypes contains parsed package content-type declarations.
type ContentTypes struct {
	Defaults  map[string]string
	Overrides map[string]string
}

// ForPart returns the best known content type for a package part.
func (c ContentTypes) ForPart(partName string) string {
	normalized := normalizePartName(partName)
	if contentType, ok := c.Overrides[normalized]; ok {
		return contentType
	}
	extension := strings.TrimPrefix(strings.ToLower(path.Ext(normalized)), ".")
	return c.Defaults[extension]
}

// Relationship is an Open Packaging Convention relationship.
type Relationship struct {
	ID         string
	Type       string
	Target     string
	TargetMode string
}

// Open reads enough of a .pptx package to validate core structure and expose
// presentation slide order. It does not mutate or normalize the package.
func Open(ctx context.Context, filePath string) (*Package, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.ToLower(filepath.Ext(filePath)) != ".pptx" {
		return nil, packageError(ErrorUnsupportedFileType, "open", filePath, "", errors.New("expected .pptx extension"))
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, packageError(ErrorInvalidPackage, "open", filePath, "", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, packageError(ErrorInvalidPackage, "stat", filePath, "", err)
	}

	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil, packageError(ErrorInvalidPackage, "zip", filePath, "", err)
	}

	parts, err := readParts(ctx, reader)
	if err != nil {
		return nil, packageError(ErrorInvalidPackage, "read", filePath, "", err)
	}

	contentTypesData, ok := parts[contentTypesPart]
	if !ok {
		return nil, packageError(ErrorMissingPart, "read", filePath, contentTypesPart, nil)
	}
	contentTypes, err := parseContentTypes(contentTypesData)
	if err != nil {
		return nil, packageError(ErrorInvalidXML, "parse", filePath, contentTypesPart, err)
	}

	rootRelsData, ok := parts[rootRelationshipsPart]
	if !ok {
		return nil, packageError(ErrorMissingPart, "read", filePath, rootRelationshipsPart, nil)
	}
	rootRels, err := parseRelationships(rootRelsData)
	if err != nil {
		return nil, packageError(ErrorInvalidXML, "parse", filePath, rootRelationshipsPart, err)
	}

	presentationPath, err := findOfficeDocumentPath(rootRels)
	if err != nil {
		return nil, packageError(ErrorMissingRelationship, "resolve", filePath, rootRelationshipsPart, err)
	}
	presentationData, ok := parts[presentationPath]
	if !ok {
		return nil, packageError(ErrorMissingPart, "read", filePath, presentationPath, nil)
	}

	presentationRelsPath := relationshipsPartFor(presentationPath)
	presentationRelsData, ok := parts[presentationRelsPath]
	if !ok {
		return nil, packageError(ErrorMissingPart, "read", filePath, presentationRelsPath, nil)
	}
	presentationRels, err := parseRelationships(presentationRelsData)
	if err != nil {
		return nil, packageError(ErrorInvalidXML, "parse", filePath, presentationRelsPath, err)
	}

	slideIDs, err := parsePresentationSlideIDs(presentationData)
	if err != nil {
		return nil, packageError(ErrorInvalidXML, "parse", filePath, presentationPath, err)
	}
	slideParts, err := resolveSlideParts(presentationPath, slideIDs, presentationRels)
	if err != nil {
		return nil, packageError(ErrorMissingRelationship, "resolve", filePath, presentationRelsPath, err)
	}
	for _, slidePart := range slideParts {
		if _, ok := parts[slidePart]; !ok {
			return nil, packageError(ErrorMissingPart, "read", filePath, slidePart, nil)
		}
	}

	return &Package{
		Path:                      filePath,
		Parts:                     parts,
		ContentTypes:              contentTypes,
		RootRelationships:         rootRels,
		PresentationPath:          presentationPath,
		PresentationRelationships: presentationRels,
		SlideParts:                slideParts,
	}, nil
}

func readParts(ctx context.Context, reader *zip.Reader) (map[string][]byte, error) {
	parts := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if file.FileInfo().IsDir() {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(handle)
		closeErr := handle.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		parts[file.Name] = data
	}
	return parts, nil
}

type contentTypesXML struct {
	Defaults  []contentTypeDefault  `xml:"Default"`
	Overrides []contentTypeOverride `xml:"Override"`
}

type contentTypeDefault struct {
	Extension   string `xml:"Extension,attr"`
	ContentType string `xml:"ContentType,attr"`
}

type contentTypeOverride struct {
	PartName    string `xml:"PartName,attr"`
	ContentType string `xml:"ContentType,attr"`
}

func parseContentTypes(data []byte) (ContentTypes, error) {
	var raw contentTypesXML
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&raw); err != nil {
		return ContentTypes{}, err
	}
	result := ContentTypes{
		Defaults:  make(map[string]string, len(raw.Defaults)),
		Overrides: make(map[string]string, len(raw.Overrides)),
	}
	for _, item := range raw.Defaults {
		result.Defaults[strings.ToLower(item.Extension)] = item.ContentType
	}
	for _, item := range raw.Overrides {
		result.Overrides[strings.TrimPrefix(item.PartName, "/")] = item.ContentType
	}
	return result, nil
}

type relationshipsXML struct {
	Relationships []relationshipXML `xml:"Relationship"`
}

type relationshipXML struct {
	ID         string `xml:"Id,attr"`
	Type       string `xml:"Type,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr"`
}

func parseRelationships(data []byte) ([]Relationship, error) {
	var raw relationshipsXML
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&raw); err != nil {
		return nil, err
	}
	relationships := make([]Relationship, 0, len(raw.Relationships))
	for _, item := range raw.Relationships {
		relationships = append(relationships, Relationship{
			ID:         item.ID,
			Type:       item.Type,
			Target:     item.Target,
			TargetMode: item.TargetMode,
		})
	}
	return relationships, nil
}

func findOfficeDocumentPath(relationships []Relationship) (string, error) {
	for _, rel := range relationships {
		if rel.Type == officeDocumentRelType {
			return normalizePartName(rel.Target), nil
		}
	}
	return "", errors.New("office document relationship not found")
}

// RelationshipsPartFor returns the package relationships item name for a part.
func RelationshipsPartFor(partName string) string {
	dir := path.Dir(partName)
	base := path.Base(partName)
	if dir == "." {
		return path.Join("_rels", base+".rels")
	}
	return path.Join(dir, "_rels", base+".rels")
}

func relationshipsPartFor(partName string) string {
	return RelationshipsPartFor(partName)
}

type presentationXML struct {
	SlideIDList slideIDListXML `xml:"sldIdLst"`
}

type slideIDListXML struct {
	SlideIDs []slideIDXML `xml:"sldId"`
}

type slideIDXML struct {
	ID    string `xml:"id,attr"`
	Attrs []xml.Attr
}

func (s slideIDXML) relationshipID() string {
	for _, attr := range s.Attrs {
		if attr.Name.Local == "id" && attr.Name.Space != "" {
			return attr.Value
		}
	}
	for _, attr := range s.Attrs {
		if attr.Name.Local == "id" {
			return attr.Value
		}
	}
	return ""
}

func (s *slideIDXML) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "id" && attr.Name.Space == "" {
			s.ID = attr.Value
		}
		s.Attrs = append(s.Attrs, attr)
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		if end, ok := token.(xml.EndElement); ok && end.Name == start.Name {
			return nil
		}
	}
}

func parsePresentationSlideIDs(data []byte) ([]string, error) {
	var raw presentationXML
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&raw); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(raw.SlideIDList.SlideIDs))
	for _, slideID := range raw.SlideIDList.SlideIDs {
		rid := slideID.relationshipID()
		if rid == "" {
			return nil, errors.New("slide id entry missing relationship id")
		}
		result = append(result, rid)
	}
	return result, nil
}

func resolveSlideParts(presentationPath string, relationshipIDs []string, relationships []Relationship) ([]string, error) {
	byID := make(map[string]Relationship, len(relationships))
	for _, rel := range relationships {
		byID[rel.ID] = rel
	}

	slideParts := make([]string, 0, len(relationshipIDs))
	for _, relationshipID := range relationshipIDs {
		rel, ok := byID[relationshipID]
		if !ok {
			return nil, fmt.Errorf("slide relationship %q not found", relationshipID)
		}
		if rel.Type != slideRelType {
			return nil, fmt.Errorf("relationship %q is %q, not slide", relationshipID, rel.Type)
		}
		if rel.TargetMode != "" && !strings.EqualFold(rel.TargetMode, "Internal") {
			return nil, fmt.Errorf("slide relationship %q is external", relationshipID)
		}
		slideParts = append(slideParts, resolveTargetPart(presentationPath, rel.Target))
	}
	return slideParts, nil
}

func resolveTargetPart(sourcePart string, target string) string {
	return ResolveTargetPart(sourcePart, target)
}

// ResolveTargetPart resolves an internal relationship target relative to its
// source part.
func ResolveTargetPart(sourcePart string, target string) string {
	normalized := normalizePartName(target)
	if strings.HasPrefix(target, "/") {
		return normalized
	}
	return normalizePartName(path.Join(path.Dir(sourcePart), target))
}

func normalizePartName(partName string) string {
	return path.Clean(strings.TrimPrefix(partName, "/"))
}
