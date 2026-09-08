package audit

import (
	"time"
)

type EntryDTO struct {
	ID         int64  `json:"id"`
	UserID     *int64 `json:"userId"`
	UserName   string `json:"userName"`
	UserEmail  string `json:"userEmail"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	ResourceID *int64 `json:"resourceId"`
	CreatedAt  string `json:"createdAt"`
}

func toDTO(e Entry) EntryDTO {
	return EntryDTO{
		ID:         e.ID,
		UserID:     e.UserID,
		UserName:   e.UserName,
		UserEmail:  e.UserEmail,
		Action:     e.Action,
		Resource:   e.Resource,
		ResourceID: e.ResourceID,
		CreatedAt:  e.CreatedAt.UTC().Format(time.RFC3339),
	}
}
