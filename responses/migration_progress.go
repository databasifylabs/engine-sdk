package responses

type MigrationProgress struct {
	Copy             string `json:"copy" pattern:"^(running|lost|succeeded|failed)$"`
	Log              string `json:"log"`
	SubscribedTables int64  `json:"subscribedTables"`
	ReadyTables      int64  `json:"readyTables"`
	LagSeconds       *int64 `json:"lagSeconds"`
	LagUnknown       bool   `json:"lagUnknown"`
}
