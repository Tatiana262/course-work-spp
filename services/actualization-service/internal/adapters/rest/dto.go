package rest

// ActualizeRequestDTO - структура для тела POST-запроса на актуализацию.
type ActualizeRequestDTO struct {
	Category string `json:"category"` // Например, "apartments", "houses"
	Limit    int    `json:"limit"`    // Количество объектов для актуализации
}

type ActualizeObjectDTO struct {
	Id string `json:"master_object_id"`
}


type FindNewRequestDTO struct {
    // Можно добавить фильтры, чтобы не запускать ВСЕ поиски сразу
    Categories []string `json:"categories"`
    Regions    []string `json:"regions"`
}