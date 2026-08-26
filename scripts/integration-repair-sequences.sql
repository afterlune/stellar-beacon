-- Reconcile the sequence actually attached to every generated ID column.
-- This is safe to run repeatedly and does not delete or rewrite table data.
DO $$
DECLARE
    identity_column record;
    sequence_name text;
    max_id bigint;
    last_value bigint;
BEGIN
    FOR identity_column IN
        SELECT table_schema, table_name, column_name
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND (is_identity = 'YES' OR column_default LIKE 'nextval(%')
    LOOP
        sequence_name := pg_get_serial_sequence(
            format('%I.%I', identity_column.table_schema, identity_column.table_name),
            identity_column.column_name
        );
        IF sequence_name IS NULL THEN
            CONTINUE;
        END IF;

        EXECUTE format(
            'SELECT COALESCE(MAX(%I), 0) FROM %I.%I',
            identity_column.column_name,
            identity_column.table_schema,
            identity_column.table_name
        ) INTO max_id;
        EXECUTE format('SELECT last_value FROM %s', sequence_name) INTO last_value;

        IF max_id > last_value THEN
            PERFORM setval(sequence_name::regclass, max_id, true);
        END IF;
    END LOOP;
END $$;
