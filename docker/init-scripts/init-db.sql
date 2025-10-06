-- PostgreSQL initialization script for pg-squash
-- This script sets up the initial database configuration

-- Create database if it doesn't exist (for local development)
-- Note: This won't work in Docker entrypoint, but useful for reference

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create application user (if not exists)
DO $$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'pgsquash_app') THEN
      CREATE ROLE pgsquash_app LOGIN PASSWORD 'pgsquash_app_password';
   END IF;
END
$$;

-- Grant permissions
GRANT CONNECT ON DATABASE pgsquash TO pgsquash_app;
GRANT USAGE ON SCHEMA public TO pgsquash_app;
GRANT CREATE ON SCHEMA public TO pgsquash_app;

-- Set default permissions for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pgsquash_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO pgsquash_app;

-- Performance optimizations
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
ALTER SYSTEM SET pg_stat_statements.track = 'all';
ALTER SYSTEM SET pg_stat_statements.max = 1000;