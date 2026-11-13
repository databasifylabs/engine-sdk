package conformance

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/support"
)

const (
	runtimeProtocol = 1
	conformanceFile = "state/conformance-check"
	commandTimeout  = 60 * time.Second
	taskWait        = 30 * time.Second
	secret          = "p'a\"s\\s wörd"
)

type Failure struct {
	Step string
	Err  error
}

type step struct {
	name string
	run  func() error
}

type lifecycle struct {
	ctx      context.Context
	runner   Runner
	marker   string
	failures []Failure
}

func Run(t *testing.T, r Runner) {
	t.Helper()
	for _, failure := range Failures(context.Background(), r) {
		t.Errorf("%s: %v", failure.Step, failure.Err)
	}
}

func Failures(ctx context.Context, r Runner) []Failure {
	l := &lifecycle{ctx: ctx, runner: r, marker: randomHex(4)}
	for _, s := range l.steps() {
		if err := s.run(); err != nil {
			l.failures = append(l.failures, Failure{Step: s.name, Err: err})
		}
	}

	return l.failures
}

func randomHex(size int) string {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}

	return hex.EncodeToString(buffer)
}

func (l *lifecycle) steps() []step {
	return []step{
		{"engine info", l.checkEngineInfo},
		{"runtime info", l.checkRuntimeInfo},
		{"runtime run", l.checkRuntimeRun},
		{"runtime run keeps env secrets out of the reply", l.checkSecretInEnv},
		{"file round trip", l.checkFileRoundTrip},
		{"task runs to success", l.checkTask},
		{"engine info array input", func() error { return l.expectInvalid("engine info", `[]`) }},
		{"engine commands are unsupported", func() error {
			return l.expectError("node status", dtos.Empty{}, exitFor(codeUnsupported), codeUnsupported)
		}},
	}
}

func (l *lifecycle) exec(command string, body []byte) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(l.ctx, commandTimeout)
	defer cancel()
	stdout, exit, err := l.runner.Run(ctx, strings.Fields(command), body)
	if err != nil {
		return nil, 0, fmt.Errorf("runner failed: %w", err)
	}
	if err := Check(command, stdout, exit); err != nil {
		return nil, 0, err
	}

	return stdout, exit, nil
}

func (l *lifecycle) call(command string, request any) ([]byte, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	stdout, exit, err := l.exec(command, body)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return nil, fmt.Errorf("exit = %d, want 0; stdout: %s", exit, rawOutput(stdout))
	}

	return stdout, nil
}

func callReply[T any](l *lifecycle, command string, request any) (T, error) {
	var zero T
	stdout, err := l.call(command, request)
	if err != nil {
		return zero, err
	}

	return support.DecodeStrict[T](bytes.NewReader(stdout))
}

func (l *lifecycle) expectInvalid(command, body string) error {
	stdout, exit, err := l.exec(command, []byte(body))
	if err != nil {
		return err
	}
	code, err := errorCode(stdout)
	if err != nil {
		return err
	}
	if code != "invalid_request" || exit != exitFor("invalid_request") {
		return fmt.Errorf("got %s with exit %d, want invalid_request", code, exit)
	}

	return nil
}

func (l *lifecycle) expectError(command string, request any, wantExit int, wantCode string) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	stdout, exit, err := l.exec(command, body)
	if err != nil {
		return err
	}
	code, err := errorCode(stdout)
	if err != nil {
		return err
	}
	if exit != wantExit || code != wantCode {
		return fmt.Errorf("got %s with exit %d, want %s with exit %d", code, exit, wantCode, wantExit)
	}

	return nil
}

func (l *lifecycle) checkEngineInfo() error {
	reply, err := callReply[responses.EngineInfo](l, "engine info", dtos.Empty{})
	if err != nil {
		return err
	}
	if reply.Engine == "" || reply.Sdk == "" {
		return fmt.Errorf("engine info reports engine %q and sdk %q, want both", reply.Engine, reply.Sdk)
	}

	return nil
}

func (l *lifecycle) checkRuntimeInfo() error {
	reply, err := callReply[responses.RuntimeInfo](l, "runtime info", dtos.Empty{})
	if err != nil {
		return err
	}
	if reply.Protocol != runtimeProtocol {
		return fmt.Errorf("runtime info reports protocol %d, want %d", reply.Protocol, runtimeProtocol)
	}

	return nil
}

func (l *lifecycle) checkRuntimeRun() error {
	reply, err := callReply[responses.RuntimeRun](l, "runtime run", requests.RuntimeRun{Script: "echo ok", Env: map[string]string{}, TimeoutSeconds: 10})
	if err != nil {
		return err
	}
	if reply.Exit != 0 || reply.Stdout != "ok\n" {
		return fmt.Errorf("runtime run of echo ok replied exit %d, stdout %q", reply.Exit, reply.Stdout)
	}

	return nil
}

func (l *lifecycle) checkSecretInEnv() error {
	stdout, err := l.call("runtime run", requests.RuntimeRun{Script: `[ -n "$SECRET" ] && echo set`, Env: map[string]string{"SECRET": secret}, TimeoutSeconds: 10})
	if err != nil {
		return err
	}
	escaped, _ := json.Marshal(secret)
	if bytes.Contains(stdout, []byte(secret)) || bytes.Contains(stdout, escaped[1:len(escaped)-1]) {
		return errors.New("the reply echoes a secret passed in env")
	}

	return nil
}

func (l *lifecycle) checkFileRoundTrip() error {
	content := base64.StdEncoding.EncodeToString([]byte(l.marker))
	if _, err := l.call("file put", requests.FilePut{Path: conformanceFile, ContentBase64: content, Mode: 0o600}); err != nil {
		return err
	}
	reply, err := callReply[responses.FileGet](l, "file get", requests.FileGet{Path: conformanceFile})
	if err != nil {
		return err
	}
	if !reply.Exists || reply.ContentBase64 != content {
		return fmt.Errorf("file get of %s returned %+v, want what file put wrote", conformanceFile, reply)
	}

	return nil
}

func (l *lifecycle) checkTask() error {
	key := "conformance-" + l.marker
	if _, err := l.call("task start", requests.TaskStart{Key: key, Script: "echo done", Env: map[string]string{}}); err != nil {
		return err
	}
	deadline := time.Now().Add(taskWait)
	for time.Now().Before(deadline) {
		operation, err := callReply[dtos.Operation](l, "task status", requests.TaskKey{Key: key})
		if err != nil {
			return err
		}
		if operation.State == enums.OperationStateSucceeded {
			return nil
		}
		if operation.State != enums.OperationStateRunning {
			return fmt.Errorf("task ended %s", operation.State)
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("task did not finish within %s", taskWait)
}
