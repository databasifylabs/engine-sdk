package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type MigrationStart struct {
	Source      dtos.MigrationSource `json:"source"`
	Databases   []string             `json:"databases" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
	Continuous  bool                 `json:"continuous"`
	Owner       *string              `json:"owner" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
	Publication *string              `json:"publication" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
}
