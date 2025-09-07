#!/bin/bash
# Transaction simulation script

set -e

# Configuration
COORDINATOR_HOST=${COORDINATOR_HOST:-"localhost:9001"}
TPS=${TPS:-10}  # Transactions per second
BATCH_SIZE=${BATCH_SIZE:-100}
FRAUD_RATE=${FRAUD_RATE:-0.02}  # 2% fraud rate
SIMULATION_DURATION=${SIMULATION_DURATION:-300}  # 5 minutes

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[SIMULATOR]${NC} $1"
}

log_metric() {
    echo -e "${YELLOW}[METRIC]${NC} $1"
}

# Start simulator
log_info "Starting transaction simulator..."
log_info "Configuration:"
log_info "  - TPS: $TPS"
log_info "  - Batch Size: $BATCH_SIZE"
log_info "  - Fraud Rate: ${FRAUD_RATE}%"
log_info "  - Duration: ${SIMULATION_DURATION}s"

# Run the simulator binary
./bin/simulator \
    --coordinator=$COORDINATOR_HOST \
    --tps=$TPS \
    --batch-size=$BATCH_SIZE \
    --fraud-rate=$FRAUD_RATE \
    --duration=$SIMULATION_DURATION \
    --mode=realistic \
    --output=./data/simulation_results.json

# Process results
if [ -f "./data/simulation_results.json" ]; then
    log_info "Simulation complete. Processing results..."
    
    # Extract metrics using jq (install if not available)
    if command -v jq &> /dev/null; then
        TOTAL_TX=$(jq '.total_transactions' ./data/simulation_results.json)
        FRAUD_DETECTED=$(jq '.fraud_detected' ./data/simulation_results.json)
        AVG_PROOF_TIME=$(jq '.avg_proof_generation_time' ./data/simulation_results.json)
        
        log_metric "Total Transactions: $TOTAL_TX"
        log_metric "Fraud Detected: $FRAUD_DETECTED"
        log_metric "Avg Proof Time: ${AVG_PROOF_TIME}ms"
    fi
fi

log_info "Simulation finished!"