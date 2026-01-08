#!/bin/bash

# Test Supabase Authentication and Backend Sync

SUPABASE_URL="https://rrwefqgtzyhggcssawiw.supabase.co"
SUPABASE_ANON_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InJyd2VmcWd0enloZ2djc3Nhd2l3Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3Njc3MDcwMjcsImV4cCI6MjA4MzI4MzAyN30.VIEKuqAuTGRS9EacjjfQiNgLZPDLAdygkaHT4LyyQuM"
BACKEND_URL="http://localhost:8099"

EMAIL="aruntemme+test2@gmail.com"
PASSWORD="Arun@123"

echo "========================================="
echo "Testing Authentication Flow"
echo "========================================="
echo ""

# Step 1: Sign in with Supabase
echo "Step 1: Signing in with Supabase..."
RESPONSE=$(curl -s -X POST \
  "${SUPABASE_URL}/auth/v1/token?grant_type=password" \
  -H "apikey: ${SUPABASE_ANON_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")

echo "Supabase Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# Extract access token
ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token' 2>/dev/null)

if [ "$ACCESS_TOKEN" == "null" ] || [ -z "$ACCESS_TOKEN" ]; then
    echo "❌ Failed to get access token from Supabase"
    echo "Response: $RESPONSE"
    exit 1
fi

echo "✅ Got access token: ${ACCESS_TOKEN:0:50}..."
echo ""

# Step 2: Test backend /api/auth/me
echo "Step 2: Testing backend /api/auth/me..."
ME_RESPONSE=$(curl -s -X GET \
  "${BACKEND_URL}/api/auth/me" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

echo "Backend /api/auth/me Response:"
echo "$ME_RESPONSE" | jq '.' 2>/dev/null || echo "$ME_RESPONSE"
echo ""

# Check if successful
if echo "$ME_RESPONSE" | grep -q "error"; then
    echo "❌ Backend returned error"
    echo ""
    echo "Check backend logs with:"
    echo "  docker-compose logs backend | tail -50"
else
    echo "✅ Backend authenticated successfully"
fi

echo ""
echo "========================================="
echo "Decoded JWT Token Claims:"
echo "========================================="
# Decode JWT payload (middle part)
PAYLOAD=$(echo "$ACCESS_TOKEN" | cut -d'.' -f2)
# Add padding if needed
case ${#PAYLOAD} in
    *[^0-9]*)
        echo "$PAYLOAD" | base64 -d 2>/dev/null | jq '.' || echo "Failed to decode"
        ;;
    *)
        PADDED="${PAYLOAD}$(printf '%0*d' $((4-${#PAYLOAD}%4)) 0)"
        echo "$PADDED" | base64 -d 2>/dev/null | jq '.' || echo "Failed to decode"
        ;;
esac
