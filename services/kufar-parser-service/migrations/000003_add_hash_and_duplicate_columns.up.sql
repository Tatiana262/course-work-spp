ALTER TABLE general_properties ADD COLUMN object_hash VARCHAR(64);
ALTER TABLE general_properties ADD COLUMN duplicate_of UUID NULL REFERENCES general_properties(id);

-- для производительности
CREATE INDEX idx_general_properties_object_hash ON general_properties (object_hash);