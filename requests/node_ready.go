package requests

type NodeReady struct {
	Key           string `json:"key" pattern:"^[A-Za-z0-9._-]{1,128}$"`
	ExpectPrimary bool   `json:"expectPrimary"`
}
