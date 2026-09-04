package documents

import "time"

type Status string

const (
	StatusPending  Status = "pending"
	StatusVerified Status = "verified"
	StatusRejected Status = "rejected"
	StatusExpired  Status = "expired"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusVerified, StatusRejected, StatusExpired:
		return true
	}
	return false
}

type Document struct {
	ID            int64
	Name          string
	Type          string
	ApplicantName string
	FileSizeKb    int64
	Status        Status
	UploadedAt    time.Time
}
