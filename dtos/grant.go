package dtos

import "github.com/databasifylabs/engine-sdk/enums"

type Grant struct {
	Database string       `json:"database" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
	Access   enums.Access `json:"access"`
}
