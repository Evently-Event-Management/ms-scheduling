-- Migration: Add user_id column to subscribers table
-- Version: 003
-- Description: Add user_id column to support UUID-based user lookup from Keycloak

-- Add user_id column to subscribers table (can be nullable for existing records)
ALTER TABLE subscribers 
ADD COLUMN user_id VARCHAR(255);

-- Create index for user_id lookups
CREATE INDEX idx_subscribers_user_id ON subscribers(user_id);

-- Update existing constraint to allow lookup by either email or user_id
-- Note: We keep subscriber_mail as unique but user_id can also be unique when not null
CREATE UNIQUE INDEX idx_subscribers_user_id_unique ON subscribers(user_id) WHERE user_id IS NOT NULL;