package dtos

type ConfigField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"`
	Default     string   `json:"default"`
	Options     []string `json:"options"`
	Unit        *string  `json:"unit"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Pattern     *string  `json:"pattern"`
	Restart     bool     `json:"restart"`
	Tuned       bool     `json:"tuned"`
}

type ConfigValue struct {
	Key            string `json:"key"`
	Value          string `json:"value"`
	Source         string `json:"source"`
	PendingRestart bool   `json:"pendingRestart"`
}
