CREATE TABLE IF NOT EXISTS apartments (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,
    
    -- Основные характеристики
    rooms_amount            SMALLINT,
    floor_number            SMALLINT,
    building_floors         SMALLINT,
    total_area              NUMERIC(10, 2),
    living_space_area       NUMERIC(10, 2),
    kitchen_area            NUMERIC(10, 2),
    year_built              SMALLINT,
    wall_material           VARCHAR(100),
    repair_state            VARCHAR(100),
    bathroom_type           VARCHAR(100),
    balcony_type            VARCHAR(100),
    price_per_square_meter  NUMERIC(14, 2),

    -- Все остальное, что встречается редко
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);