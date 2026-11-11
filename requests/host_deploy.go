package requests

type HostDeploy struct {
	Image      string `json:"image" pattern:"^[a-z0-9./_-]+(:[A-Za-z0-9._-]+)?@sha256:[a-f0-9]{64}$"`
	DatabaseId string `json:"databaseId" pattern:"^[A-Za-z0-9._-]{1,128}$"`
	NodeId     string `json:"nodeId" pattern:"^[A-Za-z0-9._-]{1,128}$"`
}
