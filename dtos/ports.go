package dtos

type Ports struct {
	Client int   `json:"client"`
	Pool   *int  `json:"pool"`
	Peers  []int `json:"peers"`
}
