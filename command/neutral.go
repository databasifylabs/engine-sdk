package command

import (
	"context"
	"os"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/layout"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

type handler func(ctx context.Context, input []byte) ([]byte, error)

func neutralHandlers(version string) map[string]handler {
	return map[string]handler{
		"engine info": func(_ context.Context, input []byte) ([]byte, error) {
			return Encode(engineInfo(version, input))
		},
		"logs read": func(_ context.Context, input []byte) ([]byte, error) {
			request, err := Decode[requests.LogsRead](input)
			if err != nil {
				return nil, err
			}

			return Encode(readLogs(layout.FromEnv(), request))
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

func answering[Req, Rep any](method func(Req) (Rep, error)) handler {
	return func(_ context.Context, input []byte) ([]byte, error) {
		request, err := Decode[Req](input)
		if err != nil {
			return nil, err
		}

		return Encode(method(request))
	}
}
