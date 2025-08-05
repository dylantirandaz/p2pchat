#!/bin/bash

echo "🚀 Enhanced P2P Chat Demo"
echo "========================="

echo "📦 Building application..."
go build -o p2pchat-enhanced ./cmd/enhanced_main.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

echo ""
echo "🎯 New Features Added:"
echo "✨ Cryptographic identity system (RSA key pairs)"
echo "✨ JSON-based message protocol"
echo "✨ Persistent message storage"
echo "✨ Auto peer discovery via UDP broadcast"
echo "✨ Multiple chat rooms support"
echo "✨ Private messaging"
echo "✨ File sharing framework"
echo "✨ Message search and history"
echo "✨ Enhanced terminal UI with commands"
echo "✨ Better encryption (AES-256-GCM)"
echo "✨ Connection management and handshaking"
echo ""

echo "🎮 Usage Examples:"
echo "📝 Chat Commands:"
echo "   /help              - Show all commands"
echo "   /rooms             - List available rooms"
echo "   /join <room>       - Join/create a room"
echo "   /users             - List users in room"
echo "   /private <user> <msg> - Send private message"
echo "   /search <query>    - Search message history"
echo "   /file <filename>   - Share a file"
echo ""

echo "🌐 Network Modes:"
echo "   1. Listen mode     - Wait for connections"
echo "   2. Connect mode    - Connect to specific peer"
echo "   3. Discovery mode  - Find and connect to peers"
echo "   4. Auto mode       - Listen + auto-discover (recommended)"
echo ""

echo "📁 Data Storage:"
echo "   ~/.p2pchat/identity.txt    - Your identity"
echo "   ~/.p2pchat/data/           - Message history"
echo ""

echo "🔐 Security Features:"
echo "   • RSA-2048 key pairs for identity"
echo "   • AES-256-GCM message encryption"
echo "   • SHA-256 hashing and verification"
echo "   • Nonce-based encryption"
echo ""

echo "🚀 Ready to run! Execute: ./p2pchat-enhanced"
echo ""
echo "💡 Pro Tips:"
echo "   • Run multiple instances to test P2P functionality"
echo "   • Try auto mode for easy peer discovery"
echo "   • Use /help in the app for full command list"
echo "   • Message history persists between sessions"
