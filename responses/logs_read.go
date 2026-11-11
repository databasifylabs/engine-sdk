package responses

type LogsRead struct {
	Units []string `json:"units"`
	Lines []string `json:"lines"`
}
