CREATE TYPE deal_type AS ENUM ('sale', 'rent');

CREATE TABLE IF NOT EXISTS general_properties (
    master_object_id        UUID NOT NULL REFERENCES master_objects(id) ON DELETE CASCADE,
    is_source_duplicate     BOOLEAN NOT NULL DEFAULT FALSE,

    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source                  VARCHAR(50) NOT NULL, -- 'kufar', 'realt'
    source_ad_id            BIGINT NOT NULL,      -- ID из источника
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    category                VARCHAR(255) NOT NULL,
    ad_link                 TEXT NOT NULL, 
    sale_type               VARCHAR(20) NOT NULL, 
    currency                VARCHAR(10) NOT NULL,
    images                  TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    list_time               TIMESTAMPTZ NOT NULL,
    description             TEXT NOT NULL,
    title                   TEXT NOT NULL,
    deal_type               deal_type NOT NULL,
    coordinates             GEOGRAPHY(Point, 4326) NOT NULL,
    city_or_district        VARCHAR(255) NOT NULL,
    region                  VARCHAR(255) NOT NULL,
    price_byn               NUMERIC(14, 2) NOT NULL,
    price_usd               NUMERIC(14, 2) NOT NULL,
    price_eur               NUMERIC(14, 2),
    address                 TEXT NOT NULL,

    is_agency               BOOLEAN NOT NULL,
    seller_name             TEXT NOT NULL,
    seller_details          JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    -- Ограничение уникальности, чтобы не было дублей с одного источника --
    UNIQUE ("source", "source_ad_id")
);

CREATE INDEX idx_general_properties_master_id ON general_properties(master_object_id);
CREATE INDEX idx_general_properties_source ON general_properties(source);