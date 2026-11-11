package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type UserCreate struct {
	Name       string       `json:"name" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"32"`
	Password   string       `json:"password" min:"1"`
	Grants     []dtos.Grant `json:"grants"`
	Encryption *string      `json:"encryption"`
}
