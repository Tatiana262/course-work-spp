CREATE TABLE IF NOT EXISTS houses (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,

    total_area              NUMERIC(10, 2),  -- Общая площадь дома, м²
    plot_area               NUMERIC(10, 2),  -- Площадь участка, в сотках
    living_space_area       NUMERIC(10, 2),  -- Жилая площадь, м²
    kitchen_area            NUMERIC(10, 2),  -- Размер кухни, м²

    -- Параметры строения
    year_built              SMALLINT,        -- Год постройки
    building_floors         SMALLINT,        -- Этажность дома (используем SMALLINT как в apartments)
    rooms_amount            SMALLINT,        -- Количество комнат (используем SMALLINT как в apartments)
    wall_material           VARCHAR(100),    -- Материал стен
    roof_material           VARCHAR(100),    -- Материал крыши
    house_type              VARCHAR(100),    -- Тип дома (Дом, Коттедж, Дача, Таунхаус...)

    -- Коммуникации (используем VARCHAR для гибкости, т.к. значения могут быть разными)
    electricity             VARCHAR(100),
    water                   VARCHAR(100),
    heating                 VARCHAR(100),
    sewage                  VARCHAR(100),
    gaz                     VARCHAR(100),

    completion_percent      SMALLINT, 
    is_new_condition        BOOLEAN,

    -- "Карман" для редко встречающихся или неструктурированных параметров.
    -- JSONB является бинарным форматом, он быстрее и поддерживает индексацию.
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);