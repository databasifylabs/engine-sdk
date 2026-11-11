package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type UserGrant struct {
	Name        string       `json:"name" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"32"`
	Grants      []dtos.Grant `json:"grants"`
	Replication bool         `json:"replication"`
}
