-- 002_seed_roles_users.up.sql

INSERT INTO roles (name) VALUES ('admin'), ('it'), ('staff'), ('agent'), ('viewer');

-- Example admin user (replace '<bcrypt_hash_here>' with an actual bcrypt hash)
INSERT INTO users (username, full_name, email, password_hash, role_id)
VALUES ('admin', 'Administrator', 'admin@example.com', '<bcrypt_hash_here>', (SELECT id FROM roles WHERE name='admin'));
