package responses

type RuntimeInfo struct {
	Protocol int64  `json:"protocol"`
	Engine   string `json:"engine"`
	Sdk      string `json:"sdk"`
}

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

type ServiceStatus struct {
	Units []ServiceUnit `json:"units"`
}

type ServiceUnit struct {
	Name        string `json:"name"`
	ActiveState string `json:"activeState"`
	SubState    string `json:"subState"`
	Result      string `json:"result"`
}
