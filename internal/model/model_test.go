package model

import "testing"

func TestSchemaVersionIsStable(t *testing.T) {
	if SchemaVersion != "puppt.v1" {
		t.Fatalf("unexpected schema version: %s", SchemaVersion)
	}
}
