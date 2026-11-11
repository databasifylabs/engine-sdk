package responses

import "github.com/databasifylabs/engine-sdk/dtos"

type Error struct {
	Error dtos.Error `json:"error"`
}
