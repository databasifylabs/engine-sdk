package requests

type OperationStatus struct {
	Key string `json:"key" pattern:"^[A-Za-z0-9._-]{1,128}$"`
}
