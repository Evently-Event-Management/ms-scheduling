-- Setup organization subscriptions for testing event creation notifications
-- This script creates test subscribers for organization ID 123

-- Insert test subscribers for organization ID 123
INSERT INTO subscriptions (subscriber_id, category, target_id, created_at) 
SELECT 1, 'organization', 123, NOW()  -- isurumuni.22@cse.mrt.ac.lk
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 1 AND category = 'organization' AND target_id = 123
);

INSERT INTO subscriptions (subscriber_id, category, target_id, created_at) 
SELECT 2, 'organization', 123, NOW()  -- user2@example.com
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 2 AND category = 'organization' AND target_id = 123
);

INSERT INTO subscriptions (subscriber_id, category, target_id, created_at) 
SELECT 3, 'organization', 123, NOW()  -- user3@example.com  
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 3 AND category = 'organization' AND target_id = 123
);

INSERT INTO subscriptions (subscriber_id, category, target_id, created_at) 
SELECT 4, 'organization', 123, NOW()  -- customer@example.com
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions 
    WHERE subscriber_id = 4 AND category = 'organization' AND target_id = 123
);

-- Verify the subscriptions
SELECT 
    s.subscriber_mail, 
    sub.category, 
    sub.target_id,
    sub.created_at
FROM subscribers s
JOIN subscriptions sub ON s.subscriber_id = sub.subscriber_id  
WHERE sub.category = 'organization' AND sub.target_id = 123
ORDER BY s.subscriber_mail;