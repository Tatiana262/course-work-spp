CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

CREATE TABLE IF NOT EXISTS master_objects (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    canonical_hash          VARCHAR(64) NOT NULL UNIQUE, 
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_master_objects_canonical_hash ON master_objects(canonical_hash);