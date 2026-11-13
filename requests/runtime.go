package requests

type RuntimeRun struct {
	Script         string            `json:"script" min:"1"`
	Env            map[string]string `json:"env"`
	User           *string           `json:"user" pattern:"^[a-z_][a-z0-9_-]{0,31}$"`
	TimeoutSeconds int64             `json:"timeoutSeconds" min:"1" max:"3600"`
}

type TaskStart struct {
	Key    string            `json:"key" pattern:"^[a-z0-9][a-z0-9.-]{0,79}$"`
	Script string            `json:"script" min:"1"`
	Env    map[string]string `json:"env"`
	User   *string           `json:"user" pattern:"^[a-z_][a-z0-9_-]{0,31}$"`
}

type TaskKey struct {
	Key string `json:"key" pattern:"^[a-z0-9][a-z0-9.-]{0,79}$"`
}

type FilePut struct {
	Path          string  `json:"path" min:"1" max:"255"`
	ContentBase64 string  `json:"contentBase64"`
	Mode          int64   `json:"mode"`
	Owner         *string `json:"owner" pattern:"^[a-z_][a-z0-9_-]{0,31}$"`
}

type FileGet struct {
	Path string `json:"path" min:"1" max:"255"`
}
