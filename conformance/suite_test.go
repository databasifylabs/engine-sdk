package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/support"
)

func TestConformance(t *testing.T) {
	container := os.Getenv("DATABASIFY_CONFORMANCE_CONTAINER")
	if container == "" {
		t.Skip("DATABASIFY_CONFORMANCE_CONTAINER is not set")
	}
	Run(t, DockerRunner{Container: container})
}

type override func(command string, stdin []byte) (stdout []byte, exit int, handled bool)

type fakeEngine struct {
	mu        sync.Mutex
	omit      string
	override  override
	databases map[string]bool
	users     map[string]bool
	files     map[string]string
}

func newFakeEngine(omit string, override override) *fakeEngine {
	return &fakeEngine{omit: omit, override: override, databases: map[string]bool{}, users: map[string]bool{}, files: map[string]string{}}
}

func (f *fakeEngine) Run(_ context.Context, argv []string, stdin []byte) ([]byte, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	command := strings.Join(argv, " ")
	if command == f.omit {
		return fakeError("unsupported"), 1, nil
	}
	if f.override != nil {
		if stdout, exit, handled := f.override(command, stdin); handled {
			return stdout, exit, nil
		}
	}
	stdout, exit := f.answer(command, stdin)
	return stdout, exit, nil
}

func (f *fakeEngine) answer(command string, stdin []byte) ([]byte, int) {
	switch command {
	case "engine info":
		return decoded(stdin, func(dtos.Empty) ([]byte, int) { return exampleReply(command), 0 })
	case "node status", "engine host-profile":
		return decoded(stdin, func(dtos.Empty) ([]byte, int) { return exampleReply(command), 0 })
	case "engine capabilities":
		return decoded(stdin, func(dtos.Empty) ([]byte, int) { return capabilitiesWithout(f.omit), 0 })
	case "account list":
		return decoded(stdin, func(dtos.Empty) ([]byte, int) {
			return success(responses.AccountList{Users: keys(f.users), Databases: keys(f.databases), Grants: map[string][]dtos.Grant{}})
		})
	case "database create":
		return decoded(stdin, func(r requests.DatabaseCreate) ([]byte, int) {
			return success(responses.DatabaseCreate{Created: add(f.databases, r.Name)})
		})
	case "database drop":
		return decoded(stdin, func(r requests.DatabaseDrop) ([]byte, int) {
			return success(responses.DatabaseDrop{Dropped: remove(f.databases, r.Name)})
		})
	case "user create":
		return decoded(stdin, func(r requests.UserCreate) ([]byte, int) {
			return success(responses.UserCreate{Created: add(f.users, r.Name)})
		})
	case "user set-password":
		return decoded(stdin, func(r requests.UserSetPassword) ([]byte, int) {
			return f.forExistingUser(r.Name, dtos.Empty{})
		})
	case "user grant":
		return decoded(stdin, func(r requests.UserGrant) ([]byte, int) {
			return f.forExistingUser(r.Name, dtos.Empty{})
		})
	case "user drop":
		return decoded(stdin, func(r requests.UserDrop) ([]byte, int) {
			return success(responses.UserDrop{Dropped: remove(f.users, r.Name)})
		})
	case "operation status", "operation cancel":
		return fakeError("not_found"), 1
	case "runtime info":
		return decoded(stdin, func(dtos.Empty) ([]byte, int) {
			return success(responses.RuntimeInfo{Protocol: 1, Engine: "fake", Sdk: "0.2.0"})
		})
	case "runtime run":
		return decoded(stdin, func(r requests.RuntimeRun) ([]byte, int) {
			return success(responses.RuntimeRun{Stdout: strings.TrimPrefix(r.Script, "echo ") + "\n"})
		})
	case "file put":
		return decoded(stdin, func(r requests.FilePut) ([]byte, int) {
			f.files[r.Path] = r.ContentBase64
			return success(dtos.Empty{})
		})
	case "file get":
		return decoded(stdin, func(r requests.FileGet) ([]byte, int) {
			content, exists := f.files[r.Path]
			return success(responses.FileGet{ContentBase64: content, Exists: exists})
		})
	}
	return fakeError("unsupported"), 1
}

func (f *fakeEngine) forExistingUser(name string, reply any) ([]byte, int) {
	if !f.users[name] {
		return fakeError("not_found"), 1
	}
	return success(reply)
}

func capabilitiesWithout(omitted string) []byte {
	var reply responses.EngineCapabilities
	reply.Commands = slices.Clone(fakeCommands)
	reply.PasswordEncryptions, reply.ReservedUsers, reply.ReservedDatabases = []string{}, []string{}, []string{}
	reply.Ports.Peers = []int{}
	reply.Commands = slices.DeleteFunc(reply.Commands, func(name string) bool { return name == omitted })
	return marshal(reply)
}

func decoded[T any](stdin []byte, handle func(T) ([]byte, int)) ([]byte, int) {
	request, err := support.DecodeStrict[T](bytes.NewReader(stdin))
	if err != nil {
		return fakeError("invalid_request"), 64
	}
	return handle(request)
}

func success(value any) ([]byte, int) {
	return marshal(value), 0
}

func add(set map[string]bool, name string) bool {
	created := !set[name]
	set[name] = true
	return created
}

func remove(set map[string]bool, name string) bool {
	dropped := set[name]
	delete(set, name)
	return dropped
}

func keys(set map[string]bool) []string {
	names := []string{}
	for name := range set {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func marshal(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func fakeError(code string) []byte {
	return []byte(`{"error":{"code":"` + code + `","message":"fake","details":{}}}`)
}

func failedSteps(failures []Failure) []string {
	steps := []string{}
	for _, failure := range failures {
		steps = append(steps, failure.Step)
	}
	return steps
}

func TestSuiteAgainstFakeEngine(t *testing.T) {
	engine := newFakeEngine("", nil)
	Run(t, engine)
	if len(engine.databases)+len(engine.users) != 0 {
		t.Fatalf("leftovers: %v %v", engine.databases, engine.users)
	}
}

func TestSuiteAgainstFakeEngineWithoutUserGrant(t *testing.T) {
	Run(t, newFakeEngine("user grant", nil))
}

func TestRunReportsNoFailuresForGoodEngine(t *testing.T) {
	if failures := Failures(context.Background(), newFakeEngine("", nil)); len(failures) != 0 {
		t.Fatalf("failures = %v", failures)
	}
}

func TestRunFailsWhenListedCommandAnswersUnsupported(t *testing.T) {
	engine := newFakeEngine("", func(command string, _ []byte) ([]byte, int, bool) {
		return fakeError("unsupported"), 1, command == "user grant"
	})
	if steps := failedSteps(Failures(context.Background(), engine)); !slices.Contains(steps, "user grant") {
		t.Fatalf("failed steps = %v", steps)
	}
}

func TestRunFailsWhenUnlistedCommandIsAnswered(t *testing.T) {
	omitting := newFakeEngine("", answering([]string{"engine capabilities"}, func(string) []byte {
		return capabilitiesWithout("user grant")
	}))
	if steps := failedSteps(Failures(context.Background(), omitting)); !slices.Contains(steps, "user grant unsupported") {
		t.Fatalf("failed steps = %v", steps)
	}
}

func TestRunFailsOnInternalErrorFromListedCommand(t *testing.T) {
	engine := newFakeEngine("", func(command string, _ []byte) ([]byte, int, bool) {
		return fakeError("internal"), 1, command == "node status"
	})
	if steps := failedSteps(Failures(context.Background(), engine)); !slices.Contains(steps, "node status") {
		t.Fatalf("failed steps = %v", steps)
	}
}

func TestRunFailsOnNonJSONOutput(t *testing.T) {
	engine := newFakeEngine("", func(command string, _ []byte) ([]byte, int, bool) {
		return []byte("panic: boom"), 0, command == "account list"
	})
	failures := Failures(context.Background(), engine)
	if !slices.Contains(failedSteps(failures), "account list") || !strings.Contains(failures[0].Err.Error(), "panic: boom") {
		t.Fatalf("failures = %v", failures)
	}
}

func TestRunFailsWhenDatabaseCreateIsNotIdempotent(t *testing.T) {
	engine := newFakeEngine("", func(command string, _ []byte) ([]byte, int, bool) {
		return []byte(`{"created":true}`), 0, command == "database create"
	})
	if steps := failedSteps(Failures(context.Background(), engine)); !slices.Contains(steps, "database create again") {
		t.Fatalf("failed steps = %v", steps)
	}
}

func answering(commands []string, reply func(command string) []byte) override {
	return func(command string, _ []byte) ([]byte, int, bool) {
		if !slices.Contains(commands, command) {
			return nil, 0, false
		}
		return reply(command), 0, true
	}
}

var fakeCommands = []string{
	"account list", "database create", "database drop", "engine capabilities", "engine host-profile", "engine info", "node status",
	"operation cancel", "operation status", "user create", "user drop", "user grant", "user set-password",
	"runtime info", "runtime run", "file put", "file get",
}

var cannedReplies = map[string]string{
	"engine info":         `{"engine":"fake","line":"18","version":"18.6","sdk":"0.2.0"}`,
	"engine host-profile": `{"settings":{},"floors":{},"disableTransparentHugepages":false}`,
	"node status":         `{"role":"primary","version":"18.6"}`,
	"database create":     `{"created":true}`,
	"user create":         `{"created":true}`,
}

func exampleReply(command string) []byte {
	return []byte(cannedReplies[command])
}

func assertFailedSteps(t *testing.T, engine Runner, want ...string) {
	t.Helper()
	steps := failedSteps(Failures(context.Background(), engine))
	for _, step := range want {
		if !slices.Contains(steps, step) {
			t.Errorf("failed steps = %v, want %q among them", steps, step)
		}
	}
}

func TestRunFailsWhenEngineAcceptsInvalidRequests(t *testing.T) {
	lax := answering([]string{"engine info", "database create", "user create"}, exampleReply)
	assertFailedSteps(t, newFakeEngine("", lax),
		"database create bad identifier",
		"user create empty password",
		"database create missing key",
		"database create case-variant key",
		"engine info array input",
	)
}

func TestRunFailsWhenEngineInfoIsNotListed(t *testing.T) {
	assertFailedSteps(t, newFakeEngine("engine info", nil), "engine info mandatory")
}

func TestRunFailsWhenNodeStatusIsNotListed(t *testing.T) {
	assertFailedSteps(t, newFakeEngine("node status", nil), "node status mandatory")
}

func TestRunFailsWhenUserCreateIsNotIdempotent(t *testing.T) {
	always := answering([]string{"user create"}, func(string) []byte { return []byte(`{"created":true}`) })
	assertFailedSteps(t, newFakeEngine("", always), "user create again")
}

func TestRunFailsWhenSetPasswordAcceptsMissingUser(t *testing.T) {
	always := answering([]string{"user set-password"}, func(string) []byte { return []byte(`{}`) })
	assertFailedSteps(t, newFakeEngine("", always), "user set-password missing user")
}

func TestRunFailsWhenPasswordIsEchoed(t *testing.T) {
	echo := answering([]string{"account list"}, func(string) []byte {
		return marshal(responses.AccountList{Users: []string{quotedPassword}, Databases: []string{}, Grants: map[string][]dtos.Grant{}})
	})
	assertFailedSteps(t, newFakeEngine("", echo), "account list")
}

func TestRunFailsWhenCapabilitiesDoesNotListItself(t *testing.T) {
	unlisted := answering([]string{"engine capabilities"}, func(string) []byte {
		return capabilitiesWithout("engine capabilities")
	})
	assertFailedSteps(t, newFakeEngine("", unlisted), "engine capabilities mandatory")
}

func TestRunFailsWhenHostProfileFails(t *testing.T) {
	broken := func(command string, _ []byte) ([]byte, int, bool) {
		return fakeError("internal"), 1, command == "engine host-profile"
	}
	assertFailedSteps(t, newFakeEngine("", broken), "engine host-profile")
}

func TestRunFailsOnAWrongRuntimeProtocol(t *testing.T) {
	future := answering([]string{"runtime info"}, func(string) []byte { return []byte(`{"protocol":2,"engine":"fake","sdk":"0.2.0"}`) })
	assertFailedSteps(t, newFakeEngine("", future), "runtime info")
}

func TestRunFailsWhenRuntimeRunLosesOutput(t *testing.T) {
	silent := answering([]string{"runtime run"}, func(string) []byte { return []byte(`{"exit":0,"stdout":"","stderr":"","truncated":false}`) })
	assertFailedSteps(t, newFakeEngine("", silent), "runtime run")
}

func TestRunFailsWhenAFileDoesNotRoundTrip(t *testing.T) {
	forgetful := answering([]string{"file get"}, func(string) []byte { return []byte(`{"contentBase64":"","exists":false}`) })
	assertFailedSteps(t, newFakeEngine("", forgetful), "file round trip")
}
