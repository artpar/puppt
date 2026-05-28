package pptx

import "fmt"

// ErrorKind classifies package-reader failures for CLI/API reporting.
type ErrorKind string

const (
	ErrorUnsupportedFileType ErrorKind = "unsupported_file_type"
	ErrorInvalidPackage      ErrorKind = "invalid_package"
	ErrorMissingPart         ErrorKind = "missing_part"
	ErrorInvalidXML          ErrorKind = "invalid_xml"
	ErrorMissingRelationship ErrorKind = "missing_relationship"
)

// PackageError is an explicit, actionable error from .pptx package handling.
type PackageError struct {
	Kind ErrorKind
	Op   string
	Path string
	Part string
	Err  error
}

func (e *PackageError) Error() string {
	location := e.Path
	if e.Part != "" {
		location = fmt.Sprintf("%s:%s", location, e.Part)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s %s: %v", e.Kind, location, e.Err)
	}
	return fmt.Sprintf("%s %s", e.Kind, location)
}

func (e *PackageError) Unwrap() error {
	return e.Err
}

func packageError(kind ErrorKind, op string, filePath string, part string, err error) error {
	return &PackageError{
		Kind: kind,
		Op:   op,
		Path: filePath,
		Part: part,
		Err:  err,
	}
}
