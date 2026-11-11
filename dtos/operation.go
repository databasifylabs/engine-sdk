package dtos

import (
	"time"

	"github.com/databasifylabs/engine-sdk/enums"
)

type Operation struct {
	Key        string               `json:"key" pattern:"^[A-Za-z0-9._-]{1,128}$"`
	Kind       string               `json:"kind"`
	State      enums.OperationState `json:"state"`
	Phase      string               `json:"phase"`
	Progress   *Progress            `json:"progress"`
	StartedAt  time.Time            `json:"startedAt"`
	FinishedAt *time.Time           `json:"finishedAt"`
	Error      *Error               `json:"error"`
}
