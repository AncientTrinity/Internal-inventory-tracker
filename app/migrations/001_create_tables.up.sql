-- 001_create_tables.up.sql

-- roles table with enhanced columns for RBAC
CREATE TABLE roles (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE, -- admin, it, staff, agent, viewer
  description TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- users table
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  full_name TEXT,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- permissions table for fine-grained access control
CREATE TABLE permissions (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,        -- 'create_ticket', 'view_assets', 'manage_users'
  resource TEXT NOT NULL,           -- 'tickets', 'assets', 'users'
  action TEXT NOT NULL,             -- 'create', 'read', 'update', 'delete', 'manage'
  description TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- role_permissions junction table
CREATE TABLE role_permissions (
  id SERIAL PRIMARY KEY,
  role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(role_id, permission_id)
);

-- assets table
CREATE TABLE assets (
  id BIGSERIAL PRIMARY KEY,
  internal_id TEXT NOT NULL UNIQUE,     -- DPA-PC001, AM-M001, etc.
  asset_type TEXT NOT NULL,             -- PC, Monitor, Keyboard, Mouse, Headset, UPS
  manufacturer TEXT,
  model TEXT,
  model_number TEXT,
  serial_number TEXT,
  status TEXT NOT NULL DEFAULT 'IN_STORAGE', -- IN_USE, IN_STORAGE, RETIRED, REPAIR
  in_use_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
  date_purchased DATE,
  last_service_date DATE,
  next_service_date DATE,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- asset service logs (history)
CREATE TABLE asset_service (
  id BIGSERIAL PRIMARY KEY,
  asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  performed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
  performed_at TIMESTAMP NOT NULL DEFAULT now(),
  service_type TEXT NOT NULL, -- 'MAINTENANCE', 'REPAIR', 'UPGRADE'
  next_service_date DATE,
  notes TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- tickets table
CREATE TABLE tickets (
  id BIGSERIAL PRIMARY KEY,
  ticket_num TEXT NOT NULL UNIQUE,      -- TCK-2025-0001
  title TEXT NOT NULL,
  description TEXT,
  type TEXT NOT NULL,                   -- activation, deactivation, it_help, transition, ...
  priority TEXT DEFAULT 'normal',       -- low, normal, high, critical
  status TEXT DEFAULT 'open',           -- open, in_progress, resolved, closed
  created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
  assigned_to BIGINT REFERENCES users(id) ON DELETE SET NULL,
  asset_id BIGINT REFERENCES assets(id) ON DELETE SET NULL,
  is_internal BOOLEAN DEFAULT true,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  closed_at TIMESTAMP
);

-- ticket comments
CREATE TABLE ticket_comments (
  id BIGSERIAL PRIMARY KEY,
  ticket_id BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  comment TEXT NOT NULL,
  is_internal BOOLEAN DEFAULT false, -- whether the comment is visible to non-staff
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- audit_log table for tracking important actions
CREATE TABLE audit_log (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,                -- 'USER_LOGIN', 'ASSET_CREATED', 'TICKET_UPDATED'
  resource_type TEXT NOT NULL,         -- 'user', 'asset', 'ticket'
  resource_id BIGINT,                  -- ID of the affected resource
  old_values JSONB,                    -- previous state
  new_values JSONB,                    -- new state
  ip_address INET,
  user_agent TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- useful indexes
CREATE INDEX idx_assets_internal_id ON assets (internal_id);
CREATE INDEX idx_tickets_ticket_num ON tickets (ticket_num);
CREATE INDEX idx_assets_in_use_by ON assets (in_use_by);
CREATE INDEX idx_tickets_assigned_to ON tickets (assigned_to);
CREATE INDEX idx_tickets_status ON tickets (status);
CREATE INDEX idx_assets_status ON assets (status);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_audit_log_user_id ON audit_log (user_id);
CREATE INDEX idx_audit_log_created_at ON audit_log (created_at);
CREATE INDEX idx_role_permissions_role_id ON role_permissions (role_id);