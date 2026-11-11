package requests

type TlsInstall struct {
	Certificate string `json:"certificate" min:"1"`
	Key         string `json:"key" min:"1"`
	Ca          string `json:"ca" min:"1"`
}
