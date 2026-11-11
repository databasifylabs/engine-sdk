package dtos

import "github.com/databasifylabs/engine-sdk/enums"

type LogDestination struct {
	Kind      enums.LogDestinationKind `json:"kind"`
	Endpoint  string                   `json:"endpoint" pattern:"^(tcp|tls|https)://[A-Za-z0-9.-]+:[0-9]{1,5}(/[A-Za-z0-9._/-]*)?$"`
	Index     *string                  `json:"index" pattern:"^[a-z0-9][a-z0-9._-]{0,99}$"`
	Username  *string                  `json:"username" max:"128"`
	Password  *string                  `json:"password" max:"256"`
	VerifyTls bool                     `json:"verifyTls"`
}
