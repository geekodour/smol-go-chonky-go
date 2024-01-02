-- Roles
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'appadmin') THEN
        CREATE ROLE appadmin WITH LOGIN PASSWORD 'spider';
    END IF;
END $$;

-- Database
-- There's no non-hacky way to do this as of the moment
-- See https://github.com/supabase/postgres/blob/develop/docker/all-in-one/postgres-entrypoint.sh#L209-L222
-- See https://zaiste.net/databases/postgresql/howtos/create-database-if-not-exists/
SELECT 'CREATE DATABASE catdb OWNER appadmin' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'catdb')\gexec

GRANT pg_read_server_files TO appadmin;

\c catdb appadmin;

-- Schema
-- NOTE: We're not creating any schema, for now we'll be putting everything into
-- the "public" schema of catdb
