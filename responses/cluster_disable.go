package responses

type ClusterDisable struct {
	DroppedSlots int64 `json:"droppedSlots"`
}
