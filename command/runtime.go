package command

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

const (
	outputCap    = 1 << 20
	timedOutExit = 124
	pipeGrace    = 2 * time.Second
)

type cappedOutput struct {
	data      []byte
	truncated bool
}

func (c *cappedOutput) Write(chunk []byte) (int, error) {
	room := outputCap - len(c.data)
	if len(chunk) > room {
		c.truncated = true
		c.data = append(c.data, chunk[:max(room, 0)]...)

		return len(chunk), nil
	}
	c.data = append(c.data, chunk...)

	return len(chunk), nil
}

func runScript(ctx context.Context, request requests.RuntimeRun) (responses.RuntimeRun, error) {
	var stdout, stderr cappedOutput
	exit, err := runScriptTo(ctx, request, &stdout, &stderr, true)

	return responses.RuntimeRun{Exit: exit, Stdout: string(stdout.data), Stderr: string(stderr.data), Truncated: stdout.truncated || stderr.truncated}, err
}

func runScriptTo(ctx context.Context, request requests.RuntimeRun, stdout io.Writer, stderr io.Writer, ownGroup bool) (int64, error) {
	shell := exec.Command("bash", "-euo", "pipefail", "-s")
	shell.Stdin = strings.NewReader(request.Script)
	shell.Env = environmentOf(request.Env)
	shell.SysProcAttr = &syscall.SysProcAttr{Setpgid: ownGroup}
	shell.WaitDelay = pipeGrace
	shell.Stdout, shell.Stderr = stdout, stderr
	if err := runAs(shell, request.User); err != nil {
		return 0, err
	}
	if err := shell.Start(); err != nil {
		return 0, err
	}

	ctx, cancel := deadlineOf(ctx, request.TimeoutSeconds)
	defer cancel()
	finished := make(chan struct{})
	go killWhenDone(ctx, shell.Process.Pid, ownGroup, finished)
	waitErr := shell.Wait()
	close(finished)

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return timedOutExit, nil
	}

	return exitCodeOf(shell.ProcessState, waitErr)
}

func environmentOf(values map[string]string) []string {
	environment := []string{"PATH=" + os.Getenv("PATH")}
	for name, value := range values {
		environment = append(environment, name+"="+value)
	}

	return environment
}

func runAs(shell *exec.Cmd, name *string) error {
	if name == nil {
		return nil
	}
	account, err := user.Lookup(*name)
	if err != nil {
		return Invalid("no user " + *name)
	}
	uid, _ := strconv.ParseUint(account.Uid, 10, 32)
	gid, _ := strconv.ParseUint(account.Gid, 10, 32)
	shell.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}

	return nil
}

func deadlineOf(ctx context.Context, seconds int64) (context.Context, context.CancelFunc) {
	if seconds <= 0 {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
}

func killWhenDone(ctx context.Context, pid int, ownGroup bool, finished <-chan struct{}) {
	select {
	case <-ctx.Done():
		if ownGroup {
			pid = -pid
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
	case <-finished:
	}
}

func exitCodeOf(state *os.ProcessState, err error) (int64, error) {
	if state == nil {
		return 0, err
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int64(status.Signal()), nil
	}

	return int64(state.ExitCode()), nil
}
