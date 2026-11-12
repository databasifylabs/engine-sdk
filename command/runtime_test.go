package command

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/databasifylabs/engine-sdk/requests"
)

func script(body string, timeout int64) requests.RuntimeRun {
	return requests.RuntimeRun{Script: body, Env: map[string]string{}, TimeoutSeconds: timeout}
}

func TestRunPassesTheScriptOnStdinAndReturnsOutput(t *testing.T) {
	request := script("echo \"$GREETING\"; echo oops >&2; exit 3", 5)
	request.Env["GREETING"] = "hi"

	reply, err := runScript(context.Background(), request)

	if err != nil || reply.Exit != 3 || reply.Stdout != "hi\n" || reply.Stderr != "oops\n" || reply.Truncated {
		t.Fatalf("%+v %v", reply, err)
	}
}

func TestRunStopsAtTheFirstFailingCommand(t *testing.T) {
	reply, _ := runScript(context.Background(), script("false | true\necho after", 5))

	if reply.Exit == 0 || strings.Contains(reply.Stdout, "after") {
		t.Fatalf("%+v", reply)
	}
}

func TestRunCapsHugeOutput(t *testing.T) {
	reply, _ := runScript(context.Background(), script("head -c 5000000 /dev/zero | tr '\\0' a", 10))

	if len(reply.Stdout) != 1<<20 || !reply.Truncated {
		t.Fatalf("len = %d, truncated = %v", len(reply.Stdout), reply.Truncated)
	}
}

func TestRunKillsTheProcessGroupAtTheTimeout(t *testing.T) {
	started := time.Now()

	reply, _ := runScript(context.Background(), script("sleep 30 & sleep 30", 1))

	if reply.Exit != 124 || time.Since(started) > 5*time.Second {
		t.Fatalf("exit = %d after %s", reply.Exit, time.Since(started))
	}
}

func TestRunReturnsWhenADetachedChildKeepsTheOutputOpen(t *testing.T) {
	started := time.Now()

	reply, _ := runScript(context.Background(), script("perl -e 'use POSIX; POSIX::setsid(); sleep 30' &\necho started", 2))

	if time.Since(started) > 8*time.Second || !strings.Contains(reply.Stdout, "started") {
		t.Fatalf("%+v after %s", reply, time.Since(started))
	}
}

func TestRunReportsSuccessWhenABackgroundChildOutlivesTheScript(t *testing.T) {
	started := time.Now()

	reply, _ := runScript(context.Background(), script("sleep 30 &\necho started", 10))

	if reply.Exit != 0 || reply.Stdout != "started\n" || time.Since(started) > 6*time.Second {
		t.Fatalf("%+v after %s", reply, time.Since(started))
	}
}
