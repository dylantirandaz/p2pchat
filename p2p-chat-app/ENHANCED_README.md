# Enhanced P2P Chat Application

A decentralized, encrypted peer-to-peer chat application similar to Telegram/WhatsApp, built in Go.

## 🚀 Features

### Core Features
- **Decentralized P2P Communication** - No central servers required
- **End-to-End Encryption** - AES-256-GCM encryption for all messages
- **Cryptographic Identity** - RSA key pairs for user authentication
- **Persistent Message Storage** - Local message history
- **Auto Peer Discovery** - UDP broadcast discovery on local network
- **Multiple Chat Rooms** - Create and join different chat rooms
- **Private Messages** - Send direct messages to specific users
- **File Sharing** - Share files between peers (basic implementation)
- **Rich Terminal UI** - Command-based interface with help system

### Advanced Features
- **Message History** - Persistent storage with search capabilities
- **User Management** - Identity persistence and public key verification
- **Network Resilience** - Automatic reconnection and peer management
- **Typing Indicators** - Real-time typing status (framework ready)
- **Message Status** - Delivered/read receipts (framework ready)
- **Room Management** - Create, join, and manage multiple chat rooms

## 🛠 Installation

### Prerequisites
- Go 1.18 or higher
- macOS, Linux, or Windows

### Build from Source
```bash
git clone <your-repo>
cd p2p-chat-app
go build -o p2pchat-enhanced ./cmd/enhanced_main.go
```

## 🎮 Usage

### Starting the Application
```bash
./p2pchat-enhanced
```

### First Run
1. Enter your desired username
2. Choose network mode:
   - **Listen**: Wait for connections from other peers
   - **Connect**: Connect to a specific peer address
   - **Discover**: Automatically find and connect to peers
   - **Auto**: Listen and auto-connect to discovered peers (recommended)

### Chat Commands
Once connected, use these commands:

#### Basic Commands
- `<message>` - Send a message to the current room
- `/help` - Show all available commands
- `/quit` or `/exit` - Exit the application

#### Room Management
- `/rooms` - List all available rooms
- `/join <room_name>` - Join or create a chat room
- `/users` - List users in the current room

#### Private Messaging
- `/private <user_id> <message>` - Send a private message
- `/pm <user_id> <message>` - Alias for private message

#### File Sharing
- `/file <filename>` - Share a file with the current room
- `/file <filename> <user_id>` - Share a file with a specific user

#### Search & History
- `/search <query>` - Search messages in current room

## 🔧 Technical Architecture

### Components

1. **Identity System** (`internal/identity/`)
   - RSA key pair generation and management
   - Public key cryptography for authentication
   - Identity persistence and loading

2. **Protocol Layer** (`internal/protocol/`)
   - JSON-based message protocol
   - Support for multiple message types (text, file, typing, etc.)
   - Message serialization and deserialization

3. **Network Layer** (`internal/network/`)
   - TCP connections for reliable message delivery
   - Peer discovery via UDP broadcasts
   - Connection management and handshaking

4. **Encryption** (`internal/encryption/`)
   - AES-256-GCM encryption for message content
   - Per-message nonce generation
   - Secure key derivation (to be enhanced)

5. **Storage** (`internal/storage/`)
   - Local message persistence
   - Room-based message organization
   - Message search and retrieval

6. **Chat System** (`internal/chat/`)
   - User interface and command processing
   - Message routing and display
   - Room and user management

### Message Protocol
Messages use a JSON-based protocol with these types:
- `text` - Regular chat messages
- `file` - File sharing metadata
- `typing` - Typing indicators
- `join/leave` - Room management
- `handshake` - Initial peer authentication
- `delivered/read` - Message status updates

### Security Features
- **RSA-2048** key pairs for identity
- **AES-256-GCM** for message encryption
- **SHA-256** for hashing and verification
- **Nonce-based encryption** to prevent replay attacks

## 🌐 Network Discovery

The application uses UDP broadcast on port 9001 for peer discovery:
1. Peers periodically broadcast their presence
2. Other peers respond with their information
3. Discovered peers can be connected to automatically or manually

## 📁 File Structure
```
~/.p2pchat/
├── identity.txt     # Your cryptographic identity
└── data/            # Message storage
    ├── room_general.json
    ├── private_user1_user2.json
    └── ...
```

## 🔜 Planned Enhancements

### Short Term
- [ ] Better file transfer implementation
- [ ] Message encryption with per-peer keys
- [ ] Improved UI with colored output
- [ ] NAT traversal for internet-wide P2P

### Medium Term
- [ ] Web UI interface
- [ ] Mobile companion app
- [ ] Voice/video calling
- [ ] Group voice channels

### Long Term
- [ ] Blockchain-based identity verification
- [ ] Distributed hash table (DHT) for global peer discovery
- [ ] Smart contracts for group management
- [ ] Integration with IPFS for file storage

## 🤝 Contributing

This is a learning project showcasing P2P networking concepts. Feel free to:
- Report issues
- Suggest improvements
- Submit pull requests
- Fork for your own experiments

## 📄 License

MIT License - See LICENSE file for details

## 🙏 Acknowledgments

Built with Go's excellent standard library and inspired by:
- BitTorrent protocol design
- Matrix protocol concepts
- Signal's double ratchet algorithm
- IPFS networking stack

---

**Note**: This is an educational project >_<>. For production use, additional security audits and hardening would be required.
