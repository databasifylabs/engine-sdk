package dtos

import "github.com/databasifylabs/engine-sdk/enums"

type Error struct {
	Code    enums.ErrorCode   `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}
