package responses

type NodeMetrics struct {
	Connections int64  `json:"connections"`
	Queries     int64  `json:"queries"`
	DataMb      int64  `json:"dataMb"`
	Keys        *int64 `json:"keys"`
	Hits        *int64 `json:"hits"`
	Misses      *int64 `json:"misses"`
	Evicted     *int64 `json:"evicted"`
}
