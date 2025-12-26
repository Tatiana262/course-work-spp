package rest


type ActualizeRequestDTO struct {
    Category           *string `json:"category"` // Указатель, чтобы отличить "не передано" от ""
    LimitPerCategory   int     `json:"limit_per_category"`
}

type ActualizeObjectDTO struct {
	Id string `json:"master_object_id"`
}


type FindNewRequestDTO struct {
    Categories []string `json:"categories"`
    Regions    []string `json:"regions"`
}

