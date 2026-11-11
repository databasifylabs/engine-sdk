package dtos

type Statement struct {
	Text    string `json:"text"`
	Calls   int64  `json:"calls"`
	TotalMs int64  `json:"totalMs"`
	MeanMs  int64  `json:"meanMs"`
}
