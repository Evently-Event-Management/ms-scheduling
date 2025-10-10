-- Migration: Insert Sample Data
-- Version: 002
-- Description: Insert sample subscribers and subscriptions for testing

-- Insert sample subscribers
INSERT INTO subscribers (subscriber_mail) VALUES 
('user1@example.com'),
('user2@example.com'),
('user3@example.com'),
('admin@ticketly.com'),
('test@ticketly.com')
ON CONFLICT (subscriber_mail) DO NOTHING;

-- Insert sample subscriptions
INSERT INTO subscriptions (subscriber_id, category, target_id) VALUES 
(1, 'organization', 123),
(1, 'event', 456),
(2, 'event', 456),
(2, 'session', 789),
(3, 'organization', 123)
ON CONFLICT (subscriber_id, category, target_id) DO NOTHING;