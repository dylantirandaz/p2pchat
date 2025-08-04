package chat

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"p2p-chat-app/internal/encryption"
	"sync"
)

var key = []byte("example key 1234") 

type Chat struct {
	peers    map[string]net.Conn
	mu       sync.Mutex
	Incoming chan string
}

func NewChat() *Chat {
	return &Chat{
		peers:    make(map[string]net.Conn),
		Incoming: make(chan string, 10),
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

func (c *Chat) SendMessage(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	enc, err := encryption.Encrypt([]byte(message), key)
	if err != nil {
		fmt.Println("Encryption error:", err)
		return
	}
	for _, conn := range c.peers {
		fmt.Fprintln(conn, enc)
	}
}

func (c *Chat) StartChat(netw interface{}) {
	go func() {
		for msg := range c.Incoming {
			dec, err := encryption.Decrypt(msg[:len(msg)-1], key)
			if err == nil {
				fmt.Println("Peer:", string(dec))
			}
		}
	}()
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("You: ")
		text, _ := reader.ReadString('\n')
		c.SendMessage(text)
	}
}