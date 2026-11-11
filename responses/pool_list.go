package responses

import "github.com/databasifylabs/engine-sdk/dtos"

type PoolList struct {
	Pools []dtos.Pool `json:"pools"`
}
