package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"p2p-chat-app/internal/blockchain"
	"p2p-chat-app/internal/chat"
	"p2p-chat-app/internal/discovery"
	"p2p-chat-app/internal/identity"
	"p2p-chat-app/internal/protocol"
	"sync"
	"time"
)

type EnhancedP2PNetwork struct {
	identity    *identity.Identity
	peers       map[string]*EnhancedPeer
	mu          sync.RWMutex
	chat        *chat.EnhancedChat
	discovery   *discovery.DiscoveryService
	blockchain  *blockchain.Blockchain
	listener    net.Listener
	running     bool
}

type EnhancedPeer struct {
	Conn     net.Conn
	User     protocol.User
	LastSeen time.Time
	Verified bool
}

func NewEnhancedP2PNetwork(userIdentity *identity.Identity) *EnhancedP2PNetwork {
	pubKey, _ := userIdentity.ExportPublicKey()
	user := protocol.User{
		ID:        userIdentity.ID,
		Username:  userIdentity.Username,
		PublicKey: pubKey,
		Online:    true,
	}

	discovery := discovery.NewDiscoveryService(9001, user)

	return &EnhancedP2PNetwork{
		identity:  userIdentity,
		peers:     make(map[string]*EnhancedPeer),
		discovery: discovery,
	}
}

func (n *EnhancedP2PNetwork) SetChat(chat *chat.EnhancedChat) {
	n.chat = chat
}

func (n *EnhancedP2PNetwork) SetBlockchain(bc *blockchain.Blockchain) {
	n.blockchain = bc
}

func (n *EnhancedP2PNetwork) Start() error {
	n.running = true
	
	if err := n.discovery.Start(); err != nil {
		return fmt.Errorf("failed to start discovery: %v", err)
	}

	fmt.Println("🔍 Discovery service started")
	return nil
}

func (n *EnhancedP2PNetwork) Stop() {
	n.running = false
	
	if n.discovery != nil {
		n.discovery.Stop()
	}
	
	if n.listener != nil {
		n.listener.Close()
	}
	
	n.mu.Lock()
	for _, peer := range n.peers {
		peer.Conn.Close()
	}
	n.mu.Unlock()
}

func (n *EnhancedP2PNetwork) Listen(addr string) error {
	var err error
	n.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	fmt.Printf("🎧 Listening for connections on %s\n", addr)

	go func() {
		for n.running {
			conn, err := n.listener.Accept()
			if err != nil {
				if n.running {
					fmt.Printf("Accept error: %v\n", err)
				}
				continue
			}

			go n.handleConnection(conn)
		}
	}()

	return nil
}

func (n *EnhancedP2PNetwork) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	return n.handleConnection(conn)
}

func (n *EnhancedP2PNetwork) ConnectToPeer(peerInfo *discovery.PeerInfo) error {
	addr := fmt.Sprintf("%s:9000", peerInfo.Address) 
	return n.Connect(addr)
}

func (n *EnhancedP2PNetwork) GetDiscoveredPeers() []*discovery.PeerInfo {
	return n.discovery.GetPeers()
}

func (n *EnhancedP2PNetwork) ListPeers() {
	n.mu.RLock()
	defer n.mu.RUnlock()

	fmt.Println("🔗 Connected peers:")
	for userID, peer := range n.peers {
		status := "✅"
		if time.Since(peer.LastSeen) > 5*time.Minute {
			status = "⚠️"
		}
		fmt.Printf("  %s %s (%s)\n", status, peer.User.Username, userID)
	}

	discovered := n.discovery.GetPeers()
	if len(discovered) > 0 {
		fmt.Println("📡 Discovered peers:")
		for _, peer := range discovered {
			if _, connected := n.peers[peer.User.ID]; !connected {
				fmt.Printf("  🔍 %s (%s) - %s\n", peer.User.Username, peer.User.ID, peer.Address)
			}
		}
	}
}

func (n *EnhancedP2PNetwork) handleConnection(conn net.Conn) error {
	peer, err := n.performHandshake(conn)
	if err != nil {
		conn.Close()
		return err
	}

	n.mu.Lock()
	n.peers[peer.User.ID] = peer
	n.mu.Unlock()

	if n.chat != nil {
		n.chat.AddPeer(peer.User.ID, conn)
	}

	fmt.Printf("🤝 Connected to %s (%s)\n", peer.User.Username, peer.User.ID)

	go n.handlePeerMessages(peer)

	return nil
}

func (n *EnhancedP2PNetwork) performHandshake(conn net.Conn) (*EnhancedPeer, error) {
	pubKey, _ := n.identity.ExportPublicKey()
	ourUser := protocol.User{
		ID:        n.identity.ID,
		Username:  n.identity.Username,
		PublicKey: pubKey,
		Online:    true,
	}

	handshake := protocol.HandshakeData{
		User:      ourUser,
		Version:   "1.0",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(handshake)
	if err != nil {
		return nil, err
	}

	if _, err := conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	var theirHandshake protocol.HandshakeData
	if err := json.Unmarshal([]byte(response), &theirHandshake); err != nil {
		return nil, err
	}



	peer := &EnhancedPeer{
		Conn:     conn,
		User:     theirHandshake.User,
		LastSeen: time.Now(),
		Verified: true, 
	}

	return peer, nil
}

func (n *EnhancedP2PNetwork) handlePeerMessages(peer *EnhancedPeer) {
	reader := bufio.NewReader(peer.Conn)
	
	for n.running {
		peer.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		peer.LastSeen = time.Now()

		if n.chat != nil {
			n.chat.ProcessIncomingMessage(message)
		}
	}

	n.mu.Lock()
	delete(n.peers, peer.User.ID)
	n.mu.Unlock()

	if n.chat != nil {
		n.chat.RemovePeer(peer.User.ID)
	}

	peer.Conn.Close()
	fmt.Printf("🔌 Disconnected from %s\n", peer.User.Username)
}

func (n *EnhancedP2PNetwork) Broadcast(message []byte) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, peer := range n.peers {
		_, err := peer.Conn.Write(append(message, '\n'))
		if err != nil {
			fmt.Printf("Error sending to %s: %v\n", peer.User.Username, err)
		}
	}
}

func (n *EnhancedP2PNetwork) SendToPeer(userID string, message []byte) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peer, exists := n.peers[userID]
	if !exists {
		return fmt.Errorf("peer %s not connected", userID)
	}

	_, err := peer.Conn.Write(append(message, '\n'))
	return err
}

func (n *EnhancedP2PNetwork) GetConnectedPeers() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var peers []string
	for userID := range n.peers {
		peers = append(peers, userID)
	}
	return peers
}
