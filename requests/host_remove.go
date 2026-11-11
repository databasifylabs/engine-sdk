package requests

type HostRemove struct {
	DatabaseId string `json:"databaseId" pattern:"^[A-Za-z0-9._-]{1,128}$"`
	Erase      bool   `json:"erase"`
}
