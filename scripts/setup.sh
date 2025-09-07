#!/bin/bash
# Enhanced setup script with ZK proof system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BOOTSTRAP_PORT=${BOOTSTRAP_PORT:-8080}
COORDINATOR_PORT=${COORDINATOR_PORT:-9001}
WORKER_BASE_PORT=${WORKER_BASE_PORT:-9100}
NUM_WORKERS=${NUM_WORKERS:-5}
DATA_DIR=${DATA_DIR:-"./data"}
LOG_DIR=${LOG_DIR:-"./logs"}

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up processes..."
    
    # Kill all processes
    for pid in ${PIDS[@]}; do
        if kill -0 $pid 2>/dev/null; then
            kill $pid 2>/dev/null || true
        fi
    done
    
    # Clean up data directories if requested
    if [ "$CLEAN_DATA" = "true" ]; then
        log_info "Cleaning data directories..."
        rm -rf $DATA_DIR/*
    fi
    
    exit 0
}

# Trap signals
trap cleanup SIGINT SIGTERM EXIT

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --workers) NUM_WORKERS="$2"; shift ;;
        --clean) CLEAN_DATA="true" ;;
        --simulate) RUN_SIMULATION="true" ;;
        --monitor) RUN_MONITOR="true" ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Create directories
log_info "Setting up directories..."
mkdir -p $DATA_DIR/{bootstrap,coordinator,workers,zk}
mkdir -p $LOG_DIR

# Build binaries
log_info "Building binaries..."
go build -o bin/bootstrap ./cmd/bootstrap
go build -o bin/peer ./cmd/peer
go build -o bin/simulator ./cmd/simulator

# Array to store PIDs
declare -a PIDS

# Start bootstrap node
log_info "Starting bootstrap node on port $BOOTSTRAP_PORT..."
./bin/bootstrap \
    --port=$BOOTSTRAP_PORT \
    --data-dir=$DATA_DIR/bootstrap \
    > $LOG_DIR/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
PIDS+=($BOOTSTRAP_PID)
log_info "Bootstrap node started (PID: $BOOTSTRAP_PID)"

# Wait for bootstrap to be ready
sleep 3

# Start worker nodes with ZK capabilities
log_info "Starting $NUM_WORKERS worker nodes..."
for i in $(seq 1 $NUM_WORKERS); do
    WORKER_PORT=$((WORKER_BASE_PORT + i))
    WORKER_NAME="worker-$i"
    
    log_info "Starting $WORKER_NAME on port $WORKER_PORT..."
    
    ./bin/peer \
        --name=$WORKER_NAME \
        --port=$WORKER_PORT \
        --worker \
        --zk \
        --zkdb=$DATA_DIR/zk/$WORKER_NAME \
        --bootstrap=http://localhost:$BOOTSTRAP_PORT \
        > $LOG_DIR/$WORKER_NAME.log 2>&1 &
    
    WORKER_PID=$!
    PIDS+=($WORKER_PID)
    log_info "$WORKER_NAME started (PID: $WORKER_PID)"
    
    # Stagger worker startup
    sleep 1
done

# Wait for workers to connect
sleep 5

# Start coordinator node
log_info "Starting coordinator node on port $COORDINATOR_PORT..."
./bin/peer \
    --name=coordinator \
    --port=$COORDINATOR_PORT \
    --coordinator \
    --zk \
    --zkdb=$DATA_DIR/zk/coordinator \
    --bootstrap=http://localhost:$BOOTSTRAP_PORT \
    > $LOG_DIR/coordinator.log 2>&1 &

COORDINATOR_PID=$!
PIDS+=($COORDINATOR_PID)
log_info "Coordinator started (PID: $COORDINATOR_PID)"

# Wait for network to stabilize
sleep 5

# Network status
log_info "Network setup complete!"
echo ""
echo "==================================="
echo "  DISTRIBUTED ZK COMPUTE NETWORK  "
echo "==================================="
echo "Bootstrap:    http://localhost:$BOOTSTRAP_PORT"
echo "Coordinator:  localhost:$COORDINATOR_PORT"
echo "Workers:      $NUM_WORKERS nodes running"
echo "Data Dir:     $DATA_DIR"
echo "Logs Dir:     $LOG_DIR"
echo ""

# Run simulation if requested
if [ "$RUN_SIMULATION" = "true" ]; then
    log_info "Starting transaction simulation..."
    ./scripts/simulate.sh &
    SIM_PID=$!
    PIDS+=($SIM_PID)
fi

# Run monitor if requested
if [ "$RUN_MONITOR" = "true" ]; then
    log_info "Starting network monitor..."
    ./scripts/monitor.sh &
    MON_PID=$!
    PIDS+=($MON_PID)
fi

# Keep running
log_info "Network is running. Press Ctrl+C to stop."
echo ""
echo "Quick commands:"
echo "  View logs:        tail -f $LOG_DIR/*.log"
echo "  Submit ZK task:   ./scripts/submit_task.sh"
echo "  Check status:     curl http://localhost:$BOOTSTRAP_PORT/status"
echo ""

# Wait indefinitely
while true; do
    sleep 1
done