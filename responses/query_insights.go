package responses

import "github.com/databasifylabs/engine-sdk/dtos"

type QueryInsights struct {
	Statements []dtos.Statement `json:"statements"`
	Reason     *string          `json:"reason"`
}
