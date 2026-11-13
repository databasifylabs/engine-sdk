package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/support"
)

var commandWord = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)

func Decode[T any](input []byte) (T, error) {
	value, err := support.DecodeStrict[T](bytes.NewReader(input))
	if err != nil {
		return value, Invalid(err.Error())
	}
	if checked, ok := any(value).(interface{ Check() error }); ok {
		if err := checked.Check(); err != nil {
			return value, Invalid(err.Error())
		}
	}

	return value, nil
}

func IsWellFormed(argv []string) bool {
	return len(argv) == 2 && commandWord.MatchString(argv[0]) && commandWord.MatchString(argv[1])
}

func Run(ctx context.Context, version string, argv []string, stdin io.Reader, stdout io.Writer) int {
	return Answer(stdin, stdout, func(input []byte) ([]byte, error) {
		return dispatch(ctx, version, nameOf(argv), input)
	})
}

func Answer(stdin io.Reader, stdout io.Writer, call func(input []byte) ([]byte, error)) int {
	input, err := io.ReadAll(stdin)
	if err != nil {
		return reply(stdout, nil, internal(err.Error()))
	}
	encoded, err := invoke(func() ([]byte, error) { return call(input) })

	return reply(stdout, encoded, err)
}

func nameOf(argv []string) string {
	return argv[0] + " " + argv[1]
}

func dispatch(ctx context.Context, version string, name string, input []byte) ([]byte, error) {
	if handler, found := neutralHandlers(version)[name]; found {
		return handler(ctx, input)
	}

	return nil, Unsupported(name)
}

func Encode[T any](value T, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	encoded, err := support.Encode(value)
	if err != nil {
		return nil, internal(err.Error())
	}

	return encoded, nil
}

func invoke(call func() ([]byte, error)) (encoded []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			encoded, err = nil, internal(fmt.Sprint("panic: ", recovered))
		}
	}()

	return call()
}

func reply(stdout io.Writer, encoded []byte, err error) int {
	if err != nil {
		return replyFailure(stdout, failureOf(err))
	}
	if encoded == nil {
		return replyFailure(stdout, internal("handler returned no reply"))
	}
	write(stdout, encoded)

	return 0
}

func replyFailure(stdout io.Writer, failure Failure) int {
	encoded, err := support.Encode(responses.Error{Error: dtos.Error{Code: failure.Code, Message: failure.Message, Details: failure.Details}})
	if err != nil {
		failure = internal(err.Error())
		encoded = []byte(`{"error":{"code":"internal","message":"cannot encode the error","details":{}}}`)
	}
	write(stdout, encoded)

	return failure.ExitCode()
}

func failureOf(err error) Failure {
	var failure Failure
	if errors.As(err, &failure) {
		return failure
	}

	return internal(err.Error())
}

func write(stdout io.Writer, encoded []byte) {
	_, _ = stdout.Write(append(encoded, '\n'))
}
