package requests

type NodeReset struct {
	Peers int64    `json:"peers" min:"0"`
	Hosts []string `json:"hosts" pattern:"^[0-9A-Fa-f:.]{2,45}$"`
}
