$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'common.ps1')
Import-IntegrationEnv

$sqlPath = Join-Path $script:IntegrationRepoRoot 'deploy/db/integration/002-repair-sequences.sql'
if (-not (Test-Path -LiteralPath $sqlPath)) {
    throw "Missing sequence repair SQL: $sqlPath"
}

$sql = Get-Content -LiteralPath $sqlPath -Raw
Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'postgresql', 'psql',
    '-U', 'postgres',
    '-d', 'stellar_beacon',
    '-v', 'ON_ERROR_STOP=1',
    '-c', $sql
)

$verificationSql = @'
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
            RAISE EXCEPTION 'identity sequence % is behind %.%', sequence_name, identity_column.table_schema, identity_column.table_name;
        END IF;
    END LOOP;
END $$;
'@

Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'postgresql', 'psql',
    '-U', 'postgres',
    '-d', 'stellar_beacon',
    '-v', 'ON_ERROR_STOP=1',
    '-c', $verificationSql
)

Write-Host 'Isolated database identity sequences are synchronized.' -ForegroundColor Green
