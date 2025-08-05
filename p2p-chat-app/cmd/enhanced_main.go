package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"p2p-chat-app/internal/chat"
	"p2p-chat-app/internal/identity"
	"p2p-chat-app/internal/network"
)

func main() {
	fmt.Println("🚀 Starting Enhanced P2P Chat Application...")
	fmt.Println("=====================================")

	userIdentity, err := getOrCreateIdentity()
	if err != nil {
		log.Fatalf("Failed to get user identity: %v", err)
	}

	fmt.Printf("👤 Welcome, %s (ID: %s)\n", userIdentity.Username, userIdentity.ID)

	dataDir := filepath.Join(os.Getenv("HOME"), ".p2pchat", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	chatSystem, err := chat.NewEnhancedChat(userIdentity, dataDir)
	if err != nil {
		log.Fatalf("Failed to create chat system: %v", err)
	}

	networkSystem := network.NewEnhancedP2PNetwork(userIdentity)
	networkSystem.SetChat(chatSystem)

	if err := networkSystem.Start(); err != nil {
		log.Fatalf("Failed to start network: %v", err)
	}

	mode := getUserNetworkMode()

	switch mode {
	case "listen":
		if err := networkSystem.Listen(":9000"); err != nil {
			log.Fatalf("Failed to start listening: %v", err)
		}
		fmt.Println("🎧 Listening for connections on :9000")

	case "connect":
		address := getConnectionAddress()
		if err := networkSystem.Connect(address); err != nil {
			log.Printf("Failed to connect to %s: %v", address, err)
		} else {
			fmt.Printf("🔗 Connected to %s\n", address)
		}

	case "discover":
		handlePeerDiscovery(networkSystem)

	case "auto":
		if err := networkSystem.Listen(":9000"); err != nil {
			log.Printf("Failed to start listening: %v", err)
		} else {
			fmt.Println("🎧 Listening for connections on :9000")
		}
		
		go autoConnectToPeers(networkSystem)
	}

	chatSystem.Start()

	showMainMenu(networkSystem)

	select {}
}

func getOrCreateIdentity() (*identity.Identity, error) {
	configDir := filepath.Join(os.Getenv("HOME"), ".p2pchat")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	identityFile := filepath.Join(configDir, "identity.txt")

	if _, err := os.Stat(identityFile); os.IsNotExist(err) {
		fmt.Print("Enter your username: ")
		reader := bufio.NewReader(os.Stdin)
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		if username == "" {
			username = "Anonymous"
		}

		userIdentity, err := identity.NewIdentity(username)
		if err != nil {
			return nil, err
		}

		privKey, _ := userIdentity.ExportPrivateKey()
		if err := os.WriteFile(identityFile, []byte(username+"\n"+privKey), 0600); err != nil {
			return nil, err
		}

		fmt.Println("✅ New identity created and saved")
		return userIdentity, nil
	}

	data, err := os.ReadFile(identityFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid identity file format")
	}

	username := lines[0]
	privKeyPEM := strings.Join(lines[1:], "\n")

	privKey, err := identity.ImportPrivateKey(privKeyPEM)
	if err != nil {
		return nil, err
	}

	userIdentity := &identity.Identity{
		Username:   username,
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
	}

	pubKey, _ := userIdentity.ExportPublicKey()
	userIdentity.ID = generateUserID(pubKey)

	fmt.Println("✅ Existing identity loaded")
	return userIdentity, nil
}

func getUserNetworkMode() string {
	fmt.Println("\n🌐 Network Mode:")
	fmt.Println("1. Listen for connections")
	fmt.Println("2. Connect to specific peer")
	fmt.Println("3. Discover and connect to peers")
	fmt.Println("4. Auto mode (listen + discover)")
	fmt.Print("Choose mode (1-4, default: 4): ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return "listen"
	case "2":
		return "connect"
	case "3":
		return "discover"
	default:
		return "auto"
	}
}

func getConnectionAddress() string {
	fmt.Print("Enter peer address (host:port): ")
	reader := bufio.NewReader(os.Stdin)
	address, _ := reader.ReadString('\n')
	return strings.TrimSpace(address)
}

func handlePeerDiscovery(network *network.EnhancedP2PNetwork) {
	fmt.Println("🔍 Discovering peers...")
	
	time.Sleep(2 * time.Second)
	
	peers := network.GetDiscoveredPeers()
	if len(peers) == 0 {
		fmt.Println("No peers discovered. Starting in listen mode...")
		network.Listen(":9000")
		return
	}

	fmt.Println("📡 Discovered peers:")
	for i, peer := range peers {
		fmt.Printf("%d. %s (%s) - %s\n", i+1, peer.User.Username, peer.User.ID, peer.Address)
	}

	fmt.Print("Enter peer number to connect (or 0 to listen): ")
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "0" || choice == "" {
		network.Listen(":9000")
		return
	}

	var peerIndex int
	if _, err := fmt.Sscanf(choice, "%d", &peerIndex); err == nil && peerIndex > 0 && peerIndex <= len(peers) {
		peer := peers[peerIndex-1]
		if err := network.ConnectToPeer(peer); err != nil {
			fmt.Printf("Failed to connect to %s: %v\n", peer.User.Username, err)
		} else {
			fmt.Printf("🔗 Connected to %s\n", peer.User.Username)
		}
	}

	network.Listen(":9000")
}

func autoConnectToPeers(network *network.EnhancedP2PNetwork) {
	time.Sleep(3 * time.Second) 

	peers := network.GetDiscoveredPeers()
	for _, peer := range peers {
		if err := network.ConnectToPeer(peer); err != nil {
			fmt.Printf("Auto-connect to %s failed: %v\n", peer.User.Username, err)
		} else {
			fmt.Printf("🔗 Auto-connected to %s\n", peer.User.Username)
		}
	}
}

func showMainMenu(network *network.EnhancedP2PNetwork) {
	fmt.Println("\n🎯 Enhanced P2P Chat is ready!")
	fmt.Println("💡 Tips:")
	fmt.Println("   - Type messages to chat in the current room")
	fmt.Println("   - Use /help to see all available commands")
	fmt.Println("   - Use /rooms to see available chat rooms")
	fmt.Println("   - Use /users to see who's online")
	fmt.Println("   - Use /private <user> <message> for private messages")
	fmt.Println("")
}

func generateUserID(pubKey string) string {
	hash := sha256.Sum256([]byte(pubKey))
	return hex.EncodeToString(hash[:])[:16]
}
