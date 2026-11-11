package requests

type OperationCancel struct {
	Key string `json:"key" pattern:"^[A-Za-z0-9._-]{1,128}$"`
}
