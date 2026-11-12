package command_test

import (
	"strings"
	"testing"
)

func TestRuntimeInfoReportsProtocolOne(t *testing.T) {
	code, stdout := run(fakeEngine{}, `{}`, "runtime", "info")

	if code != 0 || !strings.Contains(stdout, `"protocol":1`) || !strings.Contains(stdout, `"sdk":"0.2.0"`) || !strings.Contains(stdout, `"engine":"fake"`) {
		t.Fatalf("%d %s", code, stdout)
	}
}

func TestRuntimeRunRefusesAZeroTimeout(t *testing.T) {
	code, stdout := run(fakeEngine{}, `{"script":"echo hi","env":{},"user":null,"timeoutSeconds":0}`, "runtime", "run")

	if code != 64 || !strings.Contains(stdout, "invalid_request") {
		t.Fatalf("%d %s", code, stdout)
	}
}
