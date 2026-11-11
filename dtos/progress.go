package dtos

type Progress struct {
	Done  int64  `json:"done" min:"0"`
	Total *int64 `json:"total" min:"0"`
	Unit  string `json:"unit"`
}
