package enums

type ErrorCode string

const (
	ErrorCodeInvalidRequest ErrorCode = "invalid_request"
	ErrorCodeUnsupported    ErrorCode = "unsupported"
	ErrorCodeNotFound       ErrorCode = "not_found"
	ErrorCodeAlreadyExists  ErrorCode = "already_exists"
	ErrorCodeNotPrimary     ErrorCode = "not_primary"
	ErrorCodeBusy           ErrorCode = "busy"
	ErrorCodeUnavailable    ErrorCode = "unavailable"
	ErrorCodeInternal       ErrorCode = "internal"
)

func (r ErrorCode) Valid() bool {
	switch r {
	case ErrorCodeInvalidRequest, ErrorCodeUnsupported, ErrorCodeNotFound, ErrorCodeAlreadyExists, ErrorCodeNotPrimary, ErrorCodeBusy, ErrorCodeUnavailable, ErrorCodeInternal:
		return true
	}
	return false
}
