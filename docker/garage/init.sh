#!/bin/sh
set -e

echo "Waiting for Garage Admin API..."
while ! curl -s http://garage:3903/health > /dev/null; do
  sleep 1
done

echo "Garage Admin API is up. Initializing cluster..."

# 1. Get Node ID
NODE_ID=$(curl -s -H 'Authorization: Bearer admin_token' http://garage:3903/v1/status | jq -r '.node')
echo "Node ID: $NODE_ID"

# 2. Assign Layout
echo "Assigning layout..."
curl -s -X POST -H 'Authorization: Bearer admin_token' -H 'Content-Type: application/json' \
  -d "[{\"id\": \"$NODE_ID\", \"zone\": \"dc1\", \"capacity\": 1073741824, \"tags\": []}]" \
  http://garage:3903/v1/layout > /dev/null

# 3. Apply Layout
echo "Applying layout..."
curl -s -X POST -H 'Authorization: Bearer admin_token' -H 'Content-Type: application/json' \
  -d '{"version": 1}' \
  http://garage:3903/v1/layout/apply > /dev/null

# 4. Create Bucket
echo "Creating bucket 'rainlogs-logs'..."
BUCKET_RES=$(curl -s -X POST -H 'Authorization: Bearer admin_token' -H 'Content-Type: application/json' \
  -d '{"globalAlias": "rainlogs-logs", "localAliases": []}' \
  http://garage:3903/v1/bucket)

BUCKET_ID=$(echo "$BUCKET_RES" | jq -r '.id')

if [ "$BUCKET_ID" = "null" ] || [ -z "$BUCKET_ID" ]; then
  # Bucket might already exist, fetch its ID
  BUCKET_ID=$(curl -s -H 'Authorization: Bearer admin_token' http://garage:3903/v1/bucket | jq -r '.[] | select(.globalAliases[] == "rainlogs-logs") | .id')
fi

echo "Bucket ID: $BUCKET_ID"

# 5. Import Key
echo "Importing access key..."
curl -s -X POST -H 'Authorization: Bearer admin_token' -H 'Content-Type: application/json' \
  -d '{"name": "rainlogs-key", "accessKeyId": "GK4b3b58b73f01eadbc80e2d59", "secretAccessKey": "e528348beba2f13f5eae2632a4ff824d853fefae8c3423c1f16174e307b09d70"}' \
  http://garage:3903/v1/key/import > /dev/null || true

# 6. Allow Key to access Bucket
echo "Granting key access to bucket..."
curl -s -X POST -H 'Authorization: Bearer admin_token' -H 'Content-Type: application/json' \
  -d "{\"bucketId\": \"$BUCKET_ID\", \"accessKeyId\": \"GK4b3b58b73f01eadbc80e2d59\", \"permissions\": {\"read\": true, \"write\": true, \"owner\": true}}" \
  http://garage:3903/v1/bucket/allow > /dev/null

echo "Garage initialization complete!"
