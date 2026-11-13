package requests

import "github.com/databasifylabs/engine-sdk/responses"

type HostTune struct {
	HostProfile responses.EngineHostProfile `json:"hostProfile"`
}
