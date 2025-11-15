-- 003_insert_permissions.up.sql

-- Insert base permissions for RBAC system
INSERT INTO permissions (name, resource, action, description) VALUES
    -- User permissions
    ('users:create', 'users', 'create', 'Create new users'),
    ('users:read', 'users', 'read', 'View users'),
    ('users:update', 'users', 'update', 'Update users'),
    ('users:delete', 'users', 'delete', 'Delete users'),
    
    -- Role permissions
    ('roles:create', 'roles', 'create', 'Create roles'),
    ('roles:read', 'roles', 'read', 'View roles'),
    ('roles:update', 'roles', 'update', 'Update roles'),
    ('roles:delete', 'roles', 'delete', 'Delete roles'),
    
    -- Asset permissions
    ('assets:create', 'assets', 'create', 'Create assets'),
    ('assets:read', 'assets', 'read', 'View assets'),
    ('assets:update', 'assets', 'update', 'Update assets'),
    ('assets:delete', 'assets', 'delete', 'Delete assets'),
    ('assets:manage', 'assets', 'manage', 'Full asset management'),
    
    -- Ticket permissions
    ('tickets:create', 'tickets', 'create', 'Create tickets'),
    ('tickets:read', 'tickets', 'read', 'View tickets'),
    ('tickets:update', 'tickets', 'update', 'Update tickets'),
    ('tickets:delete', 'tickets', 'delete', 'Delete tickets'),
    ('tickets:assign', 'tickets', 'assign', 'Assign tickets to users'),
    
    -- System permissions
    ('system:admin', 'system', 'admin', 'Full system administration'),
    ('audit:read', 'audit', 'read', 'View audit logs');

-- Assign permissions to roles

-- Admin: all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin';

-- IT Staff: user read, full asset and ticket management
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'it' AND p.name IN (
    'users:read',
    'assets:create', 'assets:read', 'assets:update', 'assets:manage',
    'tickets:create', 'tickets:read', 'tickets:update', 'tickets:assign'
);

-- Staff/Team Leads: ticket and asset viewing
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'staff' AND p.name IN (
    'tickets:read', 'tickets:update',
    'assets:read'
);

-- Agents: view own tickets only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'agent' AND p.name = 'tickets:read';

-- Viewers: read-only access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'viewer' AND p.name IN ('assets:read', 'tickets:read');

-- Add notification permissions
INSERT INTO permissions (name, resource, action, description) VALUES
('notifications:read', 'notifications', 'read', 'View notifications'),
('notifications:update', 'notifications', 'update', 'Mark notifications as read');

-- Assign notification permissions to roles
-- Admin: all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'admin' AND p.name IN ('notifications:read', 'notifications:update');

-- IT Staff: all notification permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'it' AND p.name IN ('notifications:read', 'notifications:update');

-- Staff: read notifications only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'staff' AND p.name = 'notifications:read';

-- Agents: read notifications only  
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'agent' AND p.name = 'notifications:read';

-- Viewers: read notifications only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'viewer' AND p.name = 'notifications:read';