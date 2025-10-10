#!/bin/bash

# Kafka Order Event Publisher Test Script
# This script simulates publishing an order.created event to Kafka
# Run this ONLY if you have Kafka running locally

echo "🚀 Kafka Order Event Publisher Test"
echo "====================================="

# Check if Kafka is running (optional)
echo "📊 Checking Kafka availability..."

# Create the order event JSON
ORDER_EVENT=$(cat <<EOF
{
  "OrderID": "TEST_ORDER_$(date +%s)",
  "UserID": "test.customer@example.com",
  "EventID": "taylor_swift_concert_2025", 
  "SessionID": "evening_session_main",
  "Status": "CONFIRMED",
  "SubTotal": 200.00,
  "DiscountID": "VIP_EARLY",
  "DiscountCode": "VIP15",
  "DiscountAmount": 30.00,
  "Price": 170.00,
  "CreatedAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "PaymentAT": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "tickets": [
    {
      "ticket_id": "tkt_vip_001_$(date +%s)",
      "order_id": "TEST_ORDER_$(date +%s)",
      "seat_id": "VIP_A1",
      "seat_label": "VIP Section A, Row 1, Seat 1", 
      "colour": "#FFD700",
      "tier_id": "vip_tier",
      "tier_name": "VIP Premium",
      "price_at_purchase": 85.00,
      "issued_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
      "checked_in": false,
      "checked_in_time": null
    },
    {
      "ticket_id": "tkt_vip_002_$(date +%s)",
      "order_id": "TEST_ORDER_$(date +%s)", 
      "seat_id": "VIP_A2",
      "seat_label": "VIP Section A, Row 1, Seat 2",
      "colour": "#FFD700", 
      "tier_id": "vip_tier",
      "tier_name": "VIP Premium",
      "price_at_purchase": 85.00,
      "issued_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
      "checked_in": false,
      "checked_in_time": null
    }
  ]
}
EOF
)

echo "📦 Order Event Created:"
echo "$ORDER_EVENT" | jq '.'

echo ""
echo "📨 Publishing to Kafka topic: ticketly.order.created"

# Use kafka-console-producer to send message (requires Kafka installation)
if command -v kafka-console-producer.sh &> /dev/null; then
    echo "$ORDER_EVENT" | kafka-console-producer.sh \
        --broker-list localhost:9092 \
        --topic ticketly.order.created \
        --property "key.separator=:" \
        --property "parse.key=true" \
        --property "key=TEST_ORDER_$(date +%s)"
    
    echo "✅ Message sent to Kafka!"
else
    echo "⚠️ Kafka tools not found. To manually publish:"
    echo "1. Install Kafka locally"
    echo "2. Create topic: kafka-topics.sh --create --topic ticketly.order.created --bootstrap-server localhost:9092"
    echo "3. Run this script again"
fi

echo ""
echo "💡 Check your ms-scheduling service logs to see if the email was triggered!"
echo "💡 Run: docker logs <your-service-container> or check terminal where main.go is running"