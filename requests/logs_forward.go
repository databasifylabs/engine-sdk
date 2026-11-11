package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type LogsForward struct {
	Destination *dtos.LogDestination `json:"destination"`
}
