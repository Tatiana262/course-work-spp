CREATE TABLE IF NOT EXISTS commercial (
    -- Внешний ключ, связывающий с основной таблицей.
    property_id                 UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,
    
    -- Характеристики
    is_new_condition            BOOLEAN,
    property_type               VARCHAR(100),   -- Тип объекта: "Офис", "Склад", "Магазин"
    floor_number                SMALLINT,       -- Этаж расположения
    building_floors             SMALLINT,       -- Всего этажей в здании
    total_area                  NUMERIC(10, 2), -- Площадь в м²
    commercial_improvements     TEXT[],         -- Массив строк с улучшениями
    commercial_repair           VARCHAR(100),   -- Состояние/ремонт
    price_per_square_meter      NUMERIC(14, 2), -- Цена за м²
    rooms_range                 SMALLINT[],     -- Массив для хранения диапазона комнат [min, max] или точного значения [n]
    commercial_building_location VARCHAR(255),  -- Тип/расположение здания
    commercial_rent_type        VARCHAR(50),    -- Тип аренды: "Прямая", "Субаренда"

    parameters                  JSONB NOT NULL DEFAULT '{}'::jsonb
);