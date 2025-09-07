package main

import (
	"flag"
	"fmt"
	"log"
	"time"
	"context"

  "github.com/libp2p/go-libp2p/core/peer"
  "libp2p_compute/pkg/cli"
  "libp2p_compute/pkg/coordinator"
  "libp2p_compute/pkg/p2p"
  "libp2p_compute/pkg/zk"
)

func main() {
    // Parse command-line flags
    port := flag.Int("port", 0, "listening port (0 for random)")
    name := flag.String("name", "", "display name for this peer")
    peerAddr := flag.String("peer", "", "target peer multiaddr to connect to (optional)")
    worker := flag.Bool("worker", false, "enable worker mode (processes tasks)")
    coordinatorFlag := flag.Bool("coordinator", false, "enable coordinator mode (submits tasks)")
    bootstrap := flag.String("bootstrap", "", "bootstrap server URL (e.g., http://bootstrap:8080)")
    autoDemo := flag.Bool("auto-demo", false, "automatically run demo tasks (coordinator only)")

    // Add ZK-related flags
    zkEnabled := flag.Bool("zk", false, "enable ZK proof processing")
    zkDBPath := flag.String("zkdb", "./data/zk", "ZK proof storage path")

    flag.Parse()

    // Auto-assign roles based on name if not specified
    if !*worker && !*coordinatorFlag {
        if *name == "coordinator" {
            *coordinatorFlag = true
        } else {
            *worker = true // Default to worker
        }
    }

    // Auto-assign name if not provided
    if *name == "" {
        if *coordinatorFlag {
            *name = "coordinator"
        } else {
            *name = fmt.Sprintf("worker-%d", *port)
        }
    }

    // Validate required parameters
    if !*worker && !*coordinatorFlag {
        log.Fatal("❌ Please specify at least one mode: --worker or --coordinator")
    }

    // Create the libp2p host
    host, cancel, err := p2p.NewHost(*port)
    if err != nil {
        log.Fatalf("❌ Failed to create host: %v", err)
    }
    defer cancel()

    // Initialize services
    chatService := p2p.NewChatService(host, *name)
    connectionManager := p2p.NewConnectionManager(host, chatService)
    taskService := p2p.NewTaskService(host, chatService, *worker, *coordinatorFlag)

    // Initialize ZK handler if enabled
    var zkHandler *zk.ZKTaskHandler
    var zkScheduler *coordinator.ZKTaskScheduler

    if *zkEnabled {
        zkHandler, err = zk.NewZKTaskHandler(host, *zkDBPath)
        if err != nil {
            log.Fatalf("❌ Failed to initialize ZK handler: %v", err)
        }

        if *coordinatorFlag {
            zkScheduler, err = coordinator.NewZKTaskScheduler(host, taskService, zkHandler, *zkDBPath)
            if err != nil {
                log.Fatalf("❌ Failed to initialize ZK scheduler: %v", err)
            }
        }

        log.Printf("✅ ZK proof processing enabled (storage: %s)", *zkDBPath)
    }

    // Determine role string for display
    roleStr := ""
    role := ""
    if *worker && *coordinatorFlag {
        roleStr = " (Worker & Coordinator)"
        role = "hybrid"
    } else if *worker {
        roleStr = " (Worker)"
        role = "worker"
    } else if *coordinatorFlag {
        roleStr = " (Coordinator)"
        role = "coordinator"
    }

    log.Printf("✅ Peer started! Name: %s%s", *name, roleStr)
    connectionManager.PrintHostInfo()

    // Bootstrap discovery if URL provided
    if *bootstrap != "" {
        bootstrapClient := p2p.NewBootstrapClient(*bootstrap, host, *name, role)

        // Wait a moment for the bootstrap server to be ready
        time.Sleep(2 * time.Second)

        // Register with bootstrap
        if err := bootstrapClient.RegisterWithBootstrap(); err != nil {
            log.Printf("⚠️  Failed to register with bootstrap: %v", err)
        }

        // Wait a moment for other peers to register
        time.Sleep(3 * time.Second)

        // Connect to known peers
        if err := bootstrapClient.ConnectToKnownPeers(); err != nil {
            log.Printf("⚠️  Failed to connect to peers: %v", err)
        }

        // Exchange names with connected peers
        connectedPeers := connectionManager.GetConnectedPeers()
        for _, peerID := range connectedPeers {
            if err := chatService.SendName(peerID); err != nil {
                log.Printf("⚠️  Failed to exchange names with %s: %v", peerID, err)
            }
        }

        // Start periodic registration
        bootstrapClient.StartPeriodicRegistration()
    }

    // Connect to initial peer if specified (manual override)
    if *peerAddr != "" {
        if err := connectionManager.ConnectToPeer(*peerAddr); err != nil {
            log.Printf("⚠️  Failed to connect to initial peer: %v", err)
        }
    }

    // Auto-demo mode for coordinators
    if *autoDemo && *coordinatorFlag {
        if *zkEnabled && zkScheduler != nil {
            go runZKDemo(zkScheduler, *name)
        } else {
            go runAutoDemo(taskService, *name)
        }
    }

    // Start the CLI interface
    chatCLI := cli.NewChatCLI(chatService, connectionManager, taskService)
    chatCLI.Start()
}

// runZKDemo demonstrates ZK proof functionality
func runZKDemo(zkScheduler *coordinator.ZKTaskScheduler, peerName string) {
    // Wait for workers to connect
    time.Sleep(10 * time.Second)

    log.Printf("🔐 Starting ZK proof demo for %s", peerName)

    // Generate a task ID
    taskID := fmt.Sprintf("zk_demo_%d", time.Now().UnixNano())

    // Create context
    ctx := context.Background()

    // Schedule a transaction analytics task
    result, err := zkScheduler.ScheduleTransactionAnalyticsTask(
        ctx,
        taskID,
        3,         // 3 batches
        100,       // 100 transactions per batch
        true,      // require compliance
    )

    if err != nil {
        log.Printf("❌ ZK demo failed: %v", err)
        return
    }

    log.Printf("\n🎉 ZK TRANSACTION ANALYTICS COMPLETED:")
    log.Printf("====================================")
    log.Printf("Task ID: %s", result.TaskID)
    log.Printf("Total Time: %v", result.ElapsedTime)
    log.Printf("Total Transactions: %d", result.TransactionCount)
    log.Printf("Average Amount: %s", result.AverageAmount)
    log.Printf("Compliance: %v", result.CompliancePass)
    log.Printf("Workers Used: %d", result.WorkerCount)
    log.Printf("Proofs Verified: %v", result.Verified)

    // Show scheduler stats
    total, completed, failed := zkScheduler.GetStats()
    log.Printf("\n📊 ZK Scheduler Stats:")
    log.Printf("Total Tasks: %d", total)
    log.Printf("Completed: %d", completed)
    log.Printf("Failed: %d", failed)

    log.Printf("\n🏁 ZK demo completed!")
}

// Update runAutoDemo function for prime verification test
func runAutoDemo(taskService *p2p.TaskService, peerName string) {
	// Wait for workers to connect
	time.Sleep(10 * time.Second)

	log.Printf("🤖 Starting prime verification demo for %s", peerName)

	// 10 prime checking tasks for verification
	primeNumbers := []int64{97, 101, 103, 107, 109, 113, 127, 131, 137, 139}

	startTime := time.Now()
	var submittedTasks []string

	log.Printf("📋 Submitting %d prime checking tasks...", len(primeNumbers))

	// Submit all prime checking tasks
	for i, number := range primeNumbers {
		taskID, err := taskService.SubmitTask(p2p.TaskTypePrime, number, nil, 1)
		if err != nil {
			log.Printf("❌ Failed to submit prime task %d: %v", i+1, err)
			continue
		}
		submittedTasks = append(submittedTasks, taskID)
		log.Printf("📤 Submitted prime task %d/%d: Check if %d is prime (Task ID: %s)",
			i+1, len(primeNumbers), number, taskID)

		// Small delay between submissions to see round-robin in action
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for all tasks to complete and then show statistics
	go func() {
		time.Sleep(30 * time.Second) // Wait for tasks to complete

		stats := taskService.GetStatistics()
		log.Printf("\n📊 PRIME VERIFICATION TEST RESULTS:")
		log.Printf("================================")
		log.Printf("Total Time: %v", time.Since(startTime))
		log.Printf("Tasks Submitted: %d", stats.TotalSubmitted)
		log.Printf("Tasks Completed: %d", stats.TotalCompleted)
		log.Printf("Tasks Failed: %d", stats.TotalFailed)
		log.Printf("\n📈 Worker Distribution:")

		for workerID, workerStats := range stats.WorkerStats {
			workerName := taskService.GetPeerName(peer.ID(workerID))
			log.Printf("  %s:", workerName)
			log.Printf("    Assigned: %d", workerStats.TasksAssigned)
			log.Printf("    Completed: %d", workerStats.TasksCompleted)
			log.Printf("    Failed: %d", workerStats.TasksFailed)
			log.Printf("    Last Active: %v", workerStats.LastActive.Format("15:04:05"))
		}

		log.Printf("\n🏁 Prime verification demo completed!")
	}()
}
