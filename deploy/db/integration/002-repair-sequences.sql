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
        SELECT table_schema, table_name, column_name, is_identity, ordinal_position
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND (is_identity = 'YES' OR column_default LIKE 'nextval(%')
    LOOP
        IF identity_column.is_identity = 'YES' THEN
            SELECT format('%I.%I', sequence_schema.nspname, sequence_rel.relname)
            INTO sequence_name
            FROM pg_class AS sequence_rel
            JOIN pg_namespace AS sequence_schema ON sequence_schema.oid = sequence_rel.relnamespace
            JOIN pg_depend AS dependency ON dependency.objid = sequence_rel.oid
            JOIN pg_class AS table_rel ON table_rel.oid = dependency.refobjid
            JOIN pg_namespace AS table_schema ON table_schema.oid = table_rel.relnamespace
            WHERE sequence_rel.relkind = 'S'
              AND dependency.deptype = 'i'
              AND table_schema.nspname = identity_column.table_schema
              AND table_rel.relname = identity_column.table_name
              AND dependency.refobjsubid = identity_column.ordinal_position
            LIMIT 1;
        ELSE
            sequence_name := pg_get_serial_sequence(
                format('%I.%I', identity_column.table_schema, identity_column.table_name),
                identity_column.column_name
            );
        END IF;
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
