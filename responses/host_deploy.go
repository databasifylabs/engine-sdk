package responses

type HostDeploy struct {
	Engine    string `json:"engine"`
	Container string `json:"container"`
	Created   bool   `json:"created"`
	Replaced  bool   `json:"replaced"`
}
