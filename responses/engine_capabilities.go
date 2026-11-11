package responses

import "github.com/databasifylabs/engine-sdk/dtos"

type EngineCapabilities struct {
	Commands                  []string   `json:"commands"`
	PasswordEncryptions       []string   `json:"passwordEncryptions"`
	DefaultPasswordEncryption *string    `json:"defaultPasswordEncryption"`
	ReservedUsers             []string   `json:"reservedUsers"`
	ReservedDatabases         []string   `json:"reservedDatabases"`
	Ports                     dtos.Ports `json:"ports"`
}
