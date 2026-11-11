package requests

type PoolUpdate struct {
	Name     string `json:"name" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
	Database string `json:"database" pattern:"^[A-Za-z][A-Za-z0-9_]*$" max:"64"`
	Size     int64  `json:"size" min:"1" max:"1000"`
	Mode     string `json:"mode" pattern:"^(session|transaction|statement)$"`
}
