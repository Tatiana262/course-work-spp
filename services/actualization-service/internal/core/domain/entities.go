package domain

import "github.com/google/uuid"

const (
	ACTUALIZE_ARCHIVED = 1
	ACTUALIZE_ACTIVE   = 2
	FIND_NEW_OBJECTS = 3
	ACTUALIZE_OBJECT   = 4
)

const (
	KUFAR_SOURCE = "kufar"
	REALT_SOURCE = "realt"
)

// Структура для ответа API
type PropertyInfo struct {
	// ID        string    `json:"id"`

	Source    string    `json:"source"`
    AdID      int64     `json:"ad_id"`
    Link      string    `json:"ad_url"`
	TaskID    uuid.UUID `json:"task_id"`
    
    // UpdatedAt time.Time `json:"updatedAt"`
}

// Задача, которую мы отправляем в RabbitMQ
type ActualizationTask struct {
	Task       PropertyInfo // Ссылка на объект для пере-парсинга
	Source 	   string 
	Priority   uint8
	
}


type TaskInfo struct {
    Region         string `json:"region"`        
    Category	 string `json:"category"` 
	TaskID    uuid.UUID `json:"task_id"`
}

// Задача на поиск новых объектов
type FindNewLinksTask struct {
	Task TaskInfo
	RoutingKey string
	Priority   uint8
}

// TaskCompletionCommand - это команда, сообщающая task-service,
// что мы закончили отправку подзадач и сколько результатов нужно ожидать
type TaskCompletionCommand struct {
	TaskID               uuid.UUID `json:"task_id"`
	Results map[string]int `json:"results"`
	// ExpectedResultsCount int       `json:"expected_results_count"`
}


type DictionaryItem struct {
	SystemName  string 
	DisplayName string 
}