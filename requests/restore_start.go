package requests

import (
	"time"

	"github.com/databasifylabs/engine-sdk/dtos"
)

type RestoreStart struct {
	Key     string       `json:"key" pattern:"^[A-Za-z0-9._-]{1,128}$"`
	Storage dtos.Storage `json:"storage"`
	Target  time.Time    `json:"target"`
}
