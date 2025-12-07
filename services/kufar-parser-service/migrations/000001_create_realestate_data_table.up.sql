CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

CREATE TABLE IF NOT EXISTS general_properties (
    
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source                  VARCHAR(50) NOT NULL, -- 'kufar', 'onliner', etc.
    source_ad_id            BIGINT NOT NULL,      -- ID из источника
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    category                VARCHAR(255) NOT NULL,
    ad_link                 TEXT NOT NULL,
    company_ad              BOOLEAN NOT NULL,
    currency                VARCHAR(10) NOT NULL,
    images                  TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    list_time               TIMESTAMPTZ NOT NULL,
    body                    TEXT NOT NULL,
    subject                 TEXT NOT NULL,
    deal_type               VARCHAR(50) NOT NULL, -- 'sell', 'let'
    remuneration_type       VARCHAR(20) NOT NULL,           --Цена (тип оплаты)
    coordinates             GEOGRAPHY(Point, 4326) NOT NULL,
    city_or_district        VARCHAR(255) NOT NULL,
    region                  VARCHAR(255) NOT NULL,
    price_byn               NUMERIC(14, 2) NOT NULL,
    price_usd               NUMERIC(14, 2) NOT NULL,
    price_eur               NUMERIC(14, 2),
    address                 TEXT NOT NULL,
    seller_name             TEXT NOT NULL,
    contact_person          TEXT,
    unp_number              VARCHAR(9),
    company_address         TEXT,
    company_license         TEXT,         -- Лицензия
    import_link             TEXT,
    
    -- Ограничение уникальности, чтобы не было дублей с одного источника --
    UNIQUE ("source", "source_ad_id")
);

CREATE TABLE IF NOT EXISTS apartments (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,
    
    -- Основные характеристики (частота > 50%)
    rooms_amount            SMALLINT,
    condition               VARCHAR(100),
    building_floors         SMALLINT,
    total_area              NUMERIC(10, 2),
    year_built              SMALLINT,

    floor_number            SMALLINT,
    price_per_square_meter  NUMERIC(14, 2), --Цена за м^2 (square_meter)--
    living_space_area       NUMERIC(10, 2),
    kitchen_size            NUMERIC(10, 2),
    wall_material           VARCHAR(100),
    balcony                 VARCHAR(100),
    bathroom                VARCHAR(100),
    flat_repair             VARCHAR(100),
    contract_number_and_date TEXT,
    
    -- Все остальное, что встречается редко
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS houses (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,
    
    -- Основные характеристики
    total_area              NUMERIC(10, 2),
    plot_area               NUMERIC(10, 2),
    wall_material           VARCHAR(100),
    condition               VARCHAR(100),

    -- Опциональные характеристики
    year_built              SMALLINT,
    living_space_area       NUMERIC(10, 2),
    building_floors         SMALLINT,
    rooms_amount            SMALLINT,
    kitchen_size            NUMERIC(10, 2),
    electricity             BOOLEAN,
    in_gardening_community  BOOLEAN,
    water                   VARCHAR(100),
    heating                 VARCHAR(100),
    sewage                  VARCHAR(100),
    gaz                     VARCHAR(100),
    roof_material           VARCHAR(100),
    contract_number_and_date TEXT,
    house_type               TEXT,           --Дом, часть дома, дача (house_type_for_rent, house_type_for_sell)--
    
    -- Все остальное
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS commercial (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,
    
    -- Основные характеристики
    property_type           VARCHAR(255),   
    condition               VARCHAR(100),   
    
    -- Опциональные характеристики
    floor_number            SMALLINT,       
    building_floors         SMALLINT,       
    total_area              NUMERIC(10, 2), 
    commercial_improvements TEXT[],         
    commercial_repair       VARCHAR(100),   
    partly_sell_or_rent     BOOLEAN,        
    price_per_square_meter  NUMERIC(14, 2), 
    contract_number_and_date TEXT,          
    rooms_amount            SMALLINT,       
    commercial_building_location TEXT,      
    commercial_rent_type    VARCHAR(100),   
    
    -- Все остальное
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS garages_and_parkings (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,
    
    -- Основные характеристики
    property_type           VARCHAR(100), -- Гараж, машиноместо
    parking_places_amount   SMALLINT,
    total_area              NUMERIC(10, 2),

    -- Опциональные характеристики
    improvements            TEXT[],
    heating                 VARCHAR(100),
    parking_type            VARCHAR(100),

    -- Все остальное
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS plots (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,
    
    -- Основные характеристики
    plot_area               NUMERIC(10, 2),
    
    -- Опциональные характеристики
    in_gardening_community  BOOLEAN,
    property_rights         VARCHAR(100),
    electricity             VARCHAR(100),
    water                   VARCHAR(100),
    gaz                     VARCHAR(100),
    sewage                  VARCHAR(100),
    is_outbuildings         BOOLEAN,
    outbuildings_type       TEXT[],
    contract_number_and_date TEXT,          --Номер и дата договора (re_contract)--
    
    -- Все остальное
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS rooms (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,

    -- Основные поля (ваши названия сохранены)
    condition               VARCHAR(100),

    -- Специфичные поля для комнат
    bathroom                VARCHAR(100),
    suggested_rooms_amount  SMALLINT,
    rooms_amount            SMALLINT,
    floor_number            SMALLINT,
    building_floors         SMALLINT,
    total_area              NUMERIC(10, 2),
    is_balcony              BOOLEAN, -- Может быть "Есть", "Лоджия", "Нет"
    rental_type             VARCHAR(100),
    living_space_area       NUMERIC(10, 2),
    flat_repair             VARCHAR(100),
    is_furniture            BOOLEAN,
    kitchen_size            NUMERIC(10, 2),
    kitchen_items           TEXT[],
    bath_items              TEXT[],
    flat_rent_for_whom      TEXT[],
    flat_windows_side       TEXT[],
    year_built              SMALLINT,
    wall_material           VARCHAR(100),
    flat_improvement        TEXT[],
    room_type               VARCHAR(100),
    contract_number_and_date TEXT,
    flat_building_improvements TEXT[],

    -- Все остальные параметры
    parameters              JSONB NOT NULL DEFAULT '{}'::jsonb
);


CREATE TABLE IF NOT EXISTS new_buildings (
    property_id             UUID PRIMARY KEY REFERENCES general_properties(id) ON DELETE CASCADE,

    -- Специфичные поля для новостроек
    deadline                 TEXT,         -- Срок сдачи, может быть текстом "4 кв. 2025"
    room_options             SMALLINT[],   -- Массив чисел
    builder                  VARCHAR(255),
    share_participation      BOOLEAN,
    floor_options            SMALLINT[],   -- Массив чисел
    wall_material            VARCHAR(100),
    flat_ceiling_height      VARCHAR(100), -- Может быть текстом "от 2.7 м"
    layout_options           TEXT[],
    with_finishing           BOOLEAN,
    
    -- Все остальные параметры
    parameters               JSONB NOT NULL DEFAULT '{}'::jsonb
);
