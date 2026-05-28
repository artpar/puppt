package report

import (
	"encoding/json"
	"io"
)

// WriteJSON emits stable, indented JSON for command output and golden tests.
func WriteJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
