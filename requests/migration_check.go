package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type MigrationCheck struct {
	Source    dtos.MigrationSource `json:"source"`
	Databases []string             `json:"databases" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
}
