#!/bin/bash
# Advanced network monitoring with ZK metrics

set -e

# Configuration
BOOTSTRAP_URL=${BOOTSTRAP_URL:-"http://localhost:8080"}
COORDINATOR_URL=${COORDINATOR_URL:-"http://localhost:9001"}
REFRESH_INTERVAL=${REFRESH_INTERVAL:-5}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'

# Clear screen function
clear_screen() {
    printf "\033c"
}

# Get network status
get_network_status() {
    curl -s "$BOOTSTRAP_URL/status" 2>/dev/null || echo "{}"
}

# Get coordinator metrics
get_coordinator_metrics() {
    curl -s "$COORDINATOR_URL/metrics" 2>/dev/null || echo "{}"
}

# Get ZK proof metrics
get_zk_metrics() {
    curl -s "$COORDINATOR_URL/zk/metrics" 2>/dev/null || echo "{}"
}

# Format number with commas
format_number() {
    printf "%'d" $1
}

# Calculate rate
calculate_rate() {
    local current=$1
    local previous=$2
    local interval=$3
    
    if [ -z "$previous" ]; then
        echo "0"
    else
        echo $(( (current - previous) / interval ))
    fi
}

# Main monitoring loop
main() {
    local prev_tx_count=0
    local prev_proof_count=0
    
    while true; do
        clear_screen
        
        # Header
        echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${CYAN}║${WHITE}         DISTRIBUTED ZK COMPUTE NETWORK MONITOR              ${CYAN}║${NC}"
        echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
        echo ""
        
        # Get current metrics
        network_status=$(get_network_status)
        coordinator_metrics=$(get_coordinator_metrics)
        zk_metrics=$(get_zk_metrics)
        
        # Parse metrics (using jq if available)
        if command -v jq &> /dev/null; then
            # Network metrics
            peer_count=$(echo "$network_status" | jq -r '.peer_count // 0')
            worker_count=$(echo "$network_status" | jq -r '.worker_count // 0')
            coordinator_count=$(echo "$network_status" | jq -r '.coordinator_count // 0')
            
            # Task metrics
            active_tasks=$(echo "$coordinator_metrics" | jq -r '.active_tasks // 0')
            completed_tasks=$(echo "$coordinator_metrics" | jq -r '.completed_tasks // 0')
            failed_tasks=$(echo "$coordinator_metrics" | jq -r '.failed_tasks // 0')
            
            # ZK metrics
            total_proofs=$(echo "$zk_metrics" | jq -r '.total_proofs // 0')
            total_transactions=$(echo "$zk_metrics" | jq -r '.total_transactions // 0')
            avg_proof_time=$(echo "$zk_metrics" | jq -r '.avg_proof_time_ms // 0')
            verification_success_rate=$(echo "$zk_metrics" | jq -r '.verification_success_rate // 0')
            fraud_detected=$(echo "$zk_metrics" | jq -r '.fraud_detected // 0')
            
            # Calculate rates
            tx_rate=$(calculate_rate $total_transactions $prev_tx_count $REFRESH_INTERVAL)
            proof_rate=$(calculate_rate $total_proofs $prev_proof_count $REFRESH_INTERVAL)
            
            # Network Status
            echo -e "${GREEN}▶ Network Status${NC}"
            echo -e "  ${WHITE}Peers:${NC}        $(format_number $peer_count)"
            echo -e "  ${WHITE}Workers:${NC}      $(format_number $worker_count)"
            echo -e "  ${WHITE}Coordinators:${NC} $(format_number $coordinator_count)"
            echo ""
            
            # Task Status
            echo -e "${YELLOW}▶ Task Processing${NC}"
            echo -e "  ${WHITE}Active:${NC}       $(format_number $active_tasks)"
            echo -e "  ${WHITE}Completed:${NC}    $(format_number $completed_tasks)"
            echo -e "  ${WHITE}Failed:${NC}       $(format_number $failed_tasks)"
            echo ""
            
            # ZK Proof Metrics
            echo -e "${MAGENTA}▶ ZK Proof Generation${NC}"
            echo -e "  ${WHITE}Total Proofs:${NC}        $(format_number $total_proofs)"
            echo -e "  ${WHITE}Proof Rate:${NC}          ${proof_rate}/s"
            echo -e "  ${WHITE}Avg Generation Time:${NC} ${avg_proof_time}ms"
            echo -e "  ${WHITE}Verification Rate:${NC}   ${verification_success_rate}%"
            echo ""
            
            # Transaction Analytics
            echo -e "${BLUE}▶ Transaction Analytics${NC}"
            echo -e "  ${WHITE}Total Processed:${NC}     $(format_number $total_transactions)"
            echo -e "  ${WHITE}Transaction Rate:${NC}    ${tx_rate}/s"
            echo -e "  ${WHITE}Fraud Detected:${NC}      $(format_number $fraud_detected)"
            echo ""
            
            # Performance Indicators
            echo -e "${CYAN}▶ Performance${NC}"
            
            # TPS indicator
            if [ $tx_rate -gt 100 ]; then
                echo -e "  ${GREEN}● High Throughput${NC} (${tx_rate} TPS)"
            elif [ $tx_rate -gt 50 ]; then
                echo -e "  ${YELLOW}● Medium Throughput${NC} (${tx_rate} TPS)"
            else
                echo -e "  ${RED}● Low Throughput${NC} (${tx_rate} TPS)"
            fi
            
            # Proof time indicator
            if [ $avg_proof_time -lt 1000 ]; then
                echo -e "  ${GREEN}● Fast Proof Generation${NC} (<1s)"
            elif [ $avg_proof_time -lt 5000 ]; then
                echo -e "  ${YELLOW}● Normal Proof Generation${NC} (1-5s)"
            else
                echo -e "  ${RED}● Slow Proof Generation${NC} (>5s)"
            fi
            
            # Update previous values
            prev_tx_count=$total_transactions
            prev_proof_count=$total_proofs
        else
            echo -e "${RED}Error: jq not installed. Install it for detailed metrics.${NC}"
        fi
        
        # Footer
        echo ""
        echo -e "${CYAN}══════════════════════════════════════════════════════════════${NC}"
        echo -e "${WHITE}Refreshing every ${REFRESH_INTERVAL} seconds... Press Ctrl+C to exit${NC}"
        
        sleep $REFRESH_INTERVAL
    done
}

# Run main function
main