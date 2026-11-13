package command_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

func run(stdin string, argv ...string) (int, string) {
	var stdout bytes.Buffer
	code := command.Run(context.Background(), "1.0.0", argv, strings.NewReader(stdin), &stdout)

	return code, stdout.String()
}

func answer(stdin string, call func(input []byte) ([]byte, error)) (int, string) {
	var stdout bytes.Buffer
	code := command.Answer(strings.NewReader(stdin), &stdout, call)

	return code, stdout.String()
}

func errorCode(t *testing.T, stdout string) string {
	t.Helper()
	var reply responses.Error
	if err := json.Unmarshal([]byte(stdout), &reply); err != nil {
		t.Fatalf("stdout %q: %v", stdout, err)
	}

	return string(reply.Error.Code)
}

func TestExitCodesFollowTheErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		exit int
		code string
	}{
		{"invalid", command.Invalid("bad"), 64, "invalid_request"},
		{"failure", command.NotFound("replica"), 1, "not_found"},
		{"plain error", errors.New("boom"), 70, "internal"},
	}
	for _, c := range cases {
		code, stdout := answer(`{}`, func([]byte) ([]byte, error) { return nil, c.err })
		if code != c.exit || errorCode(t, stdout) != c.code || !strings.Contains(stdout, `"details":{}`) {
			t.Errorf("%s: code = %d, stdout = %q", c.name, code, stdout)
		}
	}
}

func TestPanicExits70WithOneJSONObject(t *testing.T) {
	code, stdout := answer(`{}`, func([]byte) ([]byte, error) { panic("boom") })
	if code != 70 || errorCode(t, stdout) != "internal" || strings.Count(stdout, "\n") != 1 {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
}

func TestEngineInfoReadsTheImageEnvironment(t *testing.T) {
	t.Setenv("DATABASIFY_ENGINE", "acme")
	t.Setenv("DATABASIFY_LINE", "3")
	t.Setenv("DATABASIFY_RELEASE", "3.1.4")
	code, stdout := run(`{}`, "engine", "info")
	var reply responses.EngineInfo
	if err := json.Unmarshal([]byte(stdout), &reply); code != 0 || err != nil || reply != (responses.EngineInfo{Engine: "acme", Line: "3", Version: "3.1.4", Sdk: "1.0.0"}) {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
	if code, _ := run(``, "engine", "info"); code != 64 {
		t.Fatalf("empty stdin: code = %d", code)
	}
}

func TestRuntimeInfoNamesTheImageEngine(t *testing.T) {
	t.Setenv("DATABASIFY_ENGINE", "acme")
	code, stdout := run(`{}`, "runtime", "info")
	var reply responses.RuntimeInfo
	if err := json.Unmarshal([]byte(stdout), &reply); code != 0 || err != nil || reply.Engine != "acme" || reply.Protocol != command.RuntimeProtocol {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
}

func TestIsWellFormed(t *testing.T) {
	for argv, want := range map[string]bool{"runtime run": true, "task status": true, "widget frobnicate": true, "ctl status": true, "node": false, "node status extra": false, "Node status": false, "node status!": false, "node -status": false} {
		if got := command.IsWellFormed(strings.Fields(argv)); got != want {
			t.Errorf("%s: %v", argv, got)
		}
	}
}

func TestEngineCommandsAreUnsupported(t *testing.T) {
	for _, argv := range []string{"node status", "user create", "engine capabilities", "engine host-profile", "operation status"} {
		code, stdout := run(`{}`, strings.Fields(argv)...)
		if code != 1 || errorCode(t, stdout) != "unsupported" {
			t.Errorf("%s: code = %d, stdout = %s", argv, code, stdout)
		}
	}
}

func TestDecodeErrorsNeverEchoSecrets(t *testing.T) {
	for _, stdin := range []string{`{"script":"true","env":{"SECRET":"s3cret-Pw"},"timeoutSeconds":10,"x":1}`, `{"script":12345,"env":{}}`} {
		code, stdout := run(stdin, "runtime", "run")
		if code != 64 || strings.Contains(stdout, "s3cret-Pw") || strings.Contains(stdout, "12345") {
			t.Errorf("%s: code = %d, stdout = %q", stdin, code, stdout)
		}
	}
	_ = requests.RuntimeRun{}
}
