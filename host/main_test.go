package host

import (
	"bytes"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/ctl"
)

func TestMainRefusesAMalformedCommand(t *testing.T) {
	for _, args := range [][]string{{}, {"serve"}, {"info"}, {"Node", "status"}, {"node", "status", "extra"}, {"ctl", "restart", "unit"}} {
		var stderr bytes.Buffer

		exit := Main(args, strings.NewReader(""), &bytes.Buffer{}, &stderr, "0.2.0")

		if exit != ctl.ExitUsage || !strings.HasPrefix(stderr.String(), "usage: databasify") {
			t.Errorf("%v: exit = %d, stderr = %q", args, exit, stderr.String())
		}
	}
}

func TestMainForwardsAnyWellFormedCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var stderr bytes.Buffer

	exit := Main([]string{"widget", "frobnicate"}, strings.NewReader("{}"), &bytes.Buffer{}, &stderr, "0.2.0")

	if exit != ctl.ExitUnavailable || !strings.Contains(stderr.String(), "cannot list containers") {
		t.Fatalf("exit = %d, stderr = %q", exit, stderr.String())
	}
}

func TestMainPrintsTheVersion(t *testing.T) {
	var stdout bytes.Buffer

	if exit := Main([]string{"version"}, strings.NewReader(""), &stdout, &bytes.Buffer{}, "0.2.0"); exit != 0 || stdout.String() != "0.2.0\n" {
		t.Fatalf("exit = %d, stdout = %q", exit, stdout.String())
	}
}
