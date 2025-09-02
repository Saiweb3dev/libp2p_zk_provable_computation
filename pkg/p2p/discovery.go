package p2p

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"
)

// BootstrapClient handles communication with bootstrap server
type BootstrapClient struct {
    bootstrapURL string
    host         host.Host
    peerName     string
    role         string
}

// PeerInfo represents peer information for bootstrap
type PeerInfo struct {
    ID        string   `json:"id"`
    Name      string   `json:"name"`
    Addresses []string `json:"addresses"`
    Role      string   `json:"role"`
}

// NewBootstrapClient creates a new bootstrap client
func NewBootstrapClient(bootstrapURL string, h host.Host, peerName, role string) *BootstrapClient {
    return &BootstrapClient{
        bootstrapURL: bootstrapURL,
        host:         h,
        peerName:     peerName,
        role:         role,
    }
}

// RegisterWithBootstrap registers this peer with the bootstrap server
func (bc *BootstrapClient) RegisterWithBootstrap() error {
    // Build multiaddresses for this host
    var addresses []string
    hostID := fmt.Sprintf("/p2p/%s", bc.host.ID().String())
    
    for _, addr := range bc.host.Addrs() {
        fullAddr := addr.String() + hostID
        addresses = append(addresses, fullAddr)
    }

    peerInfo := PeerInfo{
        ID:        bc.host.ID().String(),
        Name:      bc.peerName,
        Addresses: addresses,
        Role:      bc.role,
    }

    jsonData, err := json.Marshal(peerInfo)
    if err != nil {
        return fmt.Errorf("failed to marshal peer info: %v", err)
    }

    resp, err := http.Post(
        fmt.Sprintf("%s/register", bc.bootstrapURL),
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return fmt.Errorf("failed to register with bootstrap: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("bootstrap registration failed with status: %d", resp.StatusCode)
    }

    log.Printf("✅ Registered with bootstrap server")
    return nil
}

// ConnectToKnownPeers discovers and connects to peers from bootstrap
func (bc *BootstrapClient) ConnectToKnownPeers() error {
    resp, err := http.Get(fmt.Sprintf("%s/peers", bc.bootstrapURL))
    if err != nil {
        return fmt.Errorf("failed to get peers from bootstrap: %v", err)
    }
    defer resp.Body.Close()

    var peers []PeerInfo
    if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
        return fmt.Errorf("failed to decode peers response: %v", err)
    }

    connectedCount := 0
    for _, peerInfo := range peers {
        // Skip ourselves
        if peerInfo.ID == bc.host.ID().String() {
            continue
        }

        // Try to connect to this peer
        if err := bc.connectToPeer(peerInfo); err != nil {
            log.Printf("⚠️  Failed to connect to %s: %v", peerInfo.Name, err)
        } else {
            connectedCount++
            log.Printf("🔗 Connected to %s (%s)", peerInfo.Name, peerInfo.Role)
        }
    }

    log.Printf("📊 Connected to %d peers", connectedCount)
    return nil
}

// connectToPeer connects to a specific peer using their addresses
func (bc *BootstrapClient) connectToPeer(peerInfo PeerInfo) error {
    _, err := peer.Decode(peerInfo.ID)
    if err != nil {
        return fmt.Errorf("invalid peer ID: %v", err)
    }

    // Try each address until one works
    for _, addrStr := range peerInfo.Addresses {
        maddr, err := ma.NewMultiaddr(addrStr)
        if err != nil {
            continue
        }

        addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
        if err != nil {
            continue
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        
        if err := bc.host.Connect(ctx, *addrInfo); err == nil {
            return nil // Successfully connected
        }
    }

    return fmt.Errorf("failed to connect using any address")
}

// StartPeriodicRegistration keeps the peer registered with bootstrap
func (bc *BootstrapClient) StartPeriodicRegistration() {
    ticker := time.NewTicker(2 * time.Minute)
    go func() {
        for range ticker.C {
            if err := bc.RegisterWithBootstrap(); err != nil {
                log.Printf("⚠️  Failed to re-register with bootstrap: %v", err)
            }
        }
    }()
}