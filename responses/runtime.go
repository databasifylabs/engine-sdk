package responses

type RuntimeRun struct {
	Exit      int64  `json:"exit"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Truncated bool   `json:"truncated"`
}

type TaskLog struct {
	Lines string `json:"lines"`
}

type FileGet struct {
	ContentBase64 string `json:"contentBase64"`
	Exists        bool   `json:"exists"`
}
