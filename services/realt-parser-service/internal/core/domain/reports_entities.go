package domain

type ParsingTasksStats struct {
	SearchesCompleted   int // Количество новых записей, которые были вставлены (INSERT)
	NewLinksFound   int // Количество существующих записей, которые были обновлены (UPDATE)
}