#!/bin/bash

# Start multiple test backend servers on different ports
# Usage: ./run_backends.sh [num_servers]

NUM_SERVERS=${1:-3}  # Default to 3 servers if not specified

echo "Starting $NUM_SERVERS test backend servers..."

for i in $(seq 1 $NUM_SERVERS); do
    PORT=$((8080 + i))
    echo "Starting backend server on port $PORT"
    # Start a simple HTTP server that responds with its port number
    (while true; do
        echo -e "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nBackend Server $i (Port $PORT)" | nc -l -p $PORT
    done) &
done

echo "All backend servers started. Press Ctrl+C to stop all servers."
wait
