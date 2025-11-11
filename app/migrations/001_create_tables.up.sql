-- 001_create_tables.up.sql

-- roles
CREATE TABLE roles (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE -- admin, it, staff, agent, viewer
);

-- users
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  full_name TEXT,
  email TEXT UNIQUE,
  password_hash TEXT NOT NULL,
  role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- assets
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
  next_service_date DATE,
  notes TEXT
);

-- tickets
CREATE TABLE tickets (
  id BIGSERIAL PRIMARY KEY,
  ticket_num TEXT NOT NULL UNIQUE,      -- TCK-2025-0001
  title TEXT NOT NULL,
  description TEXT,
  type TEXT,                            -- activation, deactivation, it_help, transition, ...
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
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- useful indexes
CREATE INDEX idx_assets_internal_id ON assets (internal_id);
CREATE INDEX idx_tickets_ticket_num ON tickets (ticket_num);
CREATE INDEX idx_assets_in_use_by ON assets (in_use_by);
CREATE INDEX idx_tickets_assigned_to ON tickets (assigned_to);
