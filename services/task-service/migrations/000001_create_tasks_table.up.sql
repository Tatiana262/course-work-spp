-- Убедитесь, что у вас включено расширение для UUID, если его еще нет
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tasks (
    
    id UUID PRIMARY KEY,
    
    name VARCHAR(255) NOT NULL, -- Например: "Актуализация активных квартир в Минске (500 шт.)"
    type VARCHAR(100) NOT NULL, -- Например: "ACTUALIZE_ACTIVE", "FIND_NEW", "ACTUALIZE_BY_ID"
    status VARCHAR(50) NOT NULL,     -- Значения: "pending", "running", "completed", "failed"
    result_summary JSONB,   -- результат выполнения в формате JSONB. Например: {"processed_count": 100, "archived_count": 5}

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Когда задача была создана
    started_at TIMESTAMPTZ,                         -- Когда задача перешла в статус "running"
    finished_at TIMESTAMPTZ,                        -- Когда задача перешла в статус "completed" или "failed"

    created_by_user_id UUID NOT NULL
);

-- Индексы для ускорения запросов:

-- Индекс для быстрого поиска всех задач конкретного пользователя,
-- отсортированных по времени создания (основной запрос для админ-панели).
-- CREATE INDEX idx_tasks_user_id_created_at ON tasks (created_by_user_id, created_at DESC);

-- Индекс по статусу, если вы захотите искать все "зависшие" задачи в статусе "running".
-- CREATE INDEX idx_tasks_status ON tasks (status);