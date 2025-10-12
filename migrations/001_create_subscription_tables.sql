-- Migration: Create Subscription Tables
-- Version: 001
-- Description: Initial subscription system tables with categories and relationships

-- Create ENUM for subscription categories
CREATE TYPE subscription_category AS ENUM ('organization', 'event', 'session');

-- Create subscribers table (required for foreign key reference)
CREATE TABLE subscribers (
    subscriber_id SERIAL PRIMARY KEY,
    subscriber_mail VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create subscriptions table
CREATE TABLE subscriptions (
    subscription_id SERIAL PRIMARY KEY,
    subscriber_id INT REFERENCES subscribers(subscriber_id) ON DELETE CASCADE,
    category subscription_category NOT NULL,
    target_uuid VARCHAR(255) NOT NULL,  -- UUID for org_id / event_id / session_id depending on category
    subscribed_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(subscriber_id, category, target_uuid)
);

-- Create indexes for better performance
CREATE INDEX idx_subscriptions_subscriber_id ON subscriptions(subscriber_id);
CREATE INDEX idx_subscriptions_category_target_uuid ON subscriptions(category, target_uuid);
CREATE INDEX idx_subscribers_mail ON subscribers(subscriber_mail);
