package command

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
)

func taskOf(key string, body string) (requests.TaskStart, []byte) {
	request := requests.TaskStart{Key: key, Script: body, Env: map[string]string{}}
	input, _ := json.Marshal(request)

	return request, input
}

func TestTaskStartRunsOneUnitPerKey(t *testing.T) {
	o, supervisor := testOperations(t)
	request, input := taskOf("migration.copy.1", "echo hi")

	first, err := o.startTask(request, input)
	again, _ := o.startTask(request, input)

	if err != nil || len(supervisor.started) != 1 || first.Key != again.Key || first.Kind != "task" || supervisor.started[0].Unit != "databasify-op-migration.copy.1" {
		t.Fatalf("%+v %+v %d %v", first, again, len(supervisor.started), err)
	}
}

func TestATaskScriptFailureFailsTheOperation(t *testing.T) {
	o, _ := testOperations(t)
	request, input := taskOf("restore.copy.1", "echo bad >&2; exit 7")
	_, _ = o.startTask(request, input)

	_ = o.runTask(context.Background(), "restore.copy.1", os.Stdout)
	operation, _ := o.status("restore.copy.1")

	if operation.State != enums.OperationStateFailed || operation.Error == nil || operation.Error.Message != "script exited 7: bad" {
		t.Fatalf("%+v", operation)
	}
}

func TestAFinishedTaskKeyIsReturnedNotRerun(t *testing.T) {
	o, supervisor := testOperations(t)
	request, input := taskOf("rejoin.sync.1", "true")
	_, _ = o.startTask(request, input)
	_ = o.runTask(context.Background(), "rejoin.sync.1", os.Stdout)

	again, _ := o.startTask(request, input)

	if again.State != enums.OperationStateSucceeded || len(supervisor.started) != 1 {
		t.Fatalf("%+v %d", again, len(supervisor.started))
	}
}

func TestTaskLogOfAnEmptyLogIsEmpty(t *testing.T) {
	home := t.TempDir()
	_ = os.MkdirAll(filepath.Join(home, "logs"), 0o755)
	_ = os.WriteFile(filepath.Join(home, "logs", "databasify-op-copy.1.log"), nil, 0o600)

	if reply, err := taskLog(home, "copy.1"); err != nil || reply.Lines != "" {
		t.Fatalf("%q %v", reply.Lines, err)
	}
}

func TestTaskLogReturnsTheLastFiftyLines(t *testing.T) {
	home := t.TempDir()
	lines := []string{}
	for index := range 60 {
		lines = append(lines, "line "+string(rune('a'+index%26)))
	}
	_ = os.MkdirAll(filepath.Join(home, "logs"), 0o755)
	_ = os.WriteFile(filepath.Join(home, "logs", "databasify-op-copy.1.log"), []byte(strings.Join(lines, "\n")+"\n"), 0o600)

	reply, err := taskLog(home, "copy.1")

	if err != nil || strings.Count(reply.Lines, "\n") != 50 || !strings.HasSuffix(reply.Lines, lines[59]+"\n") {
		t.Fatalf("%q %v", reply.Lines, err)
	}
}

func TestATaskScriptSharesTheUnitsProcessGroup(t *testing.T) {
	o, _ := testOperations(t)
	request, input := taskOf("group.check.1", "ps -o pgid= -p $$")
	_, _ = o.startTask(request, input)
	var output strings.Builder

	_ = o.runTask(context.Background(), "group.check.1", &output)

	if strings.TrimSpace(output.String()) != strconv.Itoa(syscall.Getpgrp()) {
		t.Fatalf("task group %q, unit group %d", output.String(), syscall.Getpgrp())
	}
}

func TestTaskCommandsIgnoreOtherOperations(t *testing.T) {
	o, _ := testOperations(t)
	_, _ = o.begin("other", "ready-1", []byte(readyInput))

	_, statusErr := o.taskStatus("ready-1")
	_, stopErr := o.stopTask("ready-1")

	var failure Failure
	if !errors.As(statusErr, &failure) || failure.Code != enums.ErrorCodeNotFound || !errors.As(stopErr, &failure) || failure.Code != enums.ErrorCodeNotFound {
		t.Fatalf("%v %v", statusErr, stopErr)
	}
}

func TestAFinishedTaskForgetsItsSecrets(t *testing.T) {
	o, _ := testOperations(t)
	request := requests.TaskStart{Key: "backup.1", Script: `test -n "$PASSWORD"`, Env: map[string]string{"PASSWORD": "s3cret-value"}}
	input, _ := json.Marshal(request)
	_, _ = o.startTask(request, input)
	running, _ := os.ReadFile(o.path("backup.1"))

	_ = o.runTask(context.Background(), "backup.1", os.Stdout)
	finished, _ := os.ReadFile(o.path("backup.1"))
	operation, _ := o.status("backup.1")

	if !strings.Contains(string(running), "s3cret-value") {
		t.Fatalf("the running task lost its request: %s", running)
	}
	if strings.Contains(string(finished), "s3cret-value") || operation.State != enums.OperationStateSucceeded {
		t.Fatalf("state %s, record %s", operation.State, finished)
	}
}
