package command

import (
	"fmt"

	"github.com/databasifylabs/engine-sdk/ctl"
	"github.com/databasifylabs/engine-sdk/enums"
)

type Failure struct {
	Code    enums.ErrorCode
	Message string
	Details map[string]string
}

func (f Failure) Error() string {
	return f.Message
}

func (f Failure) ExitCode() int {
	switch f.Code {
	case enums.ErrorCodeInvalidRequest:
		return ctl.ExitUsage
	case enums.ErrorCodeInternal:
		return ctl.ExitInternal
	}

	return 1
}

func Unavailable(message string) Failure {
	return Failure{Code: enums.ErrorCodeUnavailable, Message: message}
}

func NotFound(message string) Failure {
	return Failure{Code: enums.ErrorCodeNotFound, Message: message}
}

func Invalid(message string) Failure {
	return Failure{Code: enums.ErrorCodeInvalidRequest, Message: message}
}

func Unsupported(command string) Failure {
	return Failure{Code: enums.ErrorCodeUnsupported, Message: fmt.Sprintf("this engine does not implement %q", command)}
}

func internal(message string) Failure {
	return Failure{Code: enums.ErrorCodeInternal, Message: message}
}
