package responses

type EngineInfo struct {
	Engine  string `json:"engine"`
	Line    string `json:"line"`
	Version string `json:"version"`
	Sdk     string `json:"sdk"`
}
