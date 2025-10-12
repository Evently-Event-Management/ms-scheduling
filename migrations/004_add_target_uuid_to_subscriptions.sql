-- Migration: Add target_uuid column to subscriptions table
-- Version: 004
-- Description: Add target_uuid column to support UUID-based target identification instead of int IDs

-- Add target_uuid column to subscriptions table
ALTER TABLE subscriptions 
ADD COLUMN target_uuid VARCHAR(255);

-- Create index for target_uuid lookups
CREATE INDEX idx_subscriptions_target_uuid ON subscriptions(target_uuid);

-- Update unique constraint to include target_uuid
DROP INDEX IF EXISTS idx_subscriptions_subscriber_category_target;
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_subscriber_id_category_target_id_key;

-- Create new unique constraint that uses either target_id or target_uuid
ALTER TABLE subscriptions 
ADD CONSTRAINT subscriptions_subscriber_id_category_target_key
UNIQUE (subscriber_id, category, COALESCE(target_uuid, ''::VARCHAR), COALESCE(target_id, 0));