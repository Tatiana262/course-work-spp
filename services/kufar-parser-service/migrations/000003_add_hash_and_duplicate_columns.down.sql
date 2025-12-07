-- Удаляем индекс (сначала!)
DROP INDEX IF EXISTS idx_general_properties_object_hash;

-- Удаляем добавленные колонки
ALTER TABLE general_properties DROP COLUMN IF EXISTS duplicate_of;
ALTER TABLE general_properties DROP COLUMN IF EXISTS object_hash;