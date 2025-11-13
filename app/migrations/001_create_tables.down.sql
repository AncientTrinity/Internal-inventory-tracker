-- 001_create_tables.down.sql
-- Drop tables in reverse dependency order

-- Drop the citext extension
DROP EXTENSION IF EXISTS citext;

-- Drop indexes first
DROP INDEX IF EXISTS idx_audit_log_created_at;
DROP INDEX IF EXISTS idx_audit_log_user_id;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_assets_status;
DROP INDEX IF EXISTS idx_tickets_status;
DROP INDEX IF EXISTS idx_tickets_assigned_to;
DROP INDEX IF EXISTS idx_assets_in_use_by;
DROP INDEX IF EXISTS idx_tickets_ticket_num;
DROP INDEX IF EXISTS idx_assets_internal_id;
DROP INDEX IF EXISTS idx_role_permissions_role_id;

-- Drop tables
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS ticket_comments;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS asset_service;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;