package host

import (
	"io"
	"time"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

const engineWait = 60 * time.Second

type hostHandler func(input []byte) ([]byte, error)

var hostCommands = map[string]hostHandler{
	"tune": func(input []byte) ([]byte, error) {
		request, err := command.Decode[requests.HostTune](input)
		if err != nil {
			return nil, err
		}

		return command.Encode(tuneHost(execDocker{}, hostSystem, request.HostProfile))
	},
	"restore": func(input []byte) ([]byte, error) {
		if _, err := command.Decode[dtos.Empty](input); err != nil {
			return nil, err
		}

		return command.Encode(dtos.Empty{}, restoreHost(execDocker{}, hostSystem))
	},
	"deploy": func(input []byte) ([]byte, error) {
		request, err := command.Decode[requests.HostDeploy](input)
		if err != nil {
			return nil, err
		}

		return command.Encode(deployAndSettle(execDocker{}, srvRoot, hostSystem, request))
	},
	"remove": func(input []byte) ([]byte, error) {
		request, err := command.Decode[requests.HostRemove](input)
		if err != nil {
			return nil, err
		}

		return command.Encode(dtos.Empty{}, remove(execDocker{}, srvRoot, hostSystem.restore, request))
	},
}

func IsHostCommand(args []string) bool {
	if len(args) != 2 || args[0] != "host" {
		return false
	}
	_, known := hostCommands[args[1]]

	return known
}

func RunHostCommand(args []string, stdin io.Reader, stdout io.Writer) int {
	return command.Answer(stdin, stdout, hostCommands[args[1]])
}

func deployAndSettle(docker Docker, srv string, system system, request requests.HostDeploy) (responses.HostDeploy, error) {
	reply, err := deploy(docker, srv, request)
	if err != nil {
		return responses.HostDeploy{}, err
	}
	if err := waitForEngine(docker, reply.Container); err != nil {
		return responses.HostDeploy{}, err
	}
	_, err = tuneEngineIn(docker, system, reply.Container, request.HostProfile)

	return reply, err
}

func waitForEngine(docker Docker, container string) error {
	deadline := time.Now().Add(engineWait)
	for {
		_, err := docker.Run("exec", container, "databasify", "ctl", "status")
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return command.Unavailable(container + " did not answer within a minute: " + err.Error())
		}
		time.Sleep(time.Second)
	}
}
