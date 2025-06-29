#!/bin/bash

# Generate dynamic values
ORDER_UID="$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -d'-' -f1)"
TRACK_NUMBER="WBILM$(openssl rand -hex 3 | tr '[:lower:]' '[:upper:]')"
CHRT_ID=$(( (RANDOM % 1000000) + 9000000 ))
CUSTOMER_ID="$(uuidgen | cut -c1-8)"
TRANSACTION="$ORDER_UID"
DATE_CREATED=$(date -u +"%Y-%m-%dT%H:%M:%SZ")


# Create JSON payload inline with updated fields
read -r -d '' PAYLOAD <<EOF
{
  "order_uid": "$ORDER_UID",
  "track_number": "$TRACK_NUMBER",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "$TRANSACTION",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": $CHRT_ID,
      "track_number": "$TRACK_NUMBER",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "$CUSTOMER_ID",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "$DATE_CREATED",
  "oof_shard": "1"
}
EOF

curl -X POST http://localhost:8080/order \
     -H "Content-Type: application/json" \
     -d "$PAYLOAD"
