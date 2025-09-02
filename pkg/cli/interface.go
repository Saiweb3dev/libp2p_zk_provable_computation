package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"libp2p_compute/pkg/p2p"
)

// ChatCLI handles the command-line interface for the chat application
type ChatCLI struct {
	chatService       *p2p.ChatService
	connectionManager *p2p.ConnectionManager
	taskService       *p2p.TaskService
	reader            *bufio.Reader
}

// NewChatCLI creates a new CLI instance
func NewChatCLI(cs *p2p.ChatService, cm *p2p.ConnectionManager, ts *p2p.TaskService) *ChatCLI {
	return &ChatCLI{
		chatService:       cs,
		connectionManager: cm,
		taskService:       ts,
		reader:            bufio.NewReader(os.Stdin),
	}
}

// Start begins the interactive CLI loop
func (cli *ChatCLI) Start() {
	fmt.Println("\n🚀 P2P Chat & Task Processing CLI started!")
	fmt.Println("Chat Commands:")
	fmt.Println("  <name|peerID> <message>     - Send message to peer")
	fmt.Println("\nTask Commands:")
	fmt.Println("  /task prime <number>        - Check if number is prime")
	fmt.Println("  /task factorial <number>    - Calculate factorial")
	fmt.Println("  /task fibonacci <number>    - Calculate fibonacci")
	fmt.Println("  /task sum <n1,n2,n3...>     - Calculate sum of numbers")
	fmt.Println("\nSystem Commands:")
	fmt.Println("  /peers                      - List connected peers")
	fmt.Println("  /connect <multiaddr>        - Connect to a peer")
	fmt.Println("  /queue                      - Show task queue status")
	fmt.Println("  /pending                    - Show pending tasks")
	fmt.Println("  /help                       - Show this help")
	fmt.Println("  /quit                       - Exit the application")
	fmt.Println()

	for {
		fmt.Print(">> ")
		line, err := cli.reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if err := cli.processCommand(line); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}
}

// processCommand handles different types of user commands
func (cli *ChatCLI) processCommand(line string) error {
	// Handle special commands that start with /
	if strings.HasPrefix(line, "/") {
		return cli.handleSpecialCommand(line)
	}

	// Handle regular chat messages
	return cli.handleChatMessage(line)
}

// the method signature to return error instead of bool
func (cli *ChatCLI) handleSpecialCommand(input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	command := parts[0]

	switch command {
	case "/help":
		cli.showHelp()
		return nil

	case "/peers":
		cli.listPeers()
		return nil

	case "/stats":
		cli.showStatistics()
		return nil

	case "/status":
		if len(parts) >= 2 {
			cli.showTaskStatus(parts[1])
		} else {
			cli.showAllTaskStatuses()
		}
		return nil

	case "/queue":
		cli.showQueueStatus()
		return nil

	case "/connect":
		if len(parts) >= 2 {
			addr := strings.Join(parts[1:], " ")
			if err := cli.connectionManager.ConnectToPeer(addr); err != nil {
				return fmt.Errorf("failed to connect to peer: %v", err)
			} else {
				fmt.Printf("✅ Connection attempt initiated\n")
			}
		} else {
			return fmt.Errorf("usage: /connect <multiaddr>")
		}
		return nil

	case "/task":
		if len(parts) >= 2 {
			return cli.handleTaskCommand(parts[1:])
		} else {
			return fmt.Errorf("usage: /task <type> <args>\nTypes: prime <number>, factorial <number>, fibonacci <number>, sum <numbers...>")
		}

	case "/quit", "/exit":
		fmt.Println("👋 Goodbye!")
		os.Exit(0)

	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	return nil
}

// Add the missing showQueueStatus method
func (cli *ChatCLI) showQueueStatus() {
	queueLen, queueTasks := cli.taskService.GetQueueStatus()

	fmt.Printf("\n📥 Task Queue Status:\n")
	fmt.Printf("====================\n")
	fmt.Printf("Queue Length: %d\n", queueLen)

	if queueLen > 0 {
		fmt.Printf("\nQueued Tasks:\n")
		for i, task := range queueTasks {
			fmt.Printf("  %d. %s (%s) - Priority: %d - Created: %v\n",
				i+1, task.ID, task.Type, task.Priority, task.Created.Format("15:04:05"))
		}
	} else {
		fmt.Printf("No tasks in queue\n")
	}
	fmt.Println()
}

func (cli *ChatCLI) handleTaskCommand(parts []string) error {
	if len(parts) == 0 {
		return fmt.Errorf("usage: /task <type> <args>")
	}

	taskType := parts[0]

	switch taskType {
	case "prime":
		if len(parts) >= 2 {
			if number, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				if taskID, err := cli.taskService.SubmitTask(p2p.TaskTypePrime, number, nil, 1); err == nil {
					fmt.Printf("✅ Submitted prime task: %s\n", taskID)
					return nil
				} else {
					return fmt.Errorf("failed to submit task: %v", err)
				}
			} else {
				return fmt.Errorf("invalid number: %s", parts[1])
			}
		} else {
			return fmt.Errorf("usage: /task prime <number>")
		}

	case "factorial":
		if len(parts) >= 2 {
			if number, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				if taskID, err := cli.taskService.SubmitTask(p2p.TaskTypeFactorial, number, nil, 1); err == nil {
					fmt.Printf("✅ Submitted factorial task: %s\n", taskID)
					return nil
				} else {
					return fmt.Errorf("failed to submit task: %v", err)
				}
			} else {
				return fmt.Errorf("invalid number: %s", parts[1])
			}
		} else {
			return fmt.Errorf("usage: /task factorial <number>")
		}

	case "fibonacci":
		if len(parts) >= 2 {
			if number, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				if taskID, err := cli.taskService.SubmitTask(p2p.TaskTypeFibonacci, number, nil, 1); err == nil {
					fmt.Printf("✅ Submitted fibonacci task: %s\n", taskID)
					return nil
				} else {
					return fmt.Errorf("failed to submit task: %v", err)
				}
			} else {
				return fmt.Errorf("invalid number: %s", parts[1])
			}
		} else {
			return fmt.Errorf("usage: /task fibonacci <number>")
		}

	case "sum":
		if len(parts) >= 2 {
			var numbers []int64
			for _, numStr := range parts[1:] {
				if num, err := strconv.ParseInt(numStr, 10, 64); err == nil {
					numbers = append(numbers, num)
				} else {
					return fmt.Errorf("invalid number: %s", numStr)
				}
			}
			if taskID, err := cli.taskService.SubmitTask(p2p.TaskTypeSum, 0, numbers, 1); err == nil {
				fmt.Printf("✅ Submitted sum task: %s\n", taskID)
				return nil
			} else {
				return fmt.Errorf("failed to submit task: %v", err)
			}
		} else {
			return fmt.Errorf("usage: /task sum <number1> <number2>")
		}

	default:
		return fmt.Errorf("unknown task type: %s\nAvailable types: prime, factorial, fibonacci, sum", taskType)
	}
	return nil
}

// handleChatMessage processes regular chat messages
func (cli *ChatCLI) handleChatMessage(line string) error {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("usage: <peerName|peerID> <message>")
	}

	targetStr, message := parts[0], parts[1]

	// Try to find peer by name first
	targetID, found := cli.chatService.FindPeerByName(targetStr)

	// If not found by name, try to decode as peer ID
	if !found {
		tid, err := peer.Decode(targetStr)
		if err != nil {
			return fmt.Errorf("unknown name or invalid peerID: %s", targetStr)
		}
		targetID = tid
	}

	// Send the message
	if err := cli.chatService.SendMessage(targetID, message); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	// Show confirmation
	targetName := cli.chatService.GetPeerName(targetID)
	fmt.Printf("📤 Message sent to %s\n", targetName)

	return nil
}

// Add these new methods to ChatCLI
func (cli *ChatCLI) showStatistics() {
	stats := cli.taskService.GetStatistics()

	fmt.Printf("\n📊 Task Distribution Statistics:\n")
	fmt.Printf("================================\n")
	fmt.Printf("Total Submitted: %d\n", stats.TotalSubmitted)
	fmt.Printf("Total Completed: %d\n", stats.TotalCompleted)
	fmt.Printf("Total Failed: %d\n", stats.TotalFailed)

	if len(stats.WorkerStats) > 0 {
		fmt.Printf("\n👷 Worker Statistics:\n")
		for workerID, workerStats := range stats.WorkerStats {
			workerName := cli.chatService.GetPeerName(peer.ID(workerID))
			fmt.Printf("  %s:\n", workerName)
			fmt.Printf("    Assigned: %d\n", workerStats.TasksAssigned)
			fmt.Printf("    Completed: %d\n", workerStats.TasksCompleted)
			fmt.Printf("    Failed: %d\n", workerStats.TasksFailed)
			fmt.Printf("    Last Active: %v\n", workerStats.LastActive.Format("15:04:05"))
		}
	}
	fmt.Println()
}

func (cli *ChatCLI) showTaskStatus(taskID string) {
	status := cli.taskService.GetTaskStatus(taskID)
	fmt.Printf("Task %s status: %s\n", taskID, status)
}

func (cli *ChatCLI) showAllTaskStatuses() {
	fmt.Println("📋 All Task Statuses:")

	// Get pending tasks
	pendingTasks := cli.taskService.GetPendingTasks()
	if len(pendingTasks) > 0 {
		fmt.Println("\n🔄 Pending Tasks:")
		for taskID, task := range pendingTasks {
			status := cli.taskService.GetTaskStatus(taskID)
			workerName := cli.chatService.GetPeerName(task.WorkerID)
			fmt.Printf("  %s: %s (assigned to %s)\n", taskID, status, workerName)
		}
	}

	// Get queue status if this is a worker
	queueLen, queueTasks := cli.taskService.GetQueueStatus()
	if queueLen > 0 {
		fmt.Printf("\n📥 Queue Status: %d tasks\n", queueLen)
		for i, task := range queueTasks {
			fmt.Printf("  %d. %s (%s) - Priority: %d\n", i+1, task.ID, task.Type, task.Priority)
		}
	}
}

// showHelp displays available commands
func (cli *ChatCLI) showHelp() {
	fmt.Println("\n🚀 P2P Chat & Task Processing CLI Commands:")
	fmt.Println("==========================================")
	fmt.Println("Chat Commands:")
	fmt.Println("  <name|peerID> <message>     - Send message to peer")
	fmt.Println()
	fmt.Println("Task Commands:")
	fmt.Println("  /task prime <number>        - Check if number is prime")
	fmt.Println("  /task factorial <number>    - Calculate factorial")
	fmt.Println("  /task fibonacci <number>    - Calculate fibonacci")
	fmt.Println("  /task sum <numbers...>      - Calculate sum of numbers")
	fmt.Println()
	fmt.Println("System Commands:")
	fmt.Println("  /peers                      - List connected peers")
	fmt.Println("  /connect <multiaddr>        - Connect to a peer")
	fmt.Println("  /stats                      - Show task statistics")
	fmt.Println("  /status [taskID]            - Show task status")
	fmt.Println("  /queue                      - Show task queue status")
	fmt.Println("  /help                       - Show this help")
	fmt.Println("  /quit or /exit              - Exit the application")
	fmt.Println()
}

// showPendingTasks displays pending tasks for coordinators
func (cli *ChatCLI) showPendingTasks() {
	pending := cli.taskService.GetPendingTasks()

	fmt.Printf("\n⏳ Pending Tasks: %d\n", len(pending))
	if len(pending) == 0 {
		fmt.Println("   No pending tasks")
		return
	}

	for taskID, task := range pending {
		elapsed := time.Since(task.StartTime)
		fmt.Printf("   %s [%s] - %v elapsed\n",
			taskID, task.Task.Type, elapsed.Round(time.Millisecond))
	}
	fmt.Println()
}

// listPeers shows all connected peers and their names
func (cli *ChatCLI) listPeers() {
	peers := cli.chatService.GetAllPeers()

	fmt.Printf("\n👥 Connected Peers:\n")
	fmt.Printf("===================\n")

	if len(peers) == 0 {
		fmt.Printf("No peers connected\n")
	} else {
		for peerID, name := range peers {
			fmt.Printf("  %s (%s)\n", name, peerID)
		}
	}
	fmt.Println()
}
