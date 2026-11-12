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
	"engine info": as[responses.EngineInfo](), "engine capabilities": as[responses.EngineCapabilities](), "engine host-profile": as[responses.EngineHostProfile](),
	"operation status": as[dtos.Operation](), "operation cancel": as[dtos.Operation](),
	"node status": as[responses.NodeStatus](), "node configure": as[responses.NodeConfigure](), "node start": as[dtos.Empty](), "node restart": as[dtos.Empty](),
	"node wake": as[dtos.Empty](), "node ready": as[dtos.Operation](), "node bootstrap": as[dtos.Empty](),
	"node metrics": as[responses.NodeMetrics](), "config apply": as[dtos.Empty](), "config sync": as[responses.ConfigSync](),
	"tls reload": as[dtos.Empty](), "tls install": as[dtos.Empty](), "account list": as[responses.AccountList](), "database create": as[responses.DatabaseCreate](),
	"database drop": as[responses.DatabaseDrop](), "user create": as[responses.UserCreate](), "user set-password": as[dtos.Empty](),
	"user grant": as[dtos.Empty](), "user drop": as[responses.UserDrop](), "cluster enable": as[dtos.Operation](),
	"cluster disable": as[responses.ClusterDisable](), "cluster failover": as[dtos.Empty](),
	"cluster force-quorum": as[responses.ClusterForceQuorum](), "cluster quorum": as[responses.ClusterQuorum](),
	"cluster position": as[responses.ClusterPosition](), "member admit": as[dtos.Empty](), "member promote": as[dtos.Empty](),
	"member forget": as[dtos.Empty](), "archive status": as[responses.ArchiveStatus](), "restore start": as[dtos.Operation](),
	"migration check": as[responses.MigrationCheck](), "migration start": as[dtos.Empty](),
	"migration progress": as[responses.MigrationProgress](), "migration stop": as[dtos.Empty](),
	"logs forward": as[dtos.Empty](), "logs read": as[responses.LogsRead](), "node reset": as[dtos.Empty](), "query insights": as[responses.QueryInsights](),
	"cluster rejoin": as[dtos.Operation](), "cluster membership": as[dtos.Empty](),
	"config fields": as[responses.ConfigFields](), "config show": as[responses.ConfigShow](), "config tune": as[responses.ConfigTune](),
	"pool list": as[responses.PoolList](), "pool create": as[dtos.Empty](), "pool update": as[dtos.Empty](), "pool delete": as[dtos.Empty](),
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
		return fmt.Errorf("command %q is not part of the wrapper", command)
	}
	switch exit {
	case 0:
		return validate(decode, stdout)
	case exitFailure, exitInvalid, exitInternal:
		return checkErrorReply(stdout, exit)
	}
	return fmt.Errorf("exit code %d is not allowed by the wrapper; raw output: %s", exit, rawOutput(stdout))
}
