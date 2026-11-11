package responses

type EngineHostProfile struct {
	Settings                    map[string]string `json:"settings"`
	Floors                      map[string]int64  `json:"floors"`
	DisableTransparentHugepages bool              `json:"disableTransparentHugepages"`
}
