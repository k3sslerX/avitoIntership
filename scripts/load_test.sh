#!/bin/bash

set -e

echo "Starting load testing"

BASE_URL="http://localhost:8080"

echo "Creating test data..."
TEAM_RESP=$(curl -s -X POST -H "Content-Type: application/json" -d '{
  "team_name": "load-test-team",
  "members": [
    {"user_id": "lt1", "username": "Load Test User 1", "is_active": true},
    {"user_id": "lt2", "username": "Load Test User 2", "is_active": true},
    {"user_id": "lt3", "username": "Load Test User 3", "is_active": true}
  ]
}' $BASE_URL/team/add)

echo "Response: $TEAM_RESP"

echo "Creating PRs for load testing..."
for i in {1..20}; do
  curl -s -X POST -H "Content-Type: application/json" -d "{
    \"pull_request_id\": \"load-pr-$i\",
    \"pull_request_name\": \"Load Test PR $i\",
    \"author_id\": \"lt1\"
  }" $BASE_URL/pullRequest/create > /dev/null
  echo -n "."
done
echo ""
echo "PRs created"

echo '{
  "pull_request_id": "stress-test-pr",
  "pull_request_name": "Stress Test PR",
  "author_id": "lt1"
}' > post_data.json

echo "Running load test for PR creation..."
ab -n 50 -c 5 -T "application/json" -p post_data.json -l "$BASE_URL/pullRequest/create"

echo "Running load test for PR retrieval..."
ab -n 100 -c 10 -l "$BASE_URL/pullRequest/get?pull_request_id=load-pr-1"

echo "Running load test for statistics..."
ab -n 200 -c 20 -l "$BASE_URL/stats"

echo "Running load test for team retrieval..."
ab -n 100 -c 10 -l "$BASE_URL/team/get?team_name=load-test-team"

rm -f post_data.json

echo "Load testing completed"