package chat

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"p2p-chat-app/internal/encryption"
	"p2p-chat-app/internal/identity"
	"p2p-chat-app/internal/protocol"
	"p2p-chat-app/internal/storage"
	"strings"
	"sync"
	"time"
)

type EnhancedChat struct {
	identity    *identity.Identity
	peers       map[string]net.Conn
	rooms       map[string]map[string]bool 
	currentRoom string
	mu          sync.RWMutex
	incoming    chan *protocol.Message
	storage     *storage.MessageStore
	running     bool
}

func NewEnhancedChat(userIdentity *identity.Identity, dataDir string) (*EnhancedChat, error) {
	store, err := storage.NewMessageStore(dataDir)
	if err != nil {
		return nil, err
	}

	return &EnhancedChat{
		identity:    userIdentity,
		peers:       make(map[string]net.Conn),
		rooms:       make(map[string]map[string]bool),
		currentRoom: "general", // Default room
		incoming:    make(chan *protocol.Message, 100),
		storage:     store,
	}, nil
}

func (ec *EnhancedChat) Start() {
	ec.running = true
	go ec.messageHandler()
	go ec.inputHandler()
}

func (ec *EnhancedChat) Stop() {
	ec.running = false
	close(ec.incoming)
}

func (ec *EnhancedChat) AddPeer(userID string, conn net.Conn) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	ec.peers[userID] = conn
	
	if ec.rooms[ec.currentRoom] == nil {
		ec.rooms[ec.currentRoom] = make(map[string]bool)
	}
	ec.rooms[ec.currentRoom][userID] = true
	
	fmt.Printf("✅ %s joined the chat\n", userID)
}

func (ec *EnhancedChat) RemovePeer(userID string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	delete(ec.peers, userID)
	
	for room := range ec.rooms {
		delete(ec.rooms[room], userID)
	}
	
	fmt.Printf("❌ %s left the chat\n", userID)
}

func (ec *EnhancedChat) SendMessage(content string, to string) error {
	msg := &protocol.Message{
		ID:        protocol.GenerateMessageID(),
		Type:      protocol.TextMessage,
		From:      ec.identity.ID,
		Content:   content,
		Timestamp: time.Now(),
	}

	if to != "" {
		msg.To = to
	} else {
		msg.Room = ec.currentRoom
	}

	if err := ec.storage.StoreMessage(msg); err != nil {
		fmt.Printf("Error storing message: %v\n", err)
	}

	return ec.broadcastMessage(msg)
}

func (ec *EnhancedChat) SendFile(filename string, to string) error {
	msg := protocol.NewFileMessage(ec.identity.ID, filename, 0, "application/octet-stream", "")
	
	if to != "" {
		msg.To = to
	} else {
		msg.Room = ec.currentRoom
	}

	if err := ec.storage.StoreMessage(msg); err != nil {
		fmt.Printf("Error storing file message: %v\n", err)
	}

	return ec.broadcastMessage(msg)
}

func (ec *EnhancedChat) JoinRoom(roomName string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	oldRoom := ec.currentRoom
	ec.currentRoom = roomName
	
	if ec.rooms[roomName] == nil {
		ec.rooms[roomName] = make(map[string]bool)
	}
	ec.rooms[roomName][ec.identity.ID] = true
	
	if oldRoom != roomName && ec.rooms[oldRoom] != nil {
		delete(ec.rooms[oldRoom], ec.identity.ID)
	}
	
	fmt.Printf("📋 Joined room: %s\n", roomName)
	
	ec.displayRecentMessages(roomName)
}

func (ec *EnhancedChat) GetRooms() []string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	var rooms []string
	for room := range ec.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

func (ec *EnhancedChat) GetMessages(key string, limit int) ([]*protocol.Message, error) {
	return ec.storage.GetMessages(key, limit)
}

func (ec *EnhancedChat) ListRooms() {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	fmt.Println("📋 Available rooms:")
	for room, users := range ec.rooms {
		userCount := len(users)
		current := ""
		if room == ec.currentRoom {
			current = " (current)"
		}
		fmt.Printf("  - %s (%d users)%s\n", room, userCount, current)
	}
}

func (ec *EnhancedChat) ListUsers() {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	fmt.Printf("👥 Users in %s:\n", ec.currentRoom)
	if users, exists := ec.rooms[ec.currentRoom]; exists {
		for userID := range users {
			current := ""
			if userID == ec.identity.ID {
				current = " (you)"
			}
			fmt.Printf("  - %s%s\n", userID, current)
		}
	}
}

// SearchMessages searches for messages
func (ec *EnhancedChat) SearchMessages(query string, roomKey string) ([]*protocol.Message, error) {
	return ec.storage.SearchMessages(query, roomKey)
}

func (ec *EnhancedChat) ProcessIncomingMessage(data string) {
	decrypted, err := encryption.Decrypt(strings.TrimSpace(data), key)
	if err != nil {
		fmt.Printf("Decryption error: %v\n", err)
		return
	}

	msg, err := protocol.DeserializeMessage(decrypted)
	if err != nil {
		fmt.Printf("Message parsing error: %v\n", err)
		return
	}

	if err := ec.storage.StoreMessage(msg); err != nil {
		fmt.Printf("Error storing incoming message: %v\n", err)
	}

	select {
	case ec.incoming <- msg:
	default:
		fmt.Println("Message queue full, dropping message")
	}
}

func (ec *EnhancedChat) broadcastMessage(msg *protocol.Message) error {
	data, err := protocol.SerializeMessage(msg)
	if err != nil {
		return err
	}

	encrypted, err := encryption.Encrypt(data, key)
	if err != nil {
		return err
	}

	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if msg.To != "" {
		if conn, exists := ec.peers[msg.To]; exists {
			fmt.Fprintln(conn, encrypted)
		}
	} else {
		for userID := range ec.rooms[msg.Room] {
			if userID != ec.identity.ID {
				if conn, exists := ec.peers[userID]; exists {
					fmt.Fprintln(conn, encrypted)
				}
			}
		}
	}

	return nil
}

func (ec *EnhancedChat) messageHandler() {
	for msg := range ec.incoming {
		if !ec.running {
			break
		}

		switch msg.Type {
		case protocol.TextMessage:
			ec.displayMessage(msg)
		case protocol.FileMessage:
			ec.displayFileMessage(msg)
		case protocol.TypingMessage:
			ec.displayTypingIndicator(msg)
		}
	}
}

func (ec *EnhancedChat) inputHandler() {
	reader := bufio.NewReader(os.Stdin)
	
	ec.displayHelp()
	fmt.Printf("\n💬 Welcome to P2P Chat! You are in room '%s'\n", ec.currentRoom)
	fmt.Print("> ")

	for ec.running {
		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Print("> ")
			continue
		}

		if ec.handleCommand(input) {
			fmt.Print("> ")
			continue
		}

		if err := ec.SendMessage(input, ""); err != nil {
			fmt.Printf("Error sending message: %v\n", err)
		}
		fmt.Print("> ")
	}
}

func (ec *EnhancedChat) handleCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "/") {
		return false
	}

	command := parts[0][1:] 
	args := parts[1:]

	switch command {
	case "help":
		ec.displayHelp()
	case "rooms":
		ec.ListRooms()
	case "join":
		if len(args) > 0 {
			ec.JoinRoom(args[0])
		} else {
			fmt.Println("Usage: /join <room_name>")
		}
	case "users":
		ec.ListUsers()
	case "search":
		if len(args) > 0 {
			query := strings.Join(args, " ")
			messages, err := ec.SearchMessages(query, "room:"+ec.currentRoom)
			if err != nil {
				fmt.Printf("search error: %v\n", err)
			} else {
				fmt.Printf("🔍 search results for '%s':\n", query)
				for _, msg := range messages {
					timestamp := msg.Timestamp.Format("15:04:05")
					fmt.Printf("[%s] %s: %s\n", timestamp, msg.From, msg.Content)
				}
			}
		} else {
			fmt.Println("usage: /search <query>")
		}
	case "private", "pm":
		if len(args) >= 2 {
			userID := args[0]
			message := strings.Join(args[1:], " ")
			if err := ec.SendMessage(message, userID); err != nil {
				fmt.Printf("Error sending private message: %v\n", err)
			}
		} else {
			fmt.Println("Usage: /private <user_id> <message>")
		}
	case "file":
		if len(args) > 0 {
			filename := args[0]
			to := ""
			if len(args) > 1 {
				to = args[1]
			}
			if err := ec.SendFile(filename, to); err != nil {
				fmt.Printf("Error sending file: %v\n", err)
			}
		} else {
			fmt.Println("Usage: /file <filename> [user_id]")
		}
	case "quit", "exit":
		ec.Stop()
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: /%s\n", command)
		ec.displayHelp()
	}

	return true
}

func (ec *EnhancedChat) displayHelp() {
	fmt.Println("\n📖 Available commands:")
	fmt.Println("  /help              - Show this help")
	fmt.Println("  /rooms             - List available rooms")
	fmt.Println("  /join <room>       - Join a room")
	fmt.Println("  /users             - List users in current room")
	fmt.Println("  /search <query>    - Search messages")
	fmt.Println("  /private <user> <msg> - Send private message")
	fmt.Println("  /file <filename>   - Share a file")
	fmt.Println("  /quit              - Exit the chat")
	fmt.Println("  Any other text will be sent as a message to the current room")
}

func (ec *EnhancedChat) displayMessage(msg *protocol.Message) {
	timestamp := msg.Timestamp.Format("15:04:05")
	if msg.To != "" {
		if msg.From == ec.identity.ID {
			fmt.Printf("\r🔒 [%s] To %s: %s\n> ", timestamp, msg.To, msg.Content)
		} else {
			fmt.Printf("\r🔒 [%s] From %s: %s\n> ", timestamp, msg.From, msg.Content)
		}
	} else {
		fmt.Printf("\r💬 [%s] %s: %s\n> ", timestamp, msg.From, msg.Content)
	}
}

func (ec *EnhancedChat) displayFileMessage(msg *protocol.Message) {
	timestamp := msg.Timestamp.Format("15:04:05")
	fmt.Printf("\r📎 [%s] %s shared a file: %s\n> ", timestamp, msg.From, msg.FileInfo.Name)
}

func (ec *EnhancedChat) displayTypingIndicator(msg *protocol.Message) {
	fmt.Printf("\r✏️  %s is typing...\n> ", msg.From)
}

func (ec *EnhancedChat) displayRecentMessages(room string) {
	messages, err := ec.storage.GetMessages("room:"+room, 10)
	if err != nil {
		fmt.Printf("Error loading messages: %v\n", err)
		return
	}

	if len(messages) > 0 {
		fmt.Println("📜 Recent messages:")
		for _, msg := range messages {
			ec.displayMessage(msg)
		}
	}
}
