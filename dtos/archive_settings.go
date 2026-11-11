package dtos

type ArchiveSettings struct {
	Storage       Storage `json:"storage"`
	RetentionDays int64   `json:"retentionDays"`
	BackupHour    int64   `json:"backupHour"`
}
