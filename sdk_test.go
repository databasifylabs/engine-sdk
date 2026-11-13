package sdk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runInContainer(t *testing.T, input string, args ...string) (string, int) {
	t.Helper()
	t.Setenv("DATABASIFY_ENGINE", "fake")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stdin"), []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	stdin, err := os.Open(filepath.Join(dir, "stdin"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	args0, stdin0, stdout0 := os.Args, os.Stdin, os.Stdout
	os.Args, os.Stdin, os.Stdout = append([]string{"databasify"}, args...), stdin, stdout
	exit := Run()
	os.Args, os.Stdin, os.Stdout = args0, stdin0, stdout0
	_ = stdin.Close()
	_ = stdout.Close()
	written, err := os.ReadFile(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}

	return string(written), exit
}

func TestEngineInfoReportsTheSdkVersion(t *testing.T) {
	stdout, exit := runInContainer(t, "{}", "engine", "info")

	if exit != 0 || !strings.Contains(stdout, `"sdk":"`+Version+`"`) || !strings.Contains(stdout, `"engine":"fake"`) {
		t.Fatalf("exit = %d, stdout = %s", exit, stdout)
	}
}

func TestTheVersionIsOnePointZero(t *testing.T) {
	if Version != "1.0.0" {
		t.Fatalf("Version = %s", Version)
	}
}

func TestRunAnswersUnsupportedForAnEngineCommand(t *testing.T) {
	for _, argv := range [][]string{{"node", "status"}, {"widget", "frobnicate"}, {"engine", "capabilities"}} {
		stdout, exit := runInContainer(t, "{}", argv...)

		if exit != 1 || !strings.Contains(stdout, `"code":"unsupported"`) {
			t.Fatalf("%v: exit = %d, stdout = %s", argv, exit, stdout)
		}
	}
}

func TestRunAnswersUsageForAMalformedCommand(t *testing.T) {
	stdout, exit := runInContainer(t, "{}", "Widget", "frobnicate")

	if exit != 64 || stdout != "" {
		t.Fatalf("exit = %d, stdout = %s", exit, stdout)
	}
}

func TestEngineTaskIsGone(t *testing.T) {
	stdout, exit := runInContainer(t, "{}", "engine-task", "backup")

	if exit != 1 || !strings.Contains(stdout, `"code":"unsupported"`) {
		t.Fatalf("exit = %d, stdout = %s", exit, stdout)
	}
}
