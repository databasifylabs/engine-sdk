package requests

type MemberForget struct {
	Ip string `json:"ip" pattern:"^[0-9A-Fa-f:.]{2,45}$"`
}
