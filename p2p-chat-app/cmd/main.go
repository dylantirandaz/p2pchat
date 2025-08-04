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

	go netw.Listen(":9000", cht)

	fmt.Print("Enter peer address to connect (or leave blank): ")
	addr, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}
	addr = addr[:len(addr)-1] 

	if addr != "" {
		err := netw.Connect(addr)
		if err != nil {
			fmt.Println("Failed to connect:", err)
		}
	}

	cht.StartChat(netw)
}
