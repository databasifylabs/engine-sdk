package responses

import "time"

type ArchiveStatus struct {
	Backups    int64      `json:"backups"`
	Earliest   *time.Time `json:"earliest"`
	Latest     *time.Time `json:"latest"`
	LastBackup *time.Time `json:"lastBackup"`
	Problem    *string    `json:"problem"`
}
