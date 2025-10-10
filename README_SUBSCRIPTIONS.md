# Order Processing & Subscription System

This system has been enhanced to handle order creation events from Kafka and automatically manage user subscriptions and email notifications.

## Features Added

### 1. **Order Processing**
- Listens to `ticketly.order.created` Kafka topic
- Automatically processes order events in real-time
- Extracts user and order information from Kafka messages

### 2. **Subscriber Management**
- Automatic subscriber creation from order events
- Integration with Keycloak for user email lookup
- Database storage of subscriber information

### 3. **Subscription Management**
- Automatic subscription to events and sessions when orders are placed
- Subscription tracking in PostgreSQL database
- Support for organization, event, and session subscriptions

### 4. **Email Notifications**
- Automatic order confirmation emails
- HTML formatted emails with order details
- SMTP support for various email providers

### 5. **Database Integration**
- PostgreSQL database with subscription tables
- Automatic table creation and migration
- Proper indexing for performance

## System Architecture

```
Kafka (ticketly.order.created) 
    ↓
Consumer (processOrderCreated)
    ↓
SubscriberService
    ↓ (if user not found)
KeycloakClient → Get user email
    ↓
Database → Create/Update subscriber
    ↓
Add subscriptions (event, session)
    ↓
EmailService → Send confirmation email
```

## Configuration

Copy `.env.example` to `.env` and configure:

### Required Environment Variables

```bash
# Database (Required)
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=ticketly
DATABASE_USER=postgres
DATABASE_PASSWORD=your-password

# Keycloak (Required for user lookup)
KEYCLOAK_URL=http://auth.ticketly.com:8080
KEYCLOAK_REALM=event-ticketing
KEYCLOAK_CLIENT_ID=scheduler-service-client
SCHEDULER_CLIENT_SECRET=your-client-secret

# Email (Required for notifications)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
FROM_EMAIL=noreply@ticketly.com

# Kafka (Required for order processing)
KAFKA_URL=localhost:9092
```

## Database Setup

The system automatically creates these tables:

1. **subscribers** - User email addresses and metadata
2. **subscriptions** - User subscriptions to events/sessions/organizations

### Manual Setup (Optional)

```sql
-- Run the database migration
\i migrations/db.sql
```

## Usage

### 1. **Start the Service**

```bash
# Install dependencies
go mod tidy

# Run the service
go run .
```

### 2. **Order Processing Flow**

When an order is created and published to Kafka:

1. **Order Event Received**: System receives `ticketly.order.created` event
2. **User Lookup**: Searches for user email in database or Keycloak
3. **Subscriber Creation**: Creates subscriber record if not exists
4. **Subscription Creation**: Adds subscriptions for the event and session
5. **Email Notification**: Sends order confirmation email to user

### 3. **Example Order Event**

```json
{
  "OrderID": "db5b513b-7e4f-4ade-b784-977cd4276edf",
  "UserID": "bf5377e7-c064-4f1d-8471-ce1c883b155f",
  "EventID": "e8b7f94b-2a55-49c3-82ec-f02c93965486",
  "SessionID": "7af78d86-f8eb-45bf-abac-381d6a8176d3",
  "Status": "pending",
  "Price": 150.00,
  "tickets": [
    {
      "seat_label": "6C",
      "tier_name": "General Admission",
      "price_at_purchase": 75.00
    }
  ]
}
```

## Services

### SubscriberService
- `GetOrCreateSubscriber(userID)` - Get or create subscriber
- `AddSubscription()` - Add event/session subscription
- `SendOrderConfirmationEmail()` - Send email notification

### KeycloakClient  
- `GetUserEmail(userID)` - Fetch user email from Keycloak
- `getAdminToken()` - Get Keycloak admin token

### EmailService
- `SendEmail()` - Send SMTP email
- `SendOrderConfirmationEmail()` - Send formatted order email

### DatabaseService
- `NewDatabaseService()` - Initialize PostgreSQL connection
- `InitializeTables()` - Create database tables

## Monitoring

### Logs
The system provides comprehensive logging:

```
2025-10-10 15:30:45 Processing order.created for OrderID=db5b513b... UserID=bf5377e7...
2025-10-10 15:30:46 Created new subscriber for user bf5377e7... with email user@example.com
2025-10-10 15:30:47 Email sent successfully to user@example.com
2025-10-10 15:30:47 Successfully processed order db5b513b... for user bf5377e7...
```

### Database Queries

```sql
-- Check subscribers
SELECT * FROM subscribers ORDER BY created_at DESC LIMIT 10;

-- Check subscriptions  
SELECT s.*, sub.subscriber_mail 
FROM subscriptions s 
JOIN subscribers sub ON s.subscriber_id = sub.subscriber_id 
ORDER BY s.subscribed_at DESC;

-- Get event subscribers
SELECT COUNT(*) FROM subscriptions WHERE category = 'event' AND target_id = 123;
```

## Error Handling

The system handles various error scenarios:

- **User not found in Keycloak**: Logs error and skips processing
- **Database connection issues**: Retries and logs errors
- **Email delivery failures**: Logs but doesn't block processing
- **Invalid Kafka messages**: Logs and continues processing

## Email Templates

Order confirmation emails include:
- Order ID and status
- Event and session details
- Individual ticket information
- Total price and payment status
- Professional HTML formatting

## Extension Points

### Custom Email Templates
Override `generateOrderEmailTemplate()` in SubscriberService

### Additional Subscriptions
Extend subscription categories in the database enum

### Custom User Lookup
Implement alternative user lookup methods in SubscriberService

### Notification Channels
Add SMS, push notifications by extending EmailService

## Production Considerations

1. **Database Connection Pooling**: Configure proper connection limits
2. **Email Rate Limiting**: Implement rate limiting for email sending
3. **Kafka Consumer Groups**: Ensure proper consumer group configuration
4. **Error Queues**: Implement dead letter queues for failed processing
5. **Monitoring**: Add metrics and alerts for system health
6. **Security**: Use proper secrets management for sensitive configuration

## Testing

The system can be tested by publishing order events to the Kafka topic:

```bash
# Example Kafka message
kafka-console-producer --topic ticketly.order.created --bootstrap-server localhost:9092
```

The system will automatically process the order and send confirmation emails.