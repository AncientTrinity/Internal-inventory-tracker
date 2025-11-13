-- 003_permissions.down.sql

-- Remove role permissions first (due to foreign key constraints)
DELETE FROM role_permissions;

-- Remove all permissions
DELETE FROM permissions;