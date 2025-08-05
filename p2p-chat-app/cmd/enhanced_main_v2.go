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

	"p2p-chat-app/internal/blockchain"
	"p2p-chat-app/internal/chat"
	"p2p-chat-app/internal/identity"
	"p2p-chat-app/internal/mobile"
	"p2p-chat-app/internal/network"
	"p2p-chat-app/internal/webui"
)

func main() {
	fmt.Println("🚀 starting enhanced p2p chat v2.0...")
	fmt.Println("=====================================")

	userIdentity, err := getOrCreateIdentity()
	if err != nil {
		log.Fatalf("failed to get user identity: %v", err)
	}

	fmt.Printf("👤 welcome, %s (id: %s)\n", userIdentity.Username, userIdentity.ID)

	// init blockchain for identity verification
	bc := blockchain.NewBlockchain()
	if err := bc.RegisterIdentity(userIdentity); err != nil {
		log.Printf("blockchain registration failed: %v", err)
	} else {
		fmt.Println("✅ identity registered on blockchain")
	}

	dataDir := filepath.Join(os.Getenv("HOME"), ".p2pchat", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	chatSystem, err := chat.NewEnhancedChat(userIdentity, dataDir)
	if err != nil {
		log.Fatalf("failed to create chat system: %v", err)
	}

	networkSystem := network.NewEnhancedP2PNetwork(userIdentity)
	networkSystem.SetChat(chatSystem)
	networkSystem.SetBlockchain(bc)

	if err := networkSystem.Start(); err != nil {
		log.Fatalf("failed to start network: %v", err)
	}

	// start web ui
	webServer := webui.NewWebServer("8080", chatSystem, networkSystem)
	go func() {
		if err := webServer.Start(); err != nil {
			log.Printf("web server failed: %v", err)
		}
	}()
	fmt.Println("🌐 web ui started on http://localhost:8080")

	// start mobile api
	mobileAPI := mobile.NewMobileAPI("8081", chatSystem, networkSystem)
	go func() {
		if err := mobileAPI.Start(); err != nil {
			log.Printf("mobile api failed: %v", err)
		}
	}()
	fmt.Println("📱 mobile api started on http://localhost:8081")

	mode := getUserMode()

	switch mode {
	case "terminal":
		runTerminalMode(chatSystem, networkSystem)
	case "web":
		runWebMode()
	case "headless":
		runHeadlessMode(networkSystem)
	default:
		runAutoMode(chatSystem, networkSystem)
	}
}

func getUserMode() string {
	fmt.Println("\n🎯 interface mode:")
	fmt.Println("1. terminal chat (classic)")
	fmt.Println("2. web ui only")
	fmt.Println("3. headless (api only)")
	fmt.Println("4. auto (all interfaces)")
	fmt.Print("choose mode (1-4, default: 4): ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return "terminal"
	case "2":
		return "web"
	case "3":
		return "headless"
	default:
		return "auto"
	}
}

func runTerminalMode(chat *chat.EnhancedChat, network *network.EnhancedP2PNetwork) {
	mode := getUserNetworkMode()
	setupNetwork(network, mode)
	chat.Start()
	showMainMenu(network)
	select {}
}

func runWebMode() {
	fmt.Println("🌐 web mode - open http://localhost:8080")
	fmt.Println("📱 mobile api available at http://localhost:8081")
	select {}
}

func runHeadlessMode(network *network.EnhancedP2PNetwork) {
	if err := network.Listen(":9000"); err != nil {
		log.Printf("failed to start listening: %v", err)
	}
	fmt.Println("🤖 headless mode - api only")
	select {}
}

func runAutoMode(chat *chat.EnhancedChat, network *network.EnhancedP2PNetwork) {
	if err := network.Listen(":9000"); err != nil {
		log.Printf("failed to start listening: %v", err)
	}
	
	go autoConnectToPeers(network)
	
	fmt.Println("🎯 auto mode active:")
	fmt.Println("   🌐 web ui: http://localhost:8080")
	fmt.Println("   📱 mobile api: http://localhost:8081")
	fmt.Println("   💬 terminal: type messages below")
	fmt.Println("")
	
	chat.Start()
	select {}
}

func setupNetwork(network *network.EnhancedP2PNetwork, mode string) {
	switch mode {
	case "listen":
		if err := network.Listen(":9000"); err != nil {
			log.Fatalf("failed to start listening: %v", err)
		}
		fmt.Println("🎧 listening for connections on :9000")

	case "connect":
		address := getConnectionAddress()
		if err := network.Connect(address); err != nil {
			log.Printf("failed to connect to %s: %v", address, err)
		} else {
			fmt.Printf("🔗 connected to %s\n", address)
		}

	case "discover":
		handlePeerDiscovery(network)

	case "auto":
		if err := network.Listen(":9000"); err != nil {
			log.Printf("failed to start listening: %v", err)
		} else {
			fmt.Println("🎧 listening for connections on :9000")
		}
		go autoConnectToPeers(network)
	}
}

func getOrCreateIdentity() (*identity.Identity, error) {
	configDir := filepath.Join(os.Getenv("HOME"), ".p2pchat")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	identityFile := filepath.Join(configDir, "identity.txt")

	if _, err := os.Stat(identityFile); os.IsNotExist(err) {
		fmt.Print("enter your username: ")
		reader := bufio.NewReader(os.Stdin)
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		if username == "" {
			username = "anonymous"
		}

		userIdentity, err := identity.NewIdentity(username)
		if err != nil {
			return nil, err
		}

		privKey, _ := userIdentity.ExportPrivateKey()
		if err := os.WriteFile(identityFile, []byte(username+"\n"+privKey), 0600); err != nil {
			return nil, err
		}

		fmt.Println("✅ new identity created and saved")
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

	fmt.Println("✅ existing identity loaded")
	return userIdentity, nil
}

func getUserNetworkMode() string {
	fmt.Println("\n🌐 network mode:")
	fmt.Println("1. listen for connections")
	fmt.Println("2. connect to specific peer")
	fmt.Println("3. discover and connect to peers")
	fmt.Println("4. auto mode (listen + discover)")
	fmt.Print("choose mode (1-4, default: 4): ")

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
	fmt.Print("enter peer address (host:port): ")
	reader := bufio.NewReader(os.Stdin)
	address, _ := reader.ReadString('\n')
	return strings.TrimSpace(address)
}

func handlePeerDiscovery(network *network.EnhancedP2PNetwork) {
	fmt.Println("🔍 discovering peers...")
	
	time.Sleep(2 * time.Second)
	
	peers := network.GetDiscoveredPeers()
	if len(peers) == 0 {
		fmt.Println("no peers discovered. starting in listen mode...")
		network.Listen(":9000")
		return
	}

	fmt.Println("📡 discovered peers:")
	for i, peer := range peers {
		fmt.Printf("%d. %s (%s) - %s\n", i+1, peer.User.Username, peer.User.ID, peer.Address)
	}

	fmt.Print("enter peer number to connect (or 0 to listen): ")
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
			fmt.Printf("failed to connect to %s: %v\n", peer.User.Username, err)
		} else {
			fmt.Printf("🔗 connected to %s\n", peer.User.Username)
		}
	}

	network.Listen(":9000")
}

func autoConnectToPeers(network *network.EnhancedP2PNetwork) {
	time.Sleep(3 * time.Second)

	peers := network.GetDiscoveredPeers()
	for _, peer := range peers {
		if err := network.ConnectToPeer(peer); err != nil {
			fmt.Printf("auto-connect to %s failed: %v\n", peer.User.Username, err)
		} else {
			fmt.Printf("🔗 auto-connected to %s\n", peer.User.Username)
		}
	}
}

func showMainMenu(network *network.EnhancedP2PNetwork) {
	fmt.Println("\n🎯 enhanced p2p chat v2.0 ready!")
	fmt.Println("💡 new features:")
	fmt.Println("   🌐 web ui: http://localhost:8080")
	fmt.Println("   📱 mobile api: http://localhost:8081/api")
	fmt.Println("   🔗 blockchain identity verification")
	fmt.Println("   🔐 enhanced ecdh encryption")
	fmt.Println("   💬 forward secure messaging")
	fmt.Println("")
	fmt.Println("💡 commands:")
	fmt.Println("   - type messages to chat")
	fmt.Println("   - use /help for all commands")
	fmt.Println("   - use /web to open web interface")
	fmt.Println("")
}

func generateUserID(pubKey string) string {
	hash := sha256.Sum256([]byte(pubKey))
	return hex.EncodeToString(hash[:])[:16]
}
