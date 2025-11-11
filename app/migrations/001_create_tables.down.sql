-- 001_create_tables.down.sql
-- Drop indexes first
DROP INDEX IF EXISTS idx_assets_internal_id;
DROP INDEX IF EXISTS idx_tickets_ticket_num;
DROP INDEX IF EXISTS idx_assets_in_use_by;
DROP INDEX IF EXISTS idx_tickets_assigned_to;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS ticket_comments;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS asset_service;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;
