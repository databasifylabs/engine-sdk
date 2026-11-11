package conformance

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
)

type Runner interface {
	Run(ctx context.Context, argv []string, stdin []byte) (stdout []byte, exit int, err error)
}

type DockerRunner struct {
	Container string
	Stderr    io.Writer
}

func (r DockerRunner) Run(ctx context.Context, argv []string, stdin []byte) ([]byte, int, error) {
	args := append([]string{"exec", "-i", r.Container, "databasify"}, argv...)
	command := exec.CommandContext(ctx, "docker", args...)
	command.Stdin = bytes.NewReader(stdin)
	command.Stderr = r.Stderr
	var stdout bytes.Buffer
	command.Stdout = &stdout

	err := command.Run()
	if ctx.Err() != nil && err != nil {
		return nil, 0, ctx.Err()
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return stdout.Bytes(), exitError.ExitCode(), nil
	}
	if err != nil {
		return nil, 0, err
	}
	return stdout.Bytes(), 0, nil
}
