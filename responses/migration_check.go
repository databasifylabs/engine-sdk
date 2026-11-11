package responses

import "encoding/json"

type MigrationCheck struct {
	Reachable    bool     `json:"reachable"`
	Error        *string  `json:"error"`
	Version      *string  `json:"version"`
	WalLevel     *string  `json:"walLevel"`
	CanReplicate bool     `json:"canReplicate"`
	SizeMb       *int64   `json:"sizeMb"`
	Databases    []string `json:"databases"`
	Unreadable   []string `json:"unreadable"`
	Unowned      []string `json:"unowned"`
	Occupied     []string `json:"occupied"`
	Keys         *int64   `json:"keys"`
	Warnings     []string `json:"warnings"`
	Problems     []string `json:"problems"`
}

func (c MigrationCheck) MarshalJSON() ([]byte, error) {
	type plain MigrationCheck
	for _, list := range []*[]string{&c.Databases, &c.Unreadable, &c.Unowned, &c.Occupied, &c.Warnings, &c.Problems} {
		if *list == nil {
			*list = []string{}
		}
	}

	return json.Marshal(plain(c))
}
