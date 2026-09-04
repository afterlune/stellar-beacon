-- The repository historically shipped one complete schema dump instead of
-- versioned migrations. This marker records that the existing schema is the
-- starting point; it intentionally makes no schema change.
SELECT 1;
