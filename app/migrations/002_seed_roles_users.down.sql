-- 002_seed_roles_users.down.sql
-- Remove seeded users first
DELETE FROM users WHERE username = 'admin';

-- Remove seeded roles
DELETE FROM roles WHERE name IN ('admin','it','staff','agent','viewer');
