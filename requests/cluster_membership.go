package requests

import "github.com/databasifylabs/engine-sdk/dtos"

type ClusterMembership struct {
	Members []dtos.Member `json:"members"`
}
