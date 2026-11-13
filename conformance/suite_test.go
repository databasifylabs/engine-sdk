package conformance

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestConformance(t *testing.T) {
	container := os.Getenv("DATABASIFY_CONFORMANCE_CONTAINER")
	if container == "" {
		t.Skip("DATABASIFY_CONFORMANCE_CONTAINER is not set")
	}
	Run(t, DockerRunner{Container: container})
}

type override func(command string, stdin []byte) (stdout []byte, exit int, handled bool)

type fakeRuntime struct {
	mu       sync.Mutex
	override override
	files    map[string]string
}

func newFakeRuntime(override override) *fakeRuntime {
	return &fakeRuntime{override: override, files: map[string]string{}}
}

func (f *fakeRuntime) Run(_ context.Context, argv []string, stdin []byte) ([]byte, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	command := strings.Join(argv, " ")
	if f.override != nil {
		if stdout, exit, handled := f.override(command, stdin); handled {
			return stdout, exit, nil
		}
	}
	stdout, exit := f.answer(command, stdin)

	return stdout, exit, nil
}

func (f *fakeRuntime) answer(command string, stdin []byte) ([]byte, int) {
	var body map[string]any
	if err := json.Unmarshal(stdin, &body); err != nil {
		return fakeError("invalid_request"), 64
	}
	switch command {
	case "engine info":
		return []byte(`{"engine":"fake","line":"1","version":"1.0","sdk":"1.0.0"}`), 0
	case "runtime run":
		return []byte(`{"exit":0,"stdout":"ok\n","stderr":"","truncated":false}`), 0
	case "file put":
		f.files[body["path"].(string)] = body["contentBase64"].(string)

		return []byte(`{}`), 0
	case "file get":
		content, exists := f.files[body["path"].(string)]
		reply, _ := json.Marshal(map[string]any{"contentBase64": content, "exists": exists})

		return reply, 0
	case "task start", "task status":
		return []byte(`{"key":"k","kind":"task","state":"succeeded","phase":"","progress":null,"startedAt":"2026-11-13T00:00:00Z","finishedAt":null,"error":null}`), 0
	}

	return fakeError("unsupported"), 1
}

func fakeError(code string) []byte {
	return []byte(`{"error":{"code":"` + code + `","message":"fake","details":{}}}`)
}

func answering(commands []string, reply func(string) []byte) override {
	return func(command string, _ []byte) ([]byte, int, bool) {
		if !slices.Contains(commands, command) {
			return nil, 0, false
		}

		return reply(command), 0, true
	}
}

func failedSteps(failures []Failure) []string {
	steps := []string{}
	for _, failure := range failures {
		steps = append(steps, failure.Step)
	}

	return steps
}

func assertFailedSteps(t *testing.T, r Runner, want ...string) {
	t.Helper()
	steps := failedSteps(Failures(context.Background(), r))
	for _, step := range want {
		if !slices.Contains(steps, step) {
			t.Fatalf("failed steps = %v, want %s", steps, step)
		}
	}
}

func TestRunReportsNoFailuresForAGoodRuntime(t *testing.T) {
	if failures := Failures(context.Background(), newFakeRuntime(nil)); len(failures) != 0 {
		t.Fatalf("failures = %v", failures)
	}
}

func TestRunFailsOnNonJSONOutput(t *testing.T) {
	broken := newFakeRuntime(func(command string, _ []byte) ([]byte, int, bool) {
		return []byte("panic: boom"), 0, command == "engine info"
	})
	assertFailedSteps(t, broken, "engine info")
}

func TestRunFailsWhenRuntimeRunLosesOutput(t *testing.T) {
	silent := answering([]string{"runtime run"}, func(string) []byte { return []byte(`{"exit":0,"stdout":"","stderr":"","truncated":false}`) })
	assertFailedSteps(t, newFakeRuntime(silent), "runtime run")
}

func TestRunFailsWhenAFileDoesNotRoundTrip(t *testing.T) {
	forgetful := answering([]string{"file get"}, func(string) []byte { return []byte(`{"contentBase64":"","exists":false}`) })
	assertFailedSteps(t, newFakeRuntime(forgetful), "file round trip")
}

func TestRunFailsWhenAnEnvSecretIsEchoed(t *testing.T) {
	echoing := answering([]string{"runtime run"}, func(string) []byte {
		reply, _ := json.Marshal(map[string]any{"exit": 0, "stdout": "ok\n" + secret, "stderr": "", "truncated": false})

		return reply
	})
	assertFailedSteps(t, newFakeRuntime(echoing), "runtime run keeps env secrets out of the reply")
}

func TestRunFailsWhenAnEngineCommandIsAnswered(t *testing.T) {
	answered := answering([]string{"node status"}, func(string) []byte { return []byte(`{"role":"primary","version":"1"}`) })
	assertFailedSteps(t, newFakeRuntime(answered), "engine commands are unsupported")
}
