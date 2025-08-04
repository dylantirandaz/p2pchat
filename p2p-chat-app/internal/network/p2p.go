package network

import (
    "net"
    "sync"
)

type Peer struct {
    Conn net.Conn
    Addr string
}

type P2PNetwork struct {
    peers map[string]*Peer
    mu    sync.Mutex
}

func NewP2PNetwork() *P2PNetwork {
    return &P2PNetwork{
        peers: make(map[string]*Peer),
    }
}

func (n *P2PNetwork) Connect(addr string) error {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return err
    }
    n.mu.Lock()
    n.peers[addr] = &Peer{Conn: conn, Addr: addr}
    n.mu.Unlock()
    return nil
}

func (n *P2PNetwork) Broadcast(message []byte) {
    n.mu.Lock()
    defer n.mu.Unlock()
    for _, peer := range n.peers {
        _, err := peer.Conn.Write(message)
        if err != nil {
            peer.Conn.Close()
            delete(n.peers, peer.Addr)
        }
    }
}