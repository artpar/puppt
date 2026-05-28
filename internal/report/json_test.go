package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSONUsesIndentedStableOutput(t *testing.T) {
	var output bytes.Buffer
	value := struct {
		Name string `json:"name"`
	}{Name: "puppt"}
	if err := WriteJSON(&output, value); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "\n  \"name\": \"puppt\"\n") {
		t.Fatalf("unexpected json output: %q", got)
	}
}
