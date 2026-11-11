package requests

type UserSetPassword struct {
	Name       string  `json:"name" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"32"`
	Password   string  `json:"password" min:"1"`
	Encryption *string `json:"encryption"`
}
