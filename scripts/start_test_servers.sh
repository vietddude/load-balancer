#!/bin/bash

# Kill any existing test servers
pkill -f "testserver" || true

# Start three test servers on different ports
for port in 8081 8082 8083; do
    go run cmd/testserver/main.go --port $port &
    echo "Started test server on port $port"
done

echo "All test servers started. Press Ctrl+C to stop all servers."
wait 