package dtos

type Storage struct {
	Endpoint  string `json:"endpoint" pattern:"^https://[A-Za-z0-9.-]+(:[0-9]{1,5})?$"`
	Bucket    string `json:"bucket" min:"1" max:"63"`
	Region    string `json:"region" min:"1" max:"64"`
	AccessKey string `json:"accessKey" min:"1"`
	SecretKey string `json:"secretKey" min:"1"`
	PathStyle bool   `json:"pathStyle"`
	VerifyTls bool   `json:"verifyTls"`
	Path      string `json:"path" pattern:"^[A-Za-z0-9._/-]*$"`
}
