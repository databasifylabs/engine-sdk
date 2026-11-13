package command_test

import (
	"strings"
	"testing"
)

func TestRuntimeRunRefusesAZeroTimeout(t *testing.T) {
	code, stdout := run(`{"script":"echo hi","env":{},"user":null,"timeoutSeconds":0}`, "runtime", "run")

	if code != 64 || !strings.Contains(stdout, "invalid_request") {
		t.Fatalf("%d %s", code, stdout)
	}
}
