package p2p

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "log"
    "sync"
	"time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/protocol"
)

const (
    // Protocol IDs for different communication types
    ChatProtocolID = "/chat/1.0.0"
    NameProtocolID = "/name/1.0.0"
)

// ChatService handles all chat-related functionality
type ChatService struct {
    host        host.Host
    peerNames   map[peer.ID]string // Maps peer IDs to their display names
    peerNamesMu sync.RWMutex       // Protects peerNames map
    myName      string             // This peer's display name
}

// NewChatService creates a new chat service instance
func NewChatService(h host.Host, myName string) *ChatService {
    cs := &ChatService{
        host:      h,
        peerNames: make(map[peer.ID]string),
        myName:    myName,
    }
    
    // Store our own name in the map
    cs.peerNames[h.ID()] = myName
    
    // Register protocol handlers
    h.SetStreamHandler(ChatProtocolID, cs.handleChatStream)
    h.SetStreamHandler(NameProtocolID, cs.handleNameStream)
    
    return cs
}

// handleChatStream processes incoming chat messages
func (cs *ChatService) handleChatStream(s network.Stream) {
    defer s.Close()
    
    peerID := s.Conn().RemotePeer()
    log.Printf("📩 New chat stream opened by: %s", peerID)
    
    reader := bufio.NewReader(s)
    
    // Read messages until stream closes or error occurs
    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                // Stream closed normally, no need to log as error
                log.Printf("📪 Chat stream closed by: %s", peerID)
            } else {
                log.Printf("❌ Error reading chat message: %v", err)
            }
            return
        }
        
        if message != "" {
            // Get the sender's name for display
            senderName := cs.GetPeerName(peerID)
            // Remove the extra newline and format nicely
            message = trimNewline(message)
            fmt.Printf("\n[From %s]: %s\n>> ", senderName, message)
        }
    }
}

// handleNameStream processes name exchange between peers
func (cs *ChatService) handleNameStream(s network.Stream) {
    defer s.Close()
    
    peerID := s.Conn().RemotePeer()
    reader := bufio.NewReader(s)
    
    // Read the incoming name
    name, err := reader.ReadString('\n')
    if err != nil {
        if err != io.EOF {
            log.Printf("❌ Error reading name from %s: %v", peerID, err)
        }
        return
    }
    
    // Clean up the received name and store it
    name = trimNewline(name)
    if name != "" {
        cs.setPeerName(peerID, name)
        log.Printf("✅ Registered peer %s as '%s'", peerID, name)
        
        // Send our name back
        _, err = s.Write([]byte(cs.myName + "\n"))
        if err != nil {
            log.Printf("❌ Error sending name to %s: %v", peerID, err)
            return
        }
        log.Printf("📤 Sent my name '%s' to %s", cs.myName, peerID)
    }
}

// SendName initiates name exchange with a peer
func (cs *ChatService) SendName(peerID peer.ID) error {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
    defer cancel()
    
    s, err := cs.host.NewStream(ctx, peerID, protocol.ID(NameProtocolID))
    if err != nil {
        return fmt.Errorf("error opening name stream: %v", err)
    }
    defer s.Close()
    
    // Send our name first
    _, err = s.Write([]byte(cs.myName + "\n"))
    if err != nil {
        return fmt.Errorf("error sending name: %v", err)
    }
    log.Printf("📤 Sent my name '%s' to %s", cs.myName, peerID)
    
    // Wait for their name response
    reader := bufio.NewReader(s)
    theirName, err := reader.ReadString('\n')
    if err != nil && err != io.EOF {
        return fmt.Errorf("error reading their name: %v", err)
    }
    
    theirName = trimNewline(theirName)
    if theirName != "" {
        cs.setPeerName(peerID, theirName)
        log.Printf("✅ Registered peer %s as '%s'", peerID, theirName)
    }
    
    return nil
}

// SendMessage sends a chat message to a specific peer
func (cs *ChatService) SendMessage(targetID peer.ID, message string) error {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
    defer cancel()
    
    s, err := cs.host.NewStream(ctx, targetID, ChatProtocolID)
    if err != nil {
        return fmt.Errorf("error opening chat stream: %v", err)
    }
    defer s.Close()
    
    // Format and send the message
    formattedMessage := fmt.Sprintf("[%s]: %s\n", cs.myName, message)
    _, err = s.Write([]byte(formattedMessage))
    if err != nil {
        return fmt.Errorf("error writing message: %v", err)
    }
    
    return nil
}

// GetPeerName returns the display name for a peer ID
func (cs *ChatService) GetPeerName(peerID peer.ID) string {
    cs.peerNamesMu.RLock()
    defer cs.peerNamesMu.RUnlock()
    
    if name, exists := cs.peerNames[peerID]; exists {
        return name
    }
    // Fallback to peer ID string if name not found
    return peerID.String()
}

// FindPeerByName finds a peer ID by their display name
func (cs *ChatService) FindPeerByName(name string) (peer.ID, bool) {
    cs.peerNamesMu.RLock()
    defer cs.peerNamesMu.RUnlock()
    
    for peerID, peerName := range cs.peerNames {
        if peerName == name {
            return peerID, true
        }
    }
    return "", false
}

// GetAllPeers returns a copy of all known peers
func (cs *ChatService) GetAllPeers() map[peer.ID]string {
    cs.peerNamesMu.RLock()
    defer cs.peerNamesMu.RUnlock()
    
    peers := make(map[peer.ID]string)
    for id, name := range cs.peerNames {
        peers[id] = name
    }
    return peers
}

// setPeerName safely sets a peer's name
func (cs *ChatService) setPeerName(peerID peer.ID, name string) {
    cs.peerNamesMu.Lock()
    defer cs.peerNamesMu.Unlock()
    cs.peerNames[peerID] = name
}

// trimNewline removes trailing newline characters
func trimNewline(s string) string {
    if len(s) > 0 && s[len(s)-1] == '\n' {
        s = s[:len(s)-1]
    }
    if len(s) > 0 && s[len(s)-1] == '\r' {
        s = s[:len(s)-1]
    }
    return s
}