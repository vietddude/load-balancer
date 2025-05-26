#!/bin/bash

# Configuration
LB_URL="https://localhost:8080"
NUM_REQUESTS=500
CONCURRENCY=50  # Reduced from 10 to 2
REQUEST_DELAY=0.01  # 500ms delay between requests
ENDPOINT="/"  # Changed from /health to root endpoint

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting load test...${NC}"
echo "Load Balancer URL: $LB_URL"
echo "Total Requests: $NUM_REQUESTS"
echo "Concurrency: $CONCURRENCY"
echo "Request Delay: ${REQUEST_DELAY}s"
echo "Endpoint: $ENDPOINT"
echo "----------------------------------------"

# Function to make a single request
make_request() {
    local start_time=$(date +%s%N)
    local response=$(curl -s -w "\n%{http_code}\n%{time_total}" "$LB_URL$ENDPOINT")
    local end_time=$(date +%s%N)
    local duration=$((($end_time - $start_time)/1000000)) # Convert to milliseconds
    
    # Split response into body, status code, and total time
    local body=$(echo "$response" | head -n 1)
    local status=$(echo "$response" | head -n 2 | tail -n 1)
    local total_time=$(echo "$response" | tail -n 1)
    
    # Print detailed response for debugging
    echo "Request to $LB_URL$ENDPOINT"
    echo "Status: $status"
    echo "Body: $body"
    echo "Total Time: ${total_time}s"
    echo "----------------------------------------"
    
    echo "$status $duration $body"
    sleep $REQUEST_DELAY  # Add delay after each request
}

# Create a temporary file for results
results_file=$(mktemp)

# Run concurrent requests
for ((i=1; i<=$NUM_REQUESTS; i++)); do
    make_request >> "$results_file" &
    
    # Control concurrency
    if (( i % CONCURRENCY == 0 )); then
        wait
        echo -e "${GREEN}Completed $i requests${NC}"
        sleep 1  # Add delay between batches
    fi
done

# Wait for remaining requests
wait

# Process results
echo -e "\n${YELLOW}Test Results:${NC}"
echo "----------------------------------------"

# Count status codes
echo "Status Code Distribution:"
grep -o '^[0-9]*' "$results_file" | sort | uniq -c | while read count code; do
    case $code in
        200) echo -e "${GREEN}$count requests: $code OK${NC}" ;;
        503) echo -e "${RED}$count requests: $code Service Unavailable${NC}" ;;
        502) echo -e "${RED}$count requests: $code Bad Gateway${NC}" ;;
        *) echo "$count requests: $code"
    esac
done

# Calculate average response time
avg_time=$(awk '{sum+=$2} END {print sum/NR}' "$results_file")
echo -e "\nAverage Response Time: ${YELLOW}${avg_time}ms${NC}"

# Clean up
rm "$results_file"

echo -e "\n${GREEN}Load test completed!${NC}"

# Print metrics after test
echo -e "\n${YELLOW}Current Metrics:${NC}"
echo "----------------------------------------"
curl -s "$LB_URL/metrics" 