package enums

type OperationState string

const (
	OperationStateRunning     OperationState = "running"
	OperationStateSucceeded   OperationState = "succeeded"
	OperationStateFailed      OperationState = "failed"
	OperationStateCancelled   OperationState = "cancelled"
	OperationStateInterrupted OperationState = "interrupted"
)

func (r OperationState) Valid() bool {
	switch r {
	case OperationStateRunning, OperationStateSucceeded, OperationStateFailed, OperationStateCancelled, OperationStateInterrupted:
		return true
	}
	return false
}
