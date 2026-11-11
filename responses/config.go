package responses

import "github.com/databasifylabs/engine-sdk/dtos"

type ConfigFields struct {
	Fields []dtos.ConfigField `json:"fields"`
}

type ConfigShow struct {
	Values []dtos.ConfigValue `json:"values"`
}

type ConfigTune struct {
	Tuned   map[string]string `json:"tuned"`
	Restart []string          `json:"restart"`
}
