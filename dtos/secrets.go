package dtos

type Secrets struct {
	Admin       *string `json:"admin"`
	Replication *string `json:"replication"`
	Agent       *string `json:"agent"`
}
