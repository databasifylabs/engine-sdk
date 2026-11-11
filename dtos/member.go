package dtos

type Member struct {
	Ip       string `json:"ip" pattern:"^[0-9A-Fa-f:.]{2,45}$"`
	Position int64  `json:"position"`
	Voting   bool   `json:"voting"`
	Cpus     int64  `json:"cpus" min:"1"`
	MemoryMb int64  `json:"memoryMb" min:"1"`
	DiskGb   int64  `json:"diskGb" min:"1"`
}
