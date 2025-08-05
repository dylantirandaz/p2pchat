package webui

import (
	"encoding/json"
	"html/template"
	"net/http"
	"p2p-chat-app/internal/chat"
	"p2p-chat-app/internal/network"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebServer struct {
	port     string
	chat     *chat.EnhancedChat
	network  *network.EnhancedP2PNetwork
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex
	upgrader websocket.Upgrader
}

type WebMessage struct {
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	From      string    `json:"from"`
	To        string    `json:"to,omitempty"`
	Room      string    `json:"room,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func NewWebServer(port string, chat *chat.EnhancedChat, network *network.EnhancedP2PNetwork) *WebServer {
	return &WebServer{
		port:    port,
		chat:    chat,
		network: network,
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (ws *WebServer) Start() error {
	http.HandleFunc("/", ws.handleHome)
	http.HandleFunc("/ws", ws.handleWebSocket)
	http.HandleFunc("/api/rooms", ws.handleRooms)
	http.HandleFunc("/api/peers", ws.handlePeers)
	http.HandleFunc("/api/messages", ws.handleMessages)
	http.HandleFunc("/static/", ws.handleStatic)

	return http.ListenAndServe(":"+ws.port, nil)
}

func (ws *WebServer) handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>p2p chat</title>
    <meta charset="utf-8">
    <style>
        body { font-family: -apple-system, sans-serif; margin: 0; background: #1a1a1a; color: #fff; }
        .container { display: flex; height: 100vh; }
        .sidebar { width: 250px; background: #2d2d2d; border-right: 1px solid #444; }
        .main { flex: 1; display: flex; flex-direction: column; }
        .header { padding: 15px; border-bottom: 1px solid #444; background: #2d2d2d; }
        .messages { flex: 1; overflow-y: auto; padding: 10px; }
        .input-area { padding: 15px; border-top: 1px solid #444; background: #2d2d2d; }
        .room-list, .peer-list { padding: 10px; }
        .room-item, .peer-item { padding: 8px; cursor: pointer; border-radius: 4px; margin: 2px 0; }
        .room-item:hover, .peer-item:hover { background: #444; }
        .room-item.active { background: #0066cc; }
        .message { margin: 8px 0; padding: 8px; border-radius: 6px; background: #333; }
        .message.own { background: #0066cc; margin-left: 50px; }
        .message.private { background: #cc6600; }
        .message-info { font-size: 12px; opacity: 0.7; margin-bottom: 4px; }
        input[type="text"] { width: 100%; padding: 10px; border: 1px solid #444; background: #1a1a1a; color: #fff; border-radius: 4px; }
        button { padding: 10px 15px; background: #0066cc; color: #fff; border: none; border-radius: 4px; cursor: pointer; margin-left: 10px; }
        button:hover { background: #0052a3; }
        .section-title { padding: 10px; font-weight: bold; border-bottom: 1px solid #444; }
        .status { padding: 5px 10px; font-size: 12px; background: #444; }
        .online { color: #4CAF50; }
        .offline { color: #f44336; }
    </style>
</head>
<body>
    <div class="container">
        <div class="sidebar">
            <div class="section-title">rooms</div>
            <div class="room-list" id="roomList"></div>
            <div class="section-title">peers</div>
            <div class="peer-list" id="peerList"></div>
            <div class="status" id="status">connecting...</div>
        </div>
        <div class="main">
            <div class="header">
                <span id="currentRoom">general</span>
                <button onclick="joinRoom()">join room</button>
                <button onclick="connectPeer()">connect peer</button>
            </div>
            <div class="messages" id="messages"></div>
            <div class="input-area">
                <input type="text" id="messageInput" placeholder="type message..." onkeypress="handleKeyPress(event)">
                <button onclick="sendMessage()">send</button>
            </div>
        </div>
    </div>

    <script>
        let ws;
        let currentRoom = 'general';
        let username = '';

        function connect() {
            ws = new WebSocket('ws://localhost:8080/ws');
            
            ws.onopen = function() {
                document.getElementById('status').innerHTML = '<span class="online">connected</span>';
            };
            
            ws.onmessage = function(event) {
                const msg = JSON.parse(event.data);
                handleMessage(msg);
            };
            
            ws.onclose = function() {
                document.getElementById('status').innerHTML = '<span class="offline">disconnected</span>';
                setTimeout(connect, 3000);
            };
        }

        function handleMessage(msg) {
            if (msg.type === 'message') {
                addMessage(msg);
            } else if (msg.type === 'rooms') {
                updateRooms(msg.data);
            } else if (msg.type === 'peers') {
                updatePeers(msg.data);
            }
        }

        function addMessage(msg) {
            const messages = document.getElementById('messages');
            const div = document.createElement('div');
            const isOwn = msg.from === username;
            const isPrivate = msg.to && msg.to !== '';
            
            div.className = 'message' + (isOwn ? ' own' : '') + (isPrivate ? ' private' : '');
            
            const time = new Date(msg.timestamp).toLocaleTimeString();
            const prefix = isPrivate ? (isOwn ? 'to ' + msg.to : 'from ' + msg.from) : msg.from;
            
            div.innerHTML = '<div class="message-info">' + time + ' - ' + prefix + '</div>' + msg.content;
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        }

        function sendMessage() {
            const input = document.getElementById('messageInput');
            const content = input.value.trim();
            if (!content) return;

            const msg = {
                type: 'send',
                content: content,
                room: currentRoom
            };

            ws.send(JSON.stringify(msg));
            input.value = '';
        }

        function handleKeyPress(event) {
            if (event.key === 'Enter') {
                sendMessage();
            }
        }

        function joinRoom() {
            const room = prompt('enter room name:');
            if (room) {
                currentRoom = room;
                document.getElementById('currentRoom').textContent = room;
                ws.send(JSON.stringify({type: 'join', room: room}));
            }
        }

        function connectPeer() {
            const address = prompt('enter peer address (host:port):');
            if (address) {
                ws.send(JSON.stringify({type: 'connect', address: address}));
            }
        }

        function updateRooms(rooms) {
            const list = document.getElementById('roomList');
            list.innerHTML = '';
            rooms.forEach(room => {
                const div = document.createElement('div');
                div.className = 'room-item';
                if (room === currentRoom) div.className += ' active';
                div.textContent = room;
                div.onclick = () => {
                    currentRoom = room;
                    document.getElementById('currentRoom').textContent = room;
                    updateRooms(rooms);
                };
                list.appendChild(div);
            });
        }

        function updatePeers(peers) {
            const list = document.getElementById('peerList');
            list.innerHTML = '';
            peers.forEach(peer => {
                const div = document.createElement('div');
                div.className = 'peer-item';
                div.innerHTML = '<span class="' + (peer.online ? 'online' : 'offline') + '">●</span> ' + peer.username;
                list.appendChild(div);
            });
        }

        connect();
    </script>
</body>
</html>`

	t, _ := template.New("home").Parse(tmpl)
	t.Execute(w, nil)
}

func (ws *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ws.mu.Lock()
	ws.clients[conn] = true
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		delete(ws.clients, conn)
		ws.mu.Unlock()
	}()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		switch msg["type"] {
		case "send":
			ws.chat.SendMessage(msg["content"].(string), "")
		case "join":
			ws.chat.JoinRoom(msg["room"].(string))
		case "connect":
			ws.network.Connect(msg["address"].(string))
		}
	}
}

func (ws *WebServer) handleRooms(w http.ResponseWriter, r *http.Request) {
	rooms := ws.chat.GetRooms()
	json.NewEncoder(w).Encode(rooms)
}

func (ws *WebServer) handlePeers(w http.ResponseWriter, r *http.Request) {
	peers := ws.network.GetConnectedPeers()
	json.NewEncoder(w).Encode(peers)
}

func (ws *WebServer) handleMessages(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" {
		room = "general"
	}
	
	messages, _ := ws.chat.GetMessages("room:"+room, 50)
	json.NewEncoder(w).Encode(messages)
}

func (ws *WebServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/"+r.URL.Path[8:])
}

func (ws *WebServer) BroadcastMessage(msg *WebMessage) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	for client := range ws.clients {
		client.WriteJSON(msg)
	}
}
