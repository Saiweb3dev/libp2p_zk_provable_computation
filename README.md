# LibP2P CLI Chat & Task Processing System

This is a distributed P2P chat and task processing system built with Go and libp2p. Here's the project structure and key components:

## Project Overview
A peer-to-peer system that combines chat functionality with distributed task processing, featuring automatic peer discovery through a bootstrap server.

## Architecture
- **Bootstrap Server**: Central discovery service for peer registration and discovery
- **Peers**: Individual nodes that can act as workers, coordinators, or both
- **Task Distribution**: Round-robin task assignment with priority queuing

## Key Files & Components

### Core P2P Infrastructure
- `pkg/p2p/peer.go` - Creates libp2p hosts with Noise security + TCP + Mplex
- `pkg/p2p/connection.go` - Manages peer connections and discovery
- `pkg/p2p/discovery.go` - Bootstrap client for peer registration/discovery

### Communication Systems
- `pkg/p2p/chat.go` - Chat messaging with name exchange protocol
- `pkg/p2p/task.go` - Distributed task processing system

### User Interface
- `pkg/cli/interface.go` - Interactive CLI for chat and task commands

### Applications
- `cmd/peer/main.go` - Main peer application with role configuration
- `cmd/bootstrap/main.go` - Bootstrap discovery server

## Getting Started

### Prerequisites
- Go 1.19 or later
- Linux/macOS/Windows (tested on Linux)
- Network connectivity for P2P communication

### Building the Project
Navigate to the project root and build the binaries:

```bash
# Build the bootstrap server
go build -o bin/bootstrap ./cmd/bootstrap/

# Build the peer application
go build -o bin/peer ./cmd/peer/
```

This creates executable binaries in the `bin/` directory.

### Running the System
The system consists of a bootstrap server and multiple peer nodes. Start them in the following order:

1. **Start the Bootstrap Server** (acts as a discovery hub):
   ```bash
   ./bin/bootstrap --port=8080
   ```
   - Runs an HTTP server on port 8080 for peer registration and discovery.
   - Includes endpoints like `/register`, `/peers`, and `/health`.

2. **Start a Coordinator Node** (submits tasks and manages them):
   ```bash
   ./bin/peer --name=coordinator --port=9001 --coordinator --bootstrap=http://localhost:8080 --auto-demo
   ```
   - `--auto-demo`: Automatically runs demo tasks (e.g., prime verification) for testing.
   - Connects to the bootstrap server for peer discovery.

3. **Start Worker Nodes** (process tasks):
   ```bash
   ./bin/peer --name=worker-1 --port=9002 --worker --bootstrap=http://localhost:8080
   ./bin/peer --name=worker-2 --port=9003 --worker --bootstrap=http://localhost:8080
   ./bin/peer --name=worker-3 --port=9004 --worker --bootstrap=http://localhost:8080
   ./bin/peer --name=worker-4 --port=9005 --worker --bootstrap=http://localhost:8080
   ```
   - Workers connect to the bootstrap server and register themselves.
   - Tasks are assigned via round-robin from the coordinator.

**Notes**:
- Use different ports for each peer to avoid conflicts.
- The coordinator will automatically discover and assign tasks to workers.
- All nodes exchange names and connect via the bootstrap server.
- For manual connections, use `--peer=<multiaddr>` instead of `--bootstrap`.

## Key Features

### Peer Roles
- **Worker**: Processes computational tasks (prime, factorial, fibonacci, sum)
- **Coordinator**: Submits and manages tasks
- **Hybrid**: Can act as both worker and coordinator

### Task Details
The system supports distributed computation of simple mathematical tasks. Tasks are submitted by coordinators and processed by workers using round-robin assignment. Results are returned asynchronously.

#### Supported Task Types
- **Prime Checking** (`prime <number>`): Determines if a number is prime.
  - Input: Single integer (e.g., 97).
  - Output: Boolean (true/false).
  - Limitations: Efficient for numbers up to ~10^12; uses trial division.
  - Example: `/task prime 97` → "97 is prime".

- **Factorial Calculation** (`factorial <number>`): Computes n!.
  - Input: Single integer (e.g., 5).
  - Output: Integer result.
  - Limitations: Max n=20 to prevent overflow; returns error for n>20 or negative.
  - Example: `/task factorial 5` → "factorial(5) = 120".

- **Fibonacci Sequence** (`fibonacci <number>`): Computes the nth Fibonacci number.
  - Input: Single integer (e.g., 10).
  - Output: Integer result.
  - Limitations: Max n=50 to prevent long computation; iterative calculation.
  - Example: `/task fibonacci 10` → "fibonacci(10) = 55".

- **Sum Calculation** (`sum <numbers...>`): Sums a list of numbers.
  - Input: Multiple integers (e.g., 1 2 3 4).
  - Output: Integer sum.
  - Limitations: No upper limit on input size; simple addition.
  - Example: `/task sum 1 2 3 4` → "sum([1,2,3,4]) = 10".

#### Task Behavior
- **Assignment**: Round-robin to available workers; priority-based queuing (higher priority first).
- **Status Tracking**: Tasks have statuses (pending, assigned, processing, completed, failed).
- **Results**: Returned with duration, worker ID, and success/error details.
- **Statistics**: Tracks total submitted/completed/failed tasks per worker.
- **Error Handling**: Invalid inputs or computation errors are reported.

### Protocols
- `/chat/1.0.0` - Chat messaging
- `/name/1.0.0` - Name exchange
- `/task/1.0.0` - Task distribution

### CLI Commands
The interactive CLI supports chat, task submission, and system management. Commands start with `/` for system actions; chat messages use `<name|peerID> <message>`.

#### Chat Commands
- `<name|peerID> <message>`: Send a message to a peer.
  - Example: `worker-1 Hello!` (sends "Hello!" to the peer named "worker-1").
  - If name not found, tries to decode as a peer ID.
  - Confirmation shown on send.

#### Task Commands
- `/task prime <number>`: Submit a prime check task.
  - Example: `/task prime 97`.
- `/task factorial <number>`: Submit a factorial task.
  - Example: `/task factorial 5`.
- `/task fibonacci <number>`: Submit a Fibonacci task.
  - Example: `/task fibonacci 10`.
- `/task sum <numbers...>`: Submit a sum task.
  - Example: `/task sum 1 2 3 4`.
- **Notes**: Tasks are assigned to workers automatically. Use `/stats` to monitor progress.

#### System Commands
- `/peers`: List all connected peers with names and IDs.
- `/connect <multiaddr>`: Manually connect to a peer using its multiaddr.
  - Example: `/connect /ip4/127.0.0.1/tcp/9002/p2p/12D3KooWAbc...`.
- `/stats`: Show task distribution statistics (total submitted/completed/failed, per-worker breakdown).
- `/status [taskID]`: Show status of a specific task or all tasks.
  - Example: `/status task_123_prime` or `/status` (shows all).
- `/queue`: Show the current task queue (length and queued tasks with priorities).
- `/help`: Display all available commands.
- `/quit` or `/exit`: Exit the application.

**CLI Usage Tips**:
- Commands are case-sensitive.
- Task results are logged automatically when completed.
- Use `/peers` to find peer names for chat.
- The CLI runs in a loop; type commands and press Enter.

## Current State
The system includes:
- Working P2P networking with automatic discovery
- Task distribution with round-robin assignment
- Real-time statistics and monitoring
- Auto-demo mode for testing
- Priority-based task queuing
- Comprehensive CLI interface

## Technologies
- Go with libp2p networking stack
- JSON for message serialization
- HTTP REST API for bootstrap server
- Noise protocol for security
- Mplex for stream multiplexing