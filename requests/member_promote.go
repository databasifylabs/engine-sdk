package requests

type MemberPromote struct {
	Ip       string `json:"ip" pattern:"^[0-9A-Fa-f:.]{2,45}$"`
	ReadOnly bool   `json:"readOnly"`
}
