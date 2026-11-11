package responses

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMigrationCheckAlwaysSendsLists(t *testing.T) {
	encoded, err := json.Marshal(MigrationCheck{})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(encoded), "null]") || strings.Count(string(encoded), "[]") != 6 {
		t.Fatalf("encoded = %s", encoded)
	}
}
