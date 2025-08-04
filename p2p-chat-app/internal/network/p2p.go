package network

import (
	"bufio"
	"fmt"
	"net"
	"p2p-chat-app/internal/chat"
	"sync"
)

type Peer struct {
	Conn net.Conn
	Addr string
}

type P2PNetwork struct {
	peers map[string]*Peer
	mu    sync.Mutex
	Peers map[string]*Peer // Exported peers map
}

func NewP2PNetwork() *P2PNetwork {
	return &P2PNetwork{
		peers: make(map[string]*Peer),
		Peers: make(map[string]*Peer), // Initialize exported map
	}
}

func (n *P2PNetwork) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	n.mu.Lock()
	n.Peers[addr] = &Peer{Conn: conn, Addr: addr} // Use exported map
	n.mu.Unlock()

	fmt.Println("Connected to peer:", addr)
	return nil
}

func (n *P2PNetwork) Broadcast(message []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, peer := range n.Peers { 
		_, err := peer.Conn.Write(message)
		if err != nil {
			fmt.Println("Error writing to peer:", peer.Addr, err)
			peer.Conn.Close()
			delete(n.Peers, peer.Addr) 
		}
	}
}

func (n *P2PNetwork) Listen(addr string, cht *chat.Chat) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Listen error:", err)
		return
	}
	defer ln.Close()
	fmt.Println("Listening on", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		peerAddr := conn.RemoteAddr().String()
		fmt.Println("Accepted connection from:", peerAddr)

		n.mu.Lock()
		n.Peers[peerAddr] = &Peer{Conn: conn, Addr: peerAddr} 
		n.mu.Unlock()

		cht.AddPeer(peerAddr, conn)

		go func(c net.Conn, addr string) {
			reader := bufio.NewReader(c)
			for {
				msg, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Connection closed by:", addr, err)
					cht.RemovePeer(addr)
					c.Close()
					n.mu.Lock()
					delete(n.Peers, addr) 
					n.mu.Unlock()
					return
				}
				cht.Incoming <- msg
			}
		}(conn, peerAddr)
	}
}
