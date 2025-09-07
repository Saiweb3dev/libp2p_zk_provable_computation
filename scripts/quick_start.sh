#!/bin/bash

echo "🚀 Starting ZK Distributed Compute Network..."

# Clean up any existing processes and lock files
cleanup_existing() {
    echo "🧹 Cleaning up existing processes and lock files..."
    
    # Kill processes more aggressively
    pkill -9 -f "bootstrap" 2>/dev/null || true
    pkill -9 -f "peer" 2>/dev/null || true
    
    # On Windows (Git Bash/WSL), also try taskkill
    if command -v taskkill.exe &> /dev/null; then
        taskkill.exe //F //IM bootstrap.exe 2>/dev/null || true
        taskkill.exe //F //IM peer.exe 2>/dev/null || true
    fi
    
    # Kill any process using port 8080 specifically
    if command -v lsof &> /dev/null; then
        PIDS=$(lsof -t -i:8080 2>/dev/null || true)
        if [ ! -z "$PIDS" ]; then
            echo "🔪 Killing processes using port 8080: $PIDS"
            kill -9 $PIDS 2>/dev/null || true
        fi
    fi
    
    # Windows equivalent
    if command -v netstat.exe &> /dev/null; then
        PID=$(netstat -ano | grep :8080 | awk '{print $5}' | head -1)
        if [ ! -z "$PID" ]; then
            echo "🔪 Killing process using port 8080: $PID"
            taskkill.exe //F //PID $PID 2>/dev/null || true
        fi
    fi
    
    # Wait for processes to fully terminate
    sleep 3
    
    # Remove entire zk directory to ensure clean state
    rm -rf ./data/zk 2>/dev/null || true
    
    # Remove any leftover lock files in other locations
    find . -name "LOCK" -type f -delete 2>/dev/null || true
    find . -name "MANIFEST*" -type f -delete 2>/dev/null || true
    
    sleep 2
}

# Cleanup first
cleanup_existing

# Create all necessary directories
echo "📁 Creating directories..."
mkdir -p data/{bootstrap,coordinator,workers,zk}
mkdir -p data/zk/{worker1,worker2,worker3}
mkdir -p logs  # Create logs directory!

# Verify directories were created
if [ ! -d "logs" ]; then
    echo "❌ Failed to create logs directory"
    exit 1
fi

if [ ! -d "data/zk/worker1" ]; then
    echo "❌ Failed to create ZK directories"
    exit 1
fi

# Build binaries if they don't exist
if [ ! -f "./bin/bootstrap" ]; then
    echo "🔨 Building binaries..."
    go build -o bin/bootstrap ./cmd/bootstrap/
    go build -o bin/peer ./cmd/peer/
fi

# Pick bootstrap port (default 8080, override with $BOOTSTRAP_PORT)
BOOTSTRAP_PORT=${BOOTSTRAP_PORT:-8080}

# Start bootstrap server
echo "🌐 Starting bootstrap server on port $BOOTSTRAP_PORT..."
./bin/bootstrap --port=$BOOTSTRAP_PORT > logs/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
echo "Bootstrap PID: $BOOTSTRAP_PID"
sleep 3

# Verify bootstrap is running
if ! kill -0 $BOOTSTRAP_PID 2>/dev/null; then
    echo "❌ Bootstrap server failed to start"
    cat logs/bootstrap.log
    exit 1
fi


# Start 3 worker nodes with ZK enabled
echo "👷 Starting worker nodes..."
./bin/peer --name=worker-1 --port=9101 --worker --zk --zkdb=./data/zk/worker1 --bootstrap=http://localhost:8080 > logs/worker1.log 2>&1 &
WORKER1_PID=$!
sleep 1

./bin/peer --name=worker-2 --port=9102 --worker --zk --zkdb=./data/zk/worker2 --bootstrap=http://localhost:8080 > logs/worker2.log 2>&1 &
WORKER2_PID=$!
sleep 1

./bin/peer --name=worker-3 --port=9103 --worker --zk --zkdb=./data/zk/worker3 --bootstrap=http://localhost:8080 > logs/worker3.log 2>&1 &
WORKER3_PID=$!

echo "Worker PIDs: $WORKER1_PID, $WORKER2_PID, $WORKER3_PID"
sleep 5

# Check if workers are running
workers_running=0
if kill -0 $WORKER1_PID 2>/dev/null; then ((workers_running++)); fi
if kill -0 $WORKER2_PID 2>/dev/null; then ((workers_running++)); fi
if kill -0 $WORKER3_PID 2>/dev/null; then ((workers_running++)); fi

echo "✅ $workers_running workers started successfully"

# Start coordinator with ZK and auto-demo
echo "🎯 Starting coordinator with ZK demo..."

# Use a fresh DB path for coordinator each run
COORD_DB=./data/zk/coordinator_$(date +%s)
mkdir -p "$COORD_DB"
echo "Using database path: $COORD_DB"

# Start coordinator in foreground so we can see the output
./bin/peer --name=coordinator --port=9001 --coordinator --zk --zkdb="$COORD_DB" --bootstrap=http://localhost:8080 --auto-demo

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    
    # Kill all background processes
    if [ ! -z "$BOOTSTRAP_PID" ]; then
        kill $BOOTSTRAP_PID 2>/dev/null || true
    fi
    if [ ! -z "$WORKER1_PID" ]; then
        kill $WORKER1_PID 2>/dev/null || true
    fi
    if [ ! -z "$WORKER2_PID" ]; then
        kill $WORKER2_PID 2>/dev/null || true
    fi
    if [ ! -z "$WORKER3_PID" ]; then
        kill $WORKER3_PID 2>/dev/null || true
    fi
    
    # Kill any remaining processes
    pkill -f "bootstrap\|peer" 2>/dev/null || true
    
    # Wait for graceful shutdown
    sleep 2
    
    echo "🏁 Cleanup complete"
    exit
}

trap cleanup SIGINT SIGTERM
