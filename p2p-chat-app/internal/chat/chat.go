package chat

import (
	"fmt"
	"net"
	"sync"
)

type Chat struct {
	peers map[string]net.Conn
	mu    sync.Mutex
}

func NewChat() *Chat {
	return &Chat{
		peers: make(map[string]net.Conn),
	}
}

func (c *Chat) AddPeer(address string, conn net.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.peers[address] = conn
}

func (c *Chat) RemovePeer(address string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.peers, address)
}

func (c *Chat) SendMessage(address string, message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, exists := c.peers[address]
	if !exists {
		return fmt.Errorf("peer not found: %s", address)
	}
	_, err := conn.Write([]byte(message))
	return err
}

func (c *Chat) StartChat() {
	
	}