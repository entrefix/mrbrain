#!/bin/bash

# Test Production Authentication
# Run this on your production server

SUPABASE_URL="https://rrwefqgtzyhggcssawiw.supabase.co"
SUPABASE_ANON_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InJyd2VmcWd0enloZ2djc3Nhd2l3Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3Njc3MDcwMjcsImV4cCI6MjA4MzI4MzAyN30.VIEKuqAuTGRS9EacjjfQiNgLZPDLAdygkaHT4LyyQuM"

# Use production backend URL (change this to your actual domain)
BACKEND_URL="http://localhost:8099"  # or https://your-domain.com

EMAIL="aruntemme+test2@gmail.com"
PASSWORD="Arun@123"

echo "========================================="
echo "Testing Production Authentication"
echo "========================================="
echo ""

# Step 1: Sign in with Supabase
echo "Step 1: Signing in with Supabase..."
RESPONSE=$(curl -s -X POST \
  "${SUPABASE_URL}/auth/v1/token?grant_type=password" \
  -H "apikey: ${SUPABASE_ANON_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")

ACCESS_TOKEN=$(echo "$RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
    echo "❌ Failed to get access token from Supabase"
    echo "Response: $RESPONSE"
    exit 1
fi

echo "✅ Got access token"
echo ""

# Step 2: Test backend /api/auth/me
echo "Step 2: Testing backend /api/auth/me..."
ME_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X GET \
  "${BACKEND_URL}/api/auth/me" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

HTTP_STATUS=$(echo "$ME_RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
BODY=$(echo "$ME_RESPONSE" | sed '/HTTP_STATUS/d')

echo "HTTP Status: $HTTP_STATUS"
echo "Response Body: $BODY"
echo ""

if [ "$HTTP_STATUS" == "200" ]; then
    echo "✅ Backend authenticated successfully"
elif [ "$HTTP_STATUS" == "500" ]; then
    echo "❌ Backend returned 500 error"
    echo ""
    echo "Check backend logs:"
    echo "  sudo docker compose logs backend --tail=50"
    echo ""
    echo "Check users in database:"
    echo "  sudo docker compose exec backend sqlite3 /data/todomyday.db \"SELECT id, email, supabase_id FROM users WHERE email='${EMAIL}';\""
else
    echo "❌ Backend returned error: $HTTP_STATUS"
fi
