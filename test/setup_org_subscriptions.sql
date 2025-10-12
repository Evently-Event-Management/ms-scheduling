-- Setup organization subscriptions for testing event creation notifications
-- This script creates test subscribers for organization ID f47ac10b-58cc-4372-a567-0e02b2c3d479

-- Insert test subscribers for organization ID
INSERT INTO subscriptions (subscriber_id, category, target_uuid, created_at) 
SELECT 1, 'organization', 'f47ac10b-58cc-4372-a567-0e02b2c3d479', NOW()  -- isurumuni.22@cse.mrt.ac.lk
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 1 AND category = 'organization' AND target_uuid = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
);

INSERT INTO subscriptions (subscriber_id, category, target_uuid, created_at) 
SELECT 2, 'organization', 'f47ac10b-58cc-4372-a567-0e02b2c3d479', NOW()  -- user2@example.com
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 2 AND category = 'organization' AND target_uuid = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
);

INSERT INTO subscriptions (subscriber_id, category, target_uuid, created_at) 
SELECT 3, 'organization', 'f47ac10b-58cc-4372-a567-0e02b2c3d479', NOW()  -- user3@example.com  
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 3 AND category = 'organization' AND target_uuid = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
);

INSERT INTO subscriptions (subscriber_id, category, target_uuid, created_at) 
SELECT 4, 'organization', 'f47ac10b-58cc-4372-a567-0e02b2c3d479', NOW()  -- customer@example.com
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 4 AND category = 'organization' AND target_uuid = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
);

-- Verify the subscriptions
SELECT 
    s.subscriber_mail, 
    sub.category, 
    sub.target_uuid,
    sub.created_at
FROM subscribers s
JOIN subscriptions sub ON s.subscriber_id = sub.subscriber_id  
WHERE sub.category = 'organization' AND sub.target_uuid = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
ORDER BY s.subscriber_mail;