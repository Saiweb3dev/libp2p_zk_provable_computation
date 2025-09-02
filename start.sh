#!/bin/bash
# test.sh

echo "🚀 Starting P2P Task System Test..."

# Start bootstrap
./bin/bootstrap --port=8080 &
BOOTSTRAP_PID=$!
sleep 2

# Start workers
./bin/peer --name=worker-1 --port=9002 --worker --bootstrap=http://localhost:8080 &
WORKER1_PID=$!

./bin/peer --name=worker-2 --port=9003 --worker --bootstrap=http://localhost:8080 &
WORKER2_PID=$!

./bin/peer --name=worker-3 --port=9004 --worker --bootstrap=http://localhost:8080 &
WORKER3_PID=$!

sleep 5

# Start coordinator with auto-demo
echo "🎯 Starting coordinator with auto-demo..."
./bin/peer --name=coordinator --port=9001 --coordinator --bootstrap=http://localhost:8080 --auto-demo

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up..."
    kill $BOOTSTRAP_PID $WORKER1_PID $WORKER2_PID $WORKER3_PID 2>/dev/null
    exit
}

# Set trap for cleanup
trap cleanup SIGINT SIGTERM

# Wait
wait