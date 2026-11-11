package requests

type DatabaseCreate struct {
	Name string `json:"name" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
}
