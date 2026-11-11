package requests

type LogsRead struct {
	Unit  *string `json:"unit" pattern:"^[a-z0-9-]{1,64}$"`
	Lines int64   `json:"lines" min:"1" max:"10000"`
}
