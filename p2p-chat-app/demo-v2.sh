#!/bin/bash

echo "🚀 p2p chat v2.0 - next level demo"
echo "==================================="

echo "📦 building v2..."
go build -o p2pchat-v2 ./cmd/enhanced_main_v2.go

if [ $? -ne 0 ]; then
    echo "❌ build failed"
    exit 1
fi

echo "✅ v2 build successful"
echo ""

echo "🎯 new in v2.0:"
echo "🌐 web ui interface - modern browser-based chat"
echo "📱 mobile api - rest api for mobile apps"
echo "🔗 blockchain identity - cryptographic verification"
echo "🔐 ecdh encryption - perfect forward secrecy"
echo "🖥️  multi-interface - terminal, web, headless modes"
echo "⚡ auto-discovery - seamless peer connection"
echo ""

echo "🌐 web interface features:"
echo "   • real-time messaging via websockets"
echo "   • dark theme ui"
echo "   • room management"
echo "   • peer discovery"
echo "   • message history"
echo ""

echo "📱 mobile api endpoints:"
echo "   • GET  /api/messages?room=<room>&limit=<n>"
echo "   • POST /api/send {content, to, room}"
echo "   • GET  /api/rooms"
echo "   • POST /api/join {room}"
echo "   • GET  /api/peers"
echo "   • POST /api/connect {address}"
echo "   • GET  /api/discover"
echo "   • GET  /api/status"
echo "   • GET  /api/search?q=<query>&room=<room>"
echo ""

echo "🔗 blockchain features:"
echo "   • proof-of-work identity registration"
echo "   • cryptographic signature verification"
echo "   • key exchange tracking"
echo "   • tamper-proof identity chain"
echo ""

echo "🔐 enhanced encryption:"
echo "   • ecdh key exchange (p-256)"
echo "   • aes-256-gcm per-message encryption"
echo "   • forward secure messaging"
echo "   • perfect forward secrecy"
echo ""

echo "🖥️  interface modes:"
echo "   1. terminal - classic command-line interface"
echo "   2. web only - browser interface at localhost:8080"
echo "   3. headless - api-only mode for automation"
echo "   4. auto - all interfaces enabled"
echo ""

echo "🚀 ready to run v2!"
echo "execute: ./p2pchat-v2"
echo ""

echo "💡 v2 pro tips:"
echo "   • choose auto mode for full experience"
echo "   • open web ui at http://localhost:8080"
echo "   • mobile api available at localhost:8081"
echo "   • blockchain verifies all identities"
echo "   • forward secrecy protects message history"
