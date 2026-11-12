package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

const (
	taskKind     = "task"
	taskLogLines = 50
)

type lastLine struct {
	text string
}

func (l *lastLine) Write(chunk []byte) (int, error) {
	for line := range strings.SplitSeq(strings.TrimRight(string(chunk), "\n"), "\n") {
		if line != "" {
			l.text = line
		}
	}

	return len(chunk), nil
}

func (o operations) startTask(request requests.TaskStart, input []byte) (dtos.Operation, error) {
	return o.begin(taskKind, request.Key, input)
}

func (o operations) taskStatus(key string) (dtos.Operation, error) {
	if err := o.ensureTask(key); err != nil {
		return dtos.Operation{}, err
	}

	return o.status(key)
}

func (o operations) stopTask(key string) (dtos.Operation, error) {
	if err := o.ensureTask(key); err != nil {
		return dtos.Operation{}, err
	}

	return o.cancel(key)
}

func (o operations) ensureTask(key string) error {
	rec, found, err := o.load(key)
	if err != nil {
		return err
	}
	if !found || rec.Operation.Kind != taskKind {
		return NotFound("no task with key " + key)
	}

	return nil
}

func (o operations) runTask(ctx context.Context, key string, output io.Writer) error {
	rec, found, err := o.load(key)
	if err != nil || !found {
		return errors.Join(err, errors.New("no task with key "+key))
	}
	var request requests.TaskStart
	if err := json.Unmarshal(rec.Request, &request); err != nil {
		return o.finish(rec, err)
	}

	var failure lastLine
	exit, err := runScriptTo(ctx, requests.RuntimeRun{Script: request.Script, Env: request.Env, User: request.User}, output, io.MultiWriter(os.Stderr, &failure), false)
	if err == nil && exit != 0 {
		err = fmt.Errorf("script exited %d: %s", exit, failure.text)
	}

	return o.finish(rec, err)
}

func taskLog(home string, key string) (responses.TaskLog, error) {
	data, err := os.ReadFile(filepath.Join(home, "logs", unitOf(key)+".log"))
	if errors.Is(err, os.ErrNotExist) {
		return responses.TaskLog{}, nil
	}
	if err != nil {
		return responses.TaskLog{}, err
	}
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return responses.TaskLog{}, nil
	}
	lines := strings.SplitAfter(trimmed+"\n", "\n")
	lines = lines[:len(lines)-1]

	return responses.TaskLog{Lines: strings.Join(lines[max(len(lines)-taskLogLines, 0):], "")}, nil
}
