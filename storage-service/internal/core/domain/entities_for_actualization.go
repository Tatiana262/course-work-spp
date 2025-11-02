package domain

import (
	"time"

	"github.com/google/uuid"
)

type PropertyBasicInfo struct {
	ID         uuid.UUID 
	Source     string    
	SourceAdID int64    
	UpdatedAt  time.Time 
}