package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/support"
)

const (
	rawOutputLimit  = 2048
	exitFailure     = 1
	exitInvalid     = 64
	exitInternal    = 70
	codeInvalid     = "invalid_request"
	codeInternal    = "internal"
	codeUnsupported = "unsupported"
)

type decoder func([]byte) error

func as[T any]() decoder {
	return func(stdout []byte) error {
		_, err := support.DecodeStrict[T](bytes.NewReader(stdout))

		return err
	}
}

var replies = map[string]decoder{
	"engine info": as[responses.EngineInfo](), "logs forward": as[dtos.Empty](), "logs read": as[responses.LogsRead](),
	"runtime info": as[responses.RuntimeInfo](), "runtime run": as[responses.RuntimeRun](),
	"task start": as[dtos.Operation](), "task status": as[dtos.Operation](), "task log": as[responses.TaskLog](), "task stop": as[dtos.Operation](),
	"file put": as[dtos.Empty](), "file get": as[responses.FileGet](),
	"service start": as[dtos.Empty](), "service stop": as[dtos.Empty](), "service restart": as[dtos.Empty](), "service reload": as[dtos.Empty](), "service status": as[responses.ServiceStatus](),
}

func validate(decode decoder, stdout []byte) error {
	if !json.Valid(stdout) {
		return fmt.Errorf("stdout is not a single JSON document; raw output: %s", rawOutput(stdout))
	}
	if err := decode(stdout); err != nil {
		return fmt.Errorf("stdout is not a valid reply: %w", err)
	}

	return nil
}

func rawOutput(stdout []byte) string {
	if len(stdout) <= rawOutputLimit {
		return string(stdout)
	}
	cut := rawOutputLimit
	for cut > 0 && !utf8.RuneStart(stdout[cut]) {
		cut--
	}
	return string(stdout[:cut])
}

func errorCode(stdout []byte) (string, error) {
	var reply struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout, &reply); err != nil {
		return "", err
	}
	return reply.Error.Code, nil
}

func exitFor(code string) int {
	switch code {
	case codeInvalid:
		return exitInvalid
	case codeInternal:
		return exitInternal
	}
	return exitFailure
}

func checkErrorReply(stdout []byte, exit int) error {
	if err := validate(as[responses.Error](), stdout); err != nil {
		return err
	}
	code, err := errorCode(stdout)
	if err != nil {
		return err
	}
	if want := exitFor(code); want != exit {
		return fmt.Errorf("error code %q needs exit %d, got exit %d", code, want, exit)
	}
	return nil
}

func Check(command string, stdout []byte, exit int) error {
	decode, known := replies[command]
	if !known {
		if exit == 0 {
			return fmt.Errorf("command %q is not part of the runtime but was answered", command)
		}

		return checkErrorReply(stdout, exit)
	}
	switch exit {
	case 0:
		return validate(decode, stdout)
	case exitFailure, exitInvalid, exitInternal:
		return checkErrorReply(stdout, exit)
	}
	return fmt.Errorf("exit code %d is not allowed by the wrapper; raw output: %s", exit, rawOutput(stdout))
}
