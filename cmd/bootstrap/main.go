package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"
)

// PeerInfo represents a registered peer
type PeerInfo struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Addresses []string  `json:"addresses"`
    Role      string    `json:"role"`
    LastSeen  time.Time `json:"last_seen"`
}

// BootstrapServer manages peer discovery
type BootstrapServer struct {
    peers   map[string]*PeerInfo
    peersMu sync.RWMutex
    port    int
}

// NewBootstrapServer creates a new bootstrap server
func NewBootstrapServer(port int) *BootstrapServer {
    return &BootstrapServer{
        peers: make(map[string]*PeerInfo),
        port:  port,
    }
}

// RegisterPeer handles peer registration
func (bs *BootstrapServer) RegisterPeer(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var peerInfo PeerInfo
    if err := json.NewDecoder(r.Body).Decode(&peerInfo); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    peerInfo.LastSeen = time.Now()

    bs.peersMu.Lock()
    bs.peers[peerInfo.ID] = &peerInfo
    bs.peersMu.Unlock()

    log.Printf("📝 Registered peer: %s (%s) - %s", peerInfo.Name, peerInfo.Role, peerInfo.ID)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

// GetPeers returns all registered peers
func (bs *BootstrapServer) GetPeers(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    bs.peersMu.RLock()
    peers := make([]*PeerInfo, 0, len(bs.peers))
    for _, peer := range bs.peers {
        // Only return peers seen in the last 5 minutes
        if time.Since(peer.LastSeen) < 5*time.Minute {
            peers = append(peers, peer)
        }
    }
    bs.peersMu.RUnlock()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(peers)
}

// GetPeersByRole returns peers filtered by role
func (bs *BootstrapServer) GetPeersByRole(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    role := r.URL.Query().Get("role")
    if role == "" {
        http.Error(w, "Role parameter required", http.StatusBadRequest)
        return
    }

    bs.peersMu.RLock()
    var filteredPeers []*PeerInfo
    for _, peer := range bs.peers {
        if peer.Role == role && time.Since(peer.LastSeen) < 5*time.Minute {
            filteredPeers = append(filteredPeers, peer)
        }
    }
    bs.peersMu.RUnlock()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(filteredPeers)
}

// CleanupOldPeers removes inactive peers
func (bs *BootstrapServer) CleanupOldPeers() {
    ticker := time.NewTicker(1 * time.Minute)
    go func() {
        for range ticker.C {
            bs.peersMu.Lock()
            for id, peer := range bs.peers {
                if time.Since(peer.LastSeen) > 5*time.Minute {
                    log.Printf("🧹 Removing inactive peer: %s", peer.Name)
                    delete(bs.peers, id)
                }
            }
            bs.peersMu.Unlock()
        }
    }()
}

func main() {
    port := flag.Int("port", 8080, "HTTP server port for bootstrap")
    flag.Parse()

    bs := NewBootstrapServer(*port)
    
    // Start cleanup routine
    bs.CleanupOldPeers()

    // Setup HTTP routes
    http.HandleFunc("/register", bs.RegisterPeer)
    http.HandleFunc("/peers", bs.GetPeers)
    http.HandleFunc("/peers/role", bs.GetPeersByRole)

    // Health check endpoint
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    log.Printf("🚀 Bootstrap server starting on port %d", *port)
    log.Printf("📍 Endpoints:")
    log.Printf("   POST /register - Register a peer")
    log.Printf("   GET  /peers    - Get all peers")
    log.Printf("   GET  /peers/role?role=<role> - Get peers by role")
    log.Printf("   GET  /health   - Health check")

    if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
        log.Fatalf("❌ Failed to start server: %v", err)
    }
}