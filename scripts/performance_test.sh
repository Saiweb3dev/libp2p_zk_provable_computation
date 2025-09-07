#!/bin/bash
# Performance testing for ZK proof generation

set -e

# Test configurations
TEST_CONFIGS=(
    "10,100,0.01"    # 10 TPS, 100 batch, 1% fraud
    "50,200,0.02"    # 50 TPS, 200 batch, 2% fraud
    "100,500,0.05"   # 100 TPS, 500 batch, 5% fraud
    "200,1000,0.10"  # 200 TPS, 1000 batch, 10% fraud
)

DURATION=60  # Test duration in seconds
RESULTS_DIR="./performance_results"

# Create results directory
mkdir -p $RESULTS_DIR

echo "Starting ZK Network Performance Tests"
echo "====================================="

for config in "${TEST_CONFIGS[@]}"; do
    IFS=',' read -r tps batch fraud <<< "$config"
    
    echo ""
    echo "Test Configuration:"
    echo "  TPS: $tps"
    echo "  Batch Size: $batch"
    echo "  Fraud Rate: $fraud"
    echo ""
    
    # Run test
    ./bin/simulator \
        --coordinator=localhost:9001 \
        --tps=$tps \
        --batch-size=$batch \
        --fraud-rate=$fraud \
        --duration=$DURATION \
        --output=$RESULTS_DIR/test_${tps}_${batch}_${fraud}.json
    
    # Wait between tests
    sleep 10
done

# Generate summary report
echo ""
echo "Generating Performance Report..."

cat > $RESULTS_DIR/report.md << EOF
# ZK Network Performance Test Report

## Test Date: $(date)

## Results Summary

| TPS | Batch Size | Fraud Rate | Avg Proof Time (ms) | Success Rate | Total Proofs |
|-----|------------|------------|-------------------|--------------|--------------|
EOF

# Parse results and add to report
for config in "${TEST_CONFIGS[@]}"; do
    IFS=',' read -r tps batch fraud <<< "$config"
    
    if [ -f "$RESULTS_DIR/test_${tps}_${batch}_${fraud}.json" ]; then
        avg_time=$(jq -r '.avg_proof_generation_time' $RESULTS_DIR/test_${tps}_${batch}_${fraud}.json)
        success=$(jq -r '.success_rate' $RESULTS_DIR/test_${tps}_${batch}_${fraud}.json)
        total=$(jq -r '.total_proofs' $RESULTS_DIR/test_${tps}_${batch}_${fraud}.json)
        
        echo "| $tps | $batch | $fraud | $avg_time | $success | $total |" >> $RESULTS_DIR/report.md
    fi
done

echo "" >> $RESULTS_DIR/report.md
echo "## Test Duration: ${DURATION} seconds per test" >> $RESULTS_DIR/report.md

echo "Performance report saved to: $RESULTS_DIR/report.md"