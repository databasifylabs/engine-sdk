package responses

type ClusterQuorum struct {
	Total  int64 `json:"total"`
	Online int64 `json:"online"`
}
