-- 002_seed_roles_users.up.sql

INSERT INTO roles (name, description) VALUES 
('admin', 'Full system administrator with all permissions'),
('it', 'IT staff with asset and ticket management'),
('staff', 'Team leads and QA staff'),
('agent', 'Call center agents'),
('viewer', 'Read-only access for viewing');

-- Example admin user (replace with the actual hash you generated)
INSERT INTO users (username, full_name, email, password_hash, role_id)
VALUES (
    'admin', 
    'System Administrator', 
    'admin@example.com', 
    '$2a$10$f9aP9r9teQ8hHyMiP21X1.aOmIELrWLjD.Nu8.xBT7GE3FUWsjUn.',  -- Replace this with actual hash
    (SELECT id FROM roles WHERE name='admin')
);