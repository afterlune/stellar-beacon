$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$sqlPath = Join-Path $script:IntegrationRepoRoot 'scripts/integration-repair-sequences.sql'
if (-not (Test-Path -LiteralPath $sqlPath)) {
    throw "Missing sequence repair SQL: $sqlPath"
}

$sql = Get-Content -LiteralPath $sqlPath -Raw
Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'postgresql', 'psql',
    '-U', 'postgres',
    '-d', 'benetnasch',
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
            RAISE EXCEPTION 'identity sequence % is behind %.%', sequence_name, identity_column.table_schema, identity_column.table_name;
        END IF;
    END LOOP;
END $$;
'@

Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'postgresql', 'psql',
    '-U', 'postgres',
    '-d', 'benetnasch',
    '-v', 'ON_ERROR_STOP=1',
    '-c', $verificationSql
)

Write-Host 'Isolated database identity sequences are synchronized.' -ForegroundColor Green
