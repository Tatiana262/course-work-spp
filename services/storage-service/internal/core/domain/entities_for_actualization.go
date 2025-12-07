package domain

import (
	"time"

	"github.com/google/uuid"
)

type PropertyBasicInfo struct {
	ID         uuid.UUID 
	AdID       int64
	Source     string   
	Link 	   string   
	UpdatedAt  time.Time 
}


type BatchSaveStats struct {
	Created   int // Количество новых записей, которые были вставлены (INSERT)
	Updated   int // Количество существующих записей, которые были обновлены (UPDATE)
	Archived  int // Количество записей, которые были переведены в статус "archived"
}