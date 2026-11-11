package host

import (
	"fmt"
	"os/exec"
	"strings"
)

type Docker interface {
	Run(args ...string) (string, error)
	RunWithInput(input string, args ...string) (string, error)
}

type execDocker struct{}

func (execDocker) Run(args ...string) (string, error) {
	output, err := exec.Command("docker", args...).CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		return text, fmt.Errorf("docker %s: %w: %s", args[0], err, text)
	}

	return text, nil
}

func (execDocker) RunWithInput(input string, args ...string) (string, error) {
	var stderr strings.Builder
	docker := exec.Command("docker", args...)
	docker.Stdin, docker.Stderr = strings.NewReader(input), &stderr
	output, err := docker.Output()
	if err != nil {
		return string(output), fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()+"\n"+string(output)))
	}

	return string(output), nil
}
