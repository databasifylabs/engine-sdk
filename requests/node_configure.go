package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type NodeConfigure struct {
	Clustered  bool                  `json:"clustered"`
	ReadOnly   bool                  `json:"readOnly"`
	Ip         string                `json:"ip" pattern:"^[0-9A-Fa-f:.]{2,45}$"`
	DatabaseId string                `json:"databaseId"`
	Members    []dtos.Member         `json:"members"`
	Settings   map[string]string     `json:"settings"`
	Secrets    dtos.Secrets          `json:"secrets"`
	Archive    *dtos.ArchiveSettings `json:"archive"`
	Primary    *string               `json:"primary" pattern:"^[0-9A-Fa-f:.]{2,45}$"`
}
