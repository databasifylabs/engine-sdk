package host

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/databasifylabs/engine-sdk/ctl"
)

func Forward(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	container, err := engineContainer(execDocker{})
	if err != nil {
		fmt.Fprintf(stderr, "databasify: %v\n", err)

		return ctl.ExitUnavailable
	}
	forwarded := exec.Command("docker", append([]string{"exec", "--interactive", container, "databasify"}, args...)...)
	forwarded.Stdin, forwarded.Stdout, forwarded.Stderr = stdin, stdout, stderr
	err = forwarded.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	if err != nil {
		fmt.Fprintf(stderr, "databasify: %v\n", err)

		return ctl.ExitUnavailable
	}

	return 0
}

func engineContainer(docker Docker) (string, error) {
	output, err := docker.Run("ps", "--filter", "label="+engineLabelKey, "--format", "{{.Names}}")
	if err != nil {
		return "", fmt.Errorf("cannot list containers: %w", err)
	}
	names := strings.Fields(output)
	if len(names) == 0 {
		return "", errors.New("no engine container is running")
	}
	if len(names) > 1 {
		return "", fmt.Errorf("more than one engine container is running (%s), remove all but one", strings.Join(names, ", "))
	}

	return names[0], nil
}
