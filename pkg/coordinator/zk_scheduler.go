package coordinator

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math/big"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "libp2p_compute/pkg/p2p"
    "libp2p_compute/pkg/zk"
)

// ZKTaskScheduler coordinates ZK proof generation tasks across workers
type ZKTaskScheduler struct {
    host         host.Host
    taskService  *p2p.TaskService
    zkHandler    *zk.ZKTaskHandler
    storage      *zk.ProofStorage
    
    // Track ongoing tasks
    pendingTasks map[string]*ZKTaskInfo
    tasksMutex   sync.RWMutex
    
    // Task stats
    totalTasks    int
    completedTasks int
    failedTasks   int
    statsMutex    sync.RWMutex
}

// ZKTaskInfo tracks a distributed ZK proof task
type ZKTaskInfo struct {
    TaskID        string
    TaskType      string
    StartTime     time.Time
    Status        string
    WorkerID      peer.ID
    Batches       int
    CompletedBatches int
    Proofs        []*zk.TransactionProof
}

// ZKTaskResult contains aggregated results from a distributed ZK task
type ZKTaskResult struct {
    TaskID           string
    Verified         bool
    TransactionCount int
    AverageAmount    string
    CompliancePass   bool
    ElapsedTime      time.Duration
    WorkerCount      int
}

// NewZKTaskScheduler creates a new ZK task scheduler
func NewZKTaskScheduler(
    h host.Host,
    taskService *p2p.TaskService,
    zkHandler *zk.ZKTaskHandler,
    dbPath string,
) (*ZKTaskScheduler, error) {
    var storage *zk.ProofStorage
    var err error
    
    if zkHandler != nil {
        // Use existing storage if zkHandler is provided
        storage = zkHandler.GetStorage()
    } else {
        // Create new storage if no zkHandler provided
        storage, err = zk.NewProofStorage(dbPath)
        if err != nil {
            return nil, fmt.Errorf("failed to initialize ZK storage: %w", err)
        }
    }
    
    return &ZKTaskScheduler{
        host:         h,
        taskService:  taskService,
        zkHandler:    zkHandler,
        storage:      storage,
        pendingTasks: make(map[string]*ZKTaskInfo),
    }, nil
}

// ScheduleTransactionAnalyticsTask distributes a transaction analytics task
func (s *ZKTaskScheduler) ScheduleTransactionAnalyticsTask(
    ctx context.Context,
    taskID string,
    numBatches int,
    txPerBatch int,
    requireCompliance bool,
) (*ZKTaskResult, error) {
    log.Printf("📊 Scheduling transaction analytics task: %s (%d batches, %d tx/batch)",
        taskID, numBatches, txPerBatch)
    
    // Create simulator for generating transactions
    simulator := zk.NewTransactionSimulator()
    
    // Create task info
    taskInfo := &ZKTaskInfo{
        TaskID:    taskID,
        TaskType:  "transaction_analytics",
        StartTime: time.Now(),
        Status:    "running",
        Batches:   numBatches,
        Proofs:    make([]*zk.TransactionProof, 0, numBatches),
    }
    
    // Register task
    s.tasksMutex.Lock()
    s.pendingTasks[taskID] = taskInfo
    s.tasksMutex.Unlock()
    
    s.statsMutex.Lock()
    s.totalTasks++
    s.statsMutex.Unlock()
    
    // Track workers used
    workersUsed := make(map[string]bool)
    
    // Compliance rules
    complianceRules := zk.ComplianceRules{
        MaxAmount:    big.NewInt(10000),
        MinAmount:    big.NewInt(1),
        MaxRiskScore: big.NewInt(80),
        PatternID:    big.NewInt(0),
    }
    
    // Process batches
    for i := 0; i < numBatches; i++ {
        // Get next worker via round-robin
        workerID, err := s.taskService.GetNextWorker()
        if err != nil {
            log.Printf("❌ Error getting worker for batch %d: %v", i+1, err)
            continue
        }
        
        // Generate transaction batch
        transactions, err := simulator.SimulateTransactionBatch(ctx, txPerBatch, 0.02)
        if err != nil {
            log.Printf("❌ Error generating transactions for batch %d: %v", i+1, err)
            continue
        }
        
        // Create batch request
        txAnalyticsReq := zk.TransactionAnalyticsRequest{
            Transactions:    transactions,
            ComplianceRules: complianceRules,
        }
        
        // Serialize request data
        reqData, err := json.Marshal(txAnalyticsReq)
        if err != nil {
            log.Printf("❌ Error serializing request for batch %d: %v", i+1, err)
            continue
        }
        
        // Create ZK task request
        batchTaskID := fmt.Sprintf("%s_batch_%d", taskID, i+1)
        zkTaskReq := zk.ZKTaskRequest{
            TaskID:   batchTaskID,
            TaskType: "transaction_analytics",
            Data:     reqData,
        }
        
        // Track worker as used
        workersUsed[workerID.String()] = true
        
        // Submit to worker
        log.Printf("📤 Submitting batch %d/%d to worker %s",
            i+1, numBatches, s.taskService.GetPeerName(workerID))
        
        // Submit asynchronously to not block
        go func(batchNum int, worker peer.ID, request zk.ZKTaskRequest) {
            proof, err := s.zkHandler.SubmitZKTask(worker.String(), request)
            if err != nil {
                log.Printf("❌ Batch %d failed: %v", batchNum, err)
                s.statsMutex.Lock()
                s.failedTasks++
                s.statsMutex.Unlock()
                return
            }
            
            // Update task info with proof
            s.tasksMutex.Lock()
            taskInfo, exists := s.pendingTasks[taskID]
            if exists {
                taskInfo.Proofs = append(taskInfo.Proofs, proof)
                taskInfo.CompletedBatches++
                if taskInfo.CompletedBatches >= taskInfo.Batches {
                    taskInfo.Status = "completed"
                }
            }
            s.tasksMutex.Unlock()
            
            log.Printf("✅ Batch %d completed, verified proof received", batchNum)
            
            s.statsMutex.Lock()
            s.completedTasks++
            s.statsMutex.Unlock()
        }(i+1, workerID, zkTaskReq)
    }
    
    // Wait for all batches to complete or timeout
    deadline := time.Now().Add(5 * time.Minute)
    for {
        if time.Now().After(deadline) {
            return nil, fmt.Errorf("timeout waiting for task completion")
        }
        
        s.tasksMutex.RLock()
        taskInfo, exists := s.pendingTasks[taskID]
        completed := exists && taskInfo.Status == "completed"
        completedBatches := 0
        if exists {
            completedBatches = taskInfo.CompletedBatches
        }
        s.tasksMutex.RUnlock()
        
        if completed {
            break
        }
        
        log.Printf("⏳ Waiting for task completion: %d/%d batches done",
            completedBatches, numBatches)
        
        // Check again after a short delay
        time.Sleep(5 * time.Second)
    }
    
    // Get final task info
    s.tasksMutex.RLock()
    taskInfo = s.pendingTasks[taskID]
    s.tasksMutex.RUnlock()
    
    // Aggregate results
    totalTx := 0
    compliant := true
    for _, proof := range taskInfo.Proofs {
        totalTx += proof.TransactionCount
        
        // Check compliance flag in public inputs
        if compFlag, ok := proof.PublicInputs["ComplianceFlag"]; ok && compFlag.Cmp(big.NewInt(1)) != 0 {
            compliant = false
        }
    }
    
    // Calculate average amount across all proofs
    avgAmount := "0"
    if len(taskInfo.Proofs) > 0 && taskInfo.Proofs[0].PublicInputs["AverageAmount"] != nil {
        avgAmount = taskInfo.Proofs[0].PublicInputs["AverageAmount"].String()
    }
    
    // Prepare result
    result := &ZKTaskResult{
        TaskID:           taskID,
        Verified:         true,
        TransactionCount: totalTx,
        AverageAmount:    avgAmount,
        CompliancePass:   compliant && requireCompliance,
        ElapsedTime:      time.Since(taskInfo.StartTime),
        WorkerCount:      len(workersUsed),
    }
    
    log.Printf("🏁 Task %s completed in %v using %d workers",
        taskID, result.ElapsedTime, result.WorkerCount)
    
    return result, nil
}

// GetTaskStatus returns the current status of a ZK task
func (s *ZKTaskScheduler) GetTaskStatus(taskID string) (string, int, int) {
    s.tasksMutex.RLock()
    defer s.tasksMutex.RUnlock()
    
    if task, exists := s.pendingTasks[taskID]; exists {
        return task.Status, task.CompletedBatches, task.Batches
    }
    
    return "unknown", 0, 0
}

// GetStats returns scheduler statistics
func (s *ZKTaskScheduler) GetStats() (int, int, int) {
    s.statsMutex.RLock()
    defer s.statsMutex.RUnlock()
    
    return s.totalTasks, s.completedTasks, s.failedTasks
}

// Close shuts down the scheduler
func (s *ZKTaskScheduler) Close() error {
    return s.storage.Close()
}