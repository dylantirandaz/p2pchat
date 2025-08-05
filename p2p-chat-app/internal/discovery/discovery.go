package discovery

import (
	"encoding/json"
	"fmt"
	"net"
	"p2p-chat-app/internal/protocol"
	"sync"
	"time"
)

type PeerInfo struct {
	Address   string    `json:"address"`
	User      protocol.User `json:"user"`
	LastSeen  time.Time `json:"last_seen"`
	Online    bool      `json:"online"`
}

type DiscoveryService struct {
	port      int
	peers     map[string]*PeerInfo
	mu        sync.RWMutex
	localUser protocol.User
	conn      *net.UDPConn
	running   bool
}

type DiscoveryMessage struct {
	Type      string        `json:"type"` 
	User      protocol.User `json:"user"`
	Timestamp time.Time     `json:"timestamp"`
}

func NewDiscoveryService(port int, localUser protocol.User) *DiscoveryService {
	return &DiscoveryService{
		port:      port,
		peers:     make(map[string]*PeerInfo),
		localUser: localUser,
	}
}

func (ds *DiscoveryService) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", ds.port))
	if err != nil {
		return err
	}

	ds.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	ds.running = true
	go ds.listenLoop()
	go ds.announceLoop()

	return nil
}

func (ds *DiscoveryService) Stop() {
	ds.running = false
	if ds.conn != nil {
		ds.conn.Close()
	}
}

func (ds *DiscoveryService) GetPeers() []*PeerInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var peers []*PeerInfo
	for _, peer := range ds.peers {
		if time.Since(peer.LastSeen) < 5*time.Minute {
			peer.Online = true
			peers = append(peers, peer)
		}
	}
	return peers
}

func (ds *DiscoveryService) listenLoop() {
	buffer := make([]byte, 1024)

	for ds.running {
		ds.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := ds.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if ds.running {
				fmt.Printf("Discovery listen error: %v\n", err)
			}
			continue
		}

		var msg DiscoveryMessage
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}

		if msg.User.ID == ds.localUser.ID {
			continue
		}

		ds.handleDiscoveryMessage(&msg, addr)
	}
}

func (ds *DiscoveryService) announceLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ds.announce()

	for ds.running {
		select {
		case <-ticker.C:
			ds.announce()
		}
	}
}

func (ds *DiscoveryService) announce() {
	msg := DiscoveryMessage{
		Type:      "announce",
		User:      ds.localUser,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", ds.port))
	if err != nil {
		return
	}

	if ds.conn != nil {
		ds.conn.WriteTo(data, broadcastAddr)
	}
}

func (ds *DiscoveryService) handleDiscoveryMessage(msg *DiscoveryMessage, addr *net.UDPAddr) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	peerAddr := addr.IP.String()
	peer := &PeerInfo{
		Address:  peerAddr,
		User:     msg.User,
		LastSeen: time.Now(),
		Online:   true,
	}

	ds.peers[msg.User.ID] = peer

	if msg.Type == "announce" {
		go ds.sendResponse(addr)
	}
}

func (ds *DiscoveryService) sendResponse(addr *net.UDPAddr) {
	msg := DiscoveryMessage{
		Type:      "response",
		User:      ds.localUser,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	if ds.conn != nil {
		ds.conn.WriteTo(data, addr)
	}
}

func (ds *DiscoveryService) FindPeer(userID string) *PeerInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	peer, exists := ds.peers[userID]
	if !exists {
		return nil
	}

	if time.Since(peer.LastSeen) > 5*time.Minute {
		peer.Online = false
	}

	return peer
}
