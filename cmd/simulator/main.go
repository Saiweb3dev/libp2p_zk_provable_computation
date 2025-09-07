package main

import (
    "context"
    "encoding/json"
    "flag"
    "log"
    "os"
    "time"
    
    "libp2p_compute/pkg/zk"
)

type SimulationResults struct {
    TotalTransactions      int     `json:"total_transactions"`
    FraudDetected         int     `json:"fraud_detected"`
    AvgProofGenerationTime float64 `json:"avg_proof_generation_time"`
    TotalProofs           int     `json:"total_proofs"`
    SuccessRate           float64 `json:"success_rate"`
    StartTime             string  `json:"start_time"`
    EndTime               string  `json:"end_time"`
}

func main() {
    var (
        coordinatorHost = flag.String("coordinator", "localhost:9001", "Coordinator address")
        tps            = flag.Int("tps", 10, "Transactions per second")
        batchSize      = flag.Int("batch-size", 100, "Batch size for proofs")
        fraudRate      = flag.Float64("fraud-rate", 0.02, "Fraud rate (0-1)")
        duration       = flag.Int("duration", 300, "Simulation duration in seconds")
        mode           = flag.String("mode", "realistic", "Simulation mode (realistic, stress, etc.)")
        outputPath     = flag.String("output", "./simulation_results.json", "Output path")
    )
    flag.Parse()
    
    log.Printf("Starting transaction simulator...")
    log.Printf("Coordinator: %s", *coordinatorHost)
    log.Printf("TPS: %d, Batch Size: %d, Fraud Rate: %.2f", *tps, *batchSize, *fraudRate)
    log.Printf("Mode: %s, Duration: %d seconds", *mode, *duration)
    
    // Create simulator
    simulator := zk.NewTransactionSimulator()
    
    // Create client to coordinator
    client := zk.NewCoordinatorClient(*coordinatorHost)
    
    // Results tracking
    results := &SimulationResults{
        StartTime: time.Now().Format(time.RFC3339),
    }
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*duration)*time.Second)
    defer cancel()
    
    // Transaction channel
    txChan := make(chan zk.TransactionData, *batchSize*2)
    
    // Start transaction generator
    go simulator.SimulateRealTimeStream(ctx, txChan, *tps)
    
    // Process transactions in batches
    batch := make([]zk.TransactionData, 0, *batchSize)
    proofTimes := []time.Duration{}
    
    for {
        select {
        case tx := <-txChan:
            batch = append(batch, tx)
            results.TotalTransactions++
            
            // Process batch when full
            if len(batch) >= *batchSize {
                startTime := time.Now()
                
                // Submit batch for ZK proof generation
                proof, err := client.SubmitTransactionBatch(ctx, batch)
                if err != nil {
                    log.Printf("Error submitting batch: %v", err)
                } else {
                    proofTime := time.Since(startTime)
                    proofTimes = append(proofTimes, proofTime)
                    results.TotalProofs++
                    
                    // Check for fraud in proof results
                    if proof.FraudDetected {
                        results.FraudDetected++
                    }
                    
                    log.Printf("Proof generated in %v (tx: %d, fraud: %v)", 
                        proofTime, len(batch), proof.FraudDetected)
                }
                
                // Clear batch
                batch = batch[:0]
            }
            
        case <-ctx.Done():
            // Process remaining batch
            if len(batch) > 0 {
                _, _ = client.SubmitTransactionBatch(context.Background(), batch)
            }
            
            // Calculate final metrics
            results.EndTime = time.Now().Format(time.RFC3339)
            
            if len(proofTimes) > 0 {
                var total time.Duration
                for _, t := range proofTimes {
                    total += t
                }
                results.AvgProofGenerationTime = float64(total.Milliseconds()) / float64(len(proofTimes))
            }
            
            if results.TotalProofs > 0 {
                results.SuccessRate = float64(results.TotalProofs) / float64((results.TotalTransactions+*batchSize-1) / *batchSize)
            }
            
            // Save results
            saveResults(results, *outputPath)
            
            log.Printf("Simulation complete!")
            log.Printf("Total transactions: %d", results.TotalTransactions)
            log.Printf("Total proofs: %d", results.TotalProofs)
            log.Printf("Fraud detected: %d", results.FraudDetected)
            log.Printf("Avg proof time: %.2fms", results.AvgProofGenerationTime)
            
            return
        }
    }
}

func saveResults(results *SimulationResults, path string) error {
    data, err := json.MarshalIndent(results, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(path, data, 0644)
}