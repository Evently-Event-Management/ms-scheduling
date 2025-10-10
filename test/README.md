# Order Email Test Cases

This directory contains test cases for the order confirmation email system integrated with Kafka order events.

## 📋 Test Files

### 1. `order_demo.go`

**Main Test Case** - Complete end-to-end test without external dependencies.

**What it tests:**
- ✅ Database connection and migration status
- ✅ Subscriber creation/retrieval 
- ✅ Subscription management
- ✅ Email service integration
- ✅ Order confirmation email generation
- ✅ Database state verification

**How to run:**

```bash
cd test
go run order_demo.go
```

**Expected Output:**
```
🧪 Order Email Test - Direct Database & Service Test
============================================================
✅ Database connected successfully
✅ Services initialized

📦 Test Order Created:
   📋 Order: TEST_ORDER_1728576000
   👤 Customer: customer@example.com
   🎫 Event: taylor_swift_concert_2025
   💰 Price: $162.00 (Original: $180.00, Discount: $18.00)
   🎟️ Tickets: 2
      1. VIP Section A, Row 1, Seat 1 - $90.00
      2. VIP Section A, Row 1, Seat 2 - $90.00

👤 Managing Subscriber...
✅ Subscriber: ID=1, Email=customer@example.com

📋 Adding Event Subscription...
✅ Event subscription added

📧 Sending Order Confirmation Email...
✅ Email sent successfully!

🔍 Database Verification:
   ✅ Subscriber verified: customer@example.com (created: 2025-10-10 20:30:00)
   📋 Active subscriptions: 1

🎉 Test Completed Successfully!
```

### 2. `sample_order_event.json`
**Kafka Event Structure** - Shows the exact JSON structure expected from Kafka.

**Purpose:**
- 📝 Documents the order event schema
- 🔧 Can be used for manual Kafka message publishing
- 📊 Reference for API integration testing

**Key Fields:**
```json
{
  "OrderID": "ORDER_2025_001234",
  "UserID": "user123@example.com", 
  "EventID": "evt_taylor_swift_concert_2025",
  "tickets": [...]
}
```

### 3. `order_email_test.go` *(Advanced)*
**Configuration-based Test** - Uses actual config files and environment variables.

**Features:**
- 🔧 Loads from `.env` configuration
- 🌐 Tests Keycloak integration 
- 📧 Full SMTP email sending
- 📊 Comprehensive error reporting

### 4. `kafka_order_producer_test.go` *(External Dependency)*
**Kafka Publisher Test** - Requires Kafka setup and Sarama library.

**Note:** Requires `go mod tidy` to install Kafka dependencies.

## 🧪 Running the Tests

### Quick Test (Recommended)
```bash
cd d:\CSE\SE\evershop\ms-scheduling\test
go run simple_order_test.go
```

### With Real Email (Optional)
Update SMTP settings in the code and run:
```bash
go run order_email_test.go
```

## 🔧 Test Prerequisites

### Database Setup
1. Ensure PostgreSQL is running on localhost:5432
2. Database `ms_scheduling` exists with user `ticketly:ticketly`
3. Migrations have been applied:
   ```bash
   cd ..
   go run cmd/migrate/main.go -command=up
   ```

### Verify Database Tables
```sql
-- Check if tables exist
\dt

-- Sample data
SELECT * FROM subscribers LIMIT 5;
SELECT * FROM subscriptions LIMIT 5;
```

## 📧 Expected Email Content

When the test runs successfully, an email similar to this should be generated:

```
Subject: Order Confirmation - TEST_ORDER_1728576000

Dear Customer,

Your order has been confirmed!

Order Details:
- Order ID: TEST_ORDER_1728576000
- Event ID: taylor_swift_concert_2025  
- Session ID: evening_session_main
- Status: CONFIRMED
- Total Price: $162.00
- Created At: 2025-10-10T20:30:00Z

Tickets:
- Seat: VIP Section A, Row 1, Seat 1 (VIP Gold) - $90.00
- Seat: VIP Section A, Row 1, Seat 2 (VIP Gold) - $90.00

Thank you for your purchase!

Best regards,
Ticketly Team
```

## 🐛 Troubleshooting

### Database Connection Issues
```bash
# Check if PostgreSQL is running
pg_ctl status

# Test connection
psql -h localhost -p 5432 -U ticketly -d ms_scheduling
```

### Email Issues
- ⚠️ SMTP errors are expected in test environment
- 💡 Email service will log attempts even if sending fails
- 🔧 Update SMTP credentials in code for real email testing

### Service Initialization Errors
- 🔍 Check if all migrations are applied
- 📋 Verify database schema matches models
- 🔧 Ensure environment variables are properly set

## 🎯 Integration Test Flow

The complete flow tested represents:

1. **Order Created** → Kafka Event `ticketly.order.created`
2. **Consumer Processing** → `internal/kafka/consumer.go`
3. **Subscriber Lookup** → `internal/services/subscriber_service.go`
4. **Keycloak Integration** → `internal/services/keycloak_client.go`
5. **Email Generation** → `internal/services/email_service.go`
6. **Database Updates** → PostgreSQL subscription tables

This matches the production system architecture for order notification handling.