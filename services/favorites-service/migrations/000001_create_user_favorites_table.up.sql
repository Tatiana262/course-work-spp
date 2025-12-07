CREATE TABLE user_favorites (
    user_id UUID NOT NULL,
    master_object_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, master_object_id) -- Составной первичный ключ, чтобы нельзя было добавить один и тот же объект дважды
);

-- Индекс для быстрого поиска всех избранных объектов для одного пользователя
CREATE INDEX ON user_favorites (user_id);