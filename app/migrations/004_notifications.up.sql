-- 004_notifications.up.sql

-- Notifications table
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    type TEXT NOT NULL, -- ticket_created, ticket_updated, asset_created, etc.
    related_id BIGINT, -- ID of the related ticket/asset
    related_type TEXT, -- 'ticket', 'asset', 'user'
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Index for faster queries
CREATE INDEX idx_notifications_user_id ON notifications (user_id);
CREATE INDEX idx_notifications_created_at ON notifications (created_at);
CREATE INDEX idx_notifications_is_read ON notifications (is_read);