package dtos

type DefaultAccount struct {
	Database string `json:"database" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
	User     string `json:"user" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
	Password string `json:"password" min:"1"`
}
