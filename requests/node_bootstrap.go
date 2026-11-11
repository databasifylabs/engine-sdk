package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type NodeBootstrap struct {
	Secrets dtos.Secrets         `json:"secrets"`
	Account *dtos.DefaultAccount `json:"account"`
}
