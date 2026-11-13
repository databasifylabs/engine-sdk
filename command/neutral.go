package command

import (
	"context"
	"errors"
	"os"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/layout"
	"github.com/databasifylabs/engine-sdk/logship"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/units"
)

type handler func(ctx context.Context, input []byte) ([]byte, error)

func neutralHandlers(version string) map[string]handler {
	return map[string]handler{
		"engine info": func(_ context.Context, input []byte) ([]byte, error) {
			return Encode(engineInfo(version, input))
		},
		"logs forward": func(_ context.Context, input []byte) ([]byte, error) {
			request, err := Decode[requests.LogsForward](input)
			if err != nil {
				return nil, err
			}

			return Encode(dtos.Empty{}, forwardLogs(request.Destination))
		},
		"logs read": func(_ context.Context, input []byte) ([]byte, error) {
			request, err := Decode[requests.LogsRead](input)
			if err != nil {
				return nil, err
			}

			return Encode(readLogs(layout.FromEnv(), request))
		},
		"runtime info": func(_ context.Context, input []byte) ([]byte, error) {
			return Encode(runtimeInfo(version, input))
		},
		"runtime run": func(ctx context.Context, input []byte) ([]byte, error) {
			request, err := Decode[requests.RuntimeRun](input)
			if err != nil {
				return nil, err
			}

			return Encode(runScript(ctx, request))
		},
		"task start": func(_ context.Context, input []byte) ([]byte, error) {
			request, err := Decode[requests.TaskStart](input)
			if err != nil {
				return nil, err
			}

			return Encode(operationsRuntime().startTask(request, input))
		},
		"task status": answering(func(request requests.TaskKey) (dtos.Operation, error) {
			return operationsRuntime().taskStatus(request.Key)
		}),
		"task log": answering(func(request requests.TaskKey) (responses.TaskLog, error) {
			if _, err := operationsRuntime().taskStatus(request.Key); err != nil {
				return responses.TaskLog{}, err
			}

			return taskLog(layout.FromEnv().Home, request.Key)
		}),
		"task stop": answering(func(request requests.TaskKey) (dtos.Operation, error) {
			return operationsRuntime().stopTask(request.Key)
		}),
		"file put": answering(func(request requests.FilePut) (dtos.Empty, error) {
			return dtos.Empty{}, putFile(layout.FromEnv().Home, request)
		}),
		"file get": answering(func(request requests.FileGet) (responses.FileGet, error) {
			return getFile(layout.FromEnv().Home, request)
		}),
		"service start": answering(func(request requests.ServiceUnits) (dtos.Empty, error) {
			return controlServices(serviceRuntime(), "start", request.Units)
		}),
		"service stop": answering(func(request requests.ServiceUnits) (dtos.Empty, error) {
			return controlServices(serviceRuntime(), "stop", request.Units)
		}),
		"service restart": answering(func(request requests.ServiceUnits) (dtos.Empty, error) {
			return controlServices(serviceRuntime(), "restart", request.Units)
		}),
		"service reload": answering(func(dtos.Empty) (dtos.Empty, error) {
			return reloadServices(serviceRuntime())
		}),
		"service status": answering(func(request requests.ServiceUnits) (responses.ServiceStatus, error) {
			return serviceStatusOf(serviceRuntime(), request.Units)
		}),
	}
}

func engineInfo(version string, input []byte) (responses.EngineInfo, error) {
	if _, err := Decode[dtos.Empty](input); err != nil {
		return responses.EngineInfo{}, err
	}

	return responses.EngineInfo{
		Engine:  os.Getenv("DATABASIFY_ENGINE"),
		Line:    os.Getenv("DATABASIFY_LINE"),
		Version: os.Getenv("DATABASIFY_RELEASE"),
		Sdk:     version,
	}, nil
}

func forwardLogs(destination *dtos.LogDestination) error {
	paths := layout.FromEnv()
	if err := logship.Save(paths.Home, destination); err != nil {
		if errors.Is(err, logship.ErrScheme) {
			return Invalid(err.Error())
		}

		return err
	}

	return units.Control{Socket: paths.Socket()}.Do("daemon-reload")
}

func answering[Req, Rep any](method func(Req) (Rep, error)) handler {
	return func(_ context.Context, input []byte) ([]byte, error) {
		request, err := Decode[Req](input)
		if err != nil {
			return nil, err
		}

		return Encode(method(request))
	}
}
