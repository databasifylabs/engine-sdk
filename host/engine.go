package host

import (
	"fmt"
	"strings"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/support"
)

func runningEngine(docker Docker) (string, error) {
	container, err := engineContainer(docker)
	if err != nil {
		return "", command.Unavailable(err.Error())
	}

	return container, nil
}

func ask[T any](docker Docker, container string, group string, verb string) (T, error) {
	output, err := docker.RunWithInput("{}", "exec", "--interactive", container, "databasify", group, verb)
	if err != nil {
		var nothing T

		return nothing, fmt.Errorf("%s (%s) does not answer %s %s: %w", container, imageOf(docker, container), group, verb, err)
	}

	return support.DecodeStrict[T](strings.NewReader(output))
}

func imageOf(docker Docker, container string) string {
	image, err := docker.Run("container", "inspect", "--format", "{{.Config.Image}}", container)
	if err != nil {
		return "unknown image"
	}

	return image
}

func engineNameIn(docker Docker, container string) (string, error) {
	info, err := ask[responses.EngineInfo](docker, container, "engine", "info")

	return info.Engine, err
}

func tuneHost(docker Docker, system system, profile responses.EngineHostProfile) (responses.HostTune, error) {
	container, err := runningEngine(docker)
	if err != nil {
		return responses.HostTune{}, err
	}

	return tuneEngineIn(docker, system, container, profile)
}

func tuneEngineIn(docker Docker, system system, container string, profile responses.EngineHostProfile) (responses.HostTune, error) {
	engine, err := engineNameIn(docker, container)
	if err != nil {
		return responses.HostTune{}, err
	}

	return system.tune(engine, profile)
}

func restoreHost(docker Docker, system system) error {
	container, err := runningEngine(docker)
	if err != nil {
		return err
	}
	engine, err := engineNameIn(docker, container)
	if err != nil {
		return err
	}

	return system.restore(engine)
}
