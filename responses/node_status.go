package responses

import "github.com/databasifylabs/engine-sdk/enums"

type NodeStatus struct {
	Role    enums.NodeRole `json:"role"`
	Version *string        `json:"version"`
}
