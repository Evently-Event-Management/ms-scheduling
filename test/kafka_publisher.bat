@echo off
REM Kafka Order Event Publisher Test Script (Windows)
REM This script simulates publishing an order.created event to Kafka

echo 🚀 Kafka Order Event Publisher Test (Windows)
echo =============================================

REM Create timestamp
for /f %%i in ('powershell -command "Get-Date -UFormat %%s"') do set TIMESTAMP=%%i
for /f %%i in ('powershell -command "Get-Date -Format yyyy-MM-ddTHH:mm:ssZ"') do set DATETIME=%%i

echo 📦 Creating test order event...

REM Create temporary JSON file
echo { > temp_order.json
echo   "OrderID": "TEST_ORDER_%TIMESTAMP%", >> temp_order.json
echo   "UserID": "windows.test@example.com", >> temp_order.json  
echo   "EventID": "taylor_swift_concert_2025", >> temp_order.json
echo   "SessionID": "evening_session_main", >> temp_order.json
echo   "Status": "CONFIRMED", >> temp_order.json
echo   "SubTotal": 180.00, >> temp_order.json
echo   "DiscountID": "STUDENT", >> temp_order.json
echo   "DiscountCode": "STU10", >> temp_order.json
echo   "DiscountAmount": 18.00, >> temp_order.json
echo   "Price": 162.00, >> temp_order.json
echo   "CreatedAt": "%DATETIME%", >> temp_order.json
echo   "PaymentAT": "%DATETIME%", >> temp_order.json
echo   "tickets": [ >> temp_order.json
echo     { >> temp_order.json
echo       "ticket_id": "tkt_std_001_%TIMESTAMP%", >> temp_order.json
echo       "order_id": "TEST_ORDER_%TIMESTAMP%", >> temp_order.json
echo       "seat_id": "STD_B12", >> temp_order.json
echo       "seat_label": "Standard Section B, Row 1, Seat 12", >> temp_order.json
echo       "colour": "#4A90E2", >> temp_order.json
echo       "tier_id": "standard_tier", >> temp_order.json
echo       "tier_name": "Standard", >> temp_order.json
echo       "price_at_purchase": 81.00, >> temp_order.json
echo       "issued_at": "%DATETIME%", >> temp_order.json
echo       "checked_in": false, >> temp_order.json
echo       "checked_in_time": null >> temp_order.json
echo     }, >> temp_order.json
echo     { >> temp_order.json
echo       "ticket_id": "tkt_std_002_%TIMESTAMP%", >> temp_order.json
echo       "order_id": "TEST_ORDER_%TIMESTAMP%", >> temp_order.json
echo       "seat_id": "STD_B13", >> temp_order.json
echo       "seat_label": "Standard Section B, Row 1, Seat 13", >> temp_order.json
echo       "colour": "#4A90E2", >> temp_order.json
echo       "tier_id": "standard_tier", >> temp_order.json
echo       "tier_name": "Standard", >> temp_order.json
echo       "price_at_purchase": 81.00, >> temp_order.json
echo       "issued_at": "%DATETIME%", >> temp_order.json
echo       "checked_in": false, >> temp_order.json
echo       "checked_in_time": null >> temp_order.json
echo     } >> temp_order.json
echo   ] >> temp_order.json
echo } >> temp_order.json

echo ✅ Order event created in temp_order.json

echo.
echo 📋 Order Event Content:
type temp_order.json

echo.
echo 💡 To manually test with Kafka:
echo 1. Ensure Kafka is running on localhost:9092
echo 2. Create topic if needed:
echo    kafka-topics.bat --create --topic ticketly.order.created --bootstrap-server localhost:9092
echo 3. Publish message:
echo    type temp_order.json ^| kafka-console-producer.bat --broker-list localhost:9092 --topic ticketly.order.created
echo.
echo 🔍 After publishing, check your ms-scheduling service logs for email activity!

pause