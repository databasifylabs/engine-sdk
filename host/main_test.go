package host

import (
	"bytes"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/ctl"
)

func TestMainRefusesAMalformedCommand(t *testing.T) {
	for _, args := range [][]string{{}, {"serve"}, {"info"}, {"version"}, {"Node", "status"}, {"node", "status", "extra"}, {"ctl", "restart", "unit"}} {
		var stderr bytes.Buffer

		exit := Main(args, strings.NewReader(""), &bytes.Buffer{}, &stderr)

		if exit != ctl.ExitUsage || !strings.HasPrefix(stderr.String(), "usage: databasify") {
			t.Errorf("%v: exit = %d, stderr = %q", args, exit, stderr.String())
		}
	}
}

func TestMainForwardsAnyWellFormedCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var stderr bytes.Buffer

	exit := Main([]string{"widget", "frobnicate"}, strings.NewReader("{}"), &bytes.Buffer{}, &stderr)

	if exit != ctl.ExitUnavailable || !strings.Contains(stderr.String(), "cannot list containers") {
		t.Fatalf("exit = %d, stderr = %q", exit, stderr.String())
	}
}
