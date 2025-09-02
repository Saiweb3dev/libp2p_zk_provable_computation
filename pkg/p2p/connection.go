package p2p

import (
    "context"
    "fmt"
    "log"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"
)

// ConnectionManager handles peer connections and discovery
type ConnectionManager struct {
    host        host.Host
    chatService *ChatService
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(h host.Host, cs *ChatService) *ConnectionManager {
    return &ConnectionManager{
        host:        h,
        chatService: cs,
    }
}

// ConnectToPeer connects to a peer using their multiaddr string
func (cm *ConnectionManager) ConnectToPeer(peerAddr string) error {
    // Parse the multiaddr string
    maddr, err := ma.NewMultiaddr(peerAddr)
    if err != nil {
        return fmt.Errorf("invalid multiaddr: %v", err)
    }
    
    // Extract peer info from multiaddr
    info, err := peer.AddrInfoFromP2pAddr(maddr)
    if err != nil {
        return fmt.Errorf("error extracting peer info: %v", err)
    }
    
    // Attempt to connect
    if err := cm.host.Connect(context.Background(), *info); err != nil {
        return fmt.Errorf("connection failed: %v", err)
    }
    
    log.Printf("✅ Connected to peer %s", info.ID.String())
    
    // Immediately exchange names with the new peer
    if err := cm.chatService.SendName(info.ID); err != nil {
        log.Printf("Warning: Failed to exchange names with %s: %v", info.ID, err)
    }
    
    return nil
}

// GetConnectedPeers returns a list of currently connected peers
func (cm *ConnectionManager) GetConnectedPeers() []peer.ID {
    return cm.host.Network().Peers()
}

// PrintHostInfo displays this host's connection information
func (cm *ConnectionManager) PrintHostInfo() {
    hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", cm.host.ID().String()))
    
    fmt.Println("📍 Host Information:")
    fmt.Printf("   ID: %s\n", cm.host.ID().String())
    fmt.Println("   Listening on:")
    
    for _, addr := range cm.host.Addrs() {
        fullAddr := addr.Encapsulate(hostAddr)
        fmt.Printf("     %s\n", fullAddr)
    }
}

// GetHost returns the underlying libp2p host
func (cm *ConnectionManager) GetHost() host.Host {
    return cm.host
}

// GetHostID returns this host's peer ID
func (cm *ConnectionManager) GetHostID() peer.ID {
    return cm.host.ID()
}