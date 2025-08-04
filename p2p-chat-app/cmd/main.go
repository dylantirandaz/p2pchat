package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"p2p-chat-app/internal/chat"
	"p2p-chat-app/internal/network"
)

func main() {
	fmt.Println("Starting P2P Chat Application...")

	netw := network.NewP2PNetwork()
	cht := chat.NewChat()

	fmt.Print("Enter 'listen' or peer address to connect (or leave blank to listen): ")
	action, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}
	action = action[:len(action)-1]

	if action == "listen" || action == "" {
		go netw.Listen(":9000", cht)
		fmt.Println("Listening for connections on :9000...")
	} else {
		err := netw.Connect(action)
		if err != nil {
			fmt.Println("Failed to connect:", err)
			os.Exit(1)
		}
		fmt.Println("Connected to:", action)
		netw.GetMu().Lock()
		peer := netw.Peers[action]
		netw.GetMu().Unlock()
		cht.AddPeer(action, peer.Conn)
	}

	cht.StartChat(netw)
}
