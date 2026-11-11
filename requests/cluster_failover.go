package requests

type ClusterFailover struct {
	Target string `json:"target" pattern:"^[0-9A-Fa-f:.]{2,45}$"`
}
