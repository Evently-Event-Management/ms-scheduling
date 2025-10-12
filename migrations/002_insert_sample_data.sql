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
INSERT INTO subscriptions (subscriber_id, category, target_uuid) VALUES 
(1, 'organization', 'f47ac10b-58cc-4372-a567-0e02b2c3d479'),
(1, 'event', 'e47ac10b-58cc-4372-a567-0e02b2c3d480'),
(2, 'event', 'e47ac10b-58cc-4372-a567-0e02b2c3d480'),
(2, 'session', 'd47ac10b-58cc-4372-a567-0e02b2c3d481'),
(3, 'organization', 'f47ac10b-58cc-4372-a567-0e02b2c3d479')
ON CONFLICT (subscriber_id, category, target_uuid) DO NOTHING;