SET @db := DATABASE();

SET @idx_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = @db
    AND TABLE_NAME = 'images'
    AND INDEX_NAME = 'idx_format_created'
);

SET @sql := IF(
  @idx_exists = 0,
  'CREATE INDEX idx_format_created ON images(format, created_at)',
  'SELECT ''idx_format_created already exists'';'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
