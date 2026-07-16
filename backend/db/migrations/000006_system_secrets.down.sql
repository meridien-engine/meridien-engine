BEGIN;
DROP POLICY IF EXISTS system_secrets_tenant_isolation ON system_secrets;
DROP TABLE IF EXISTS system_secrets;
COMMIT;
