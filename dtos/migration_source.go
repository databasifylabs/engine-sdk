package dtos

type MigrationSource struct {
	Host     string `json:"host" min:"1"`
	Port     int64  `json:"port"`
	Username string `json:"username" min:"1"`
	Password string `json:"password"`
	Tls      bool   `json:"tls"`
}
