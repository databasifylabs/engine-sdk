package responses

import "github.com/databasifylabs/engine-sdk/dtos"

type AccountList struct {
	Users     []string                `json:"users"`
	Databases []string                `json:"databases"`
	Grants    map[string][]dtos.Grant `json:"grants"`
}
