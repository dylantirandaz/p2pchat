package mobile

import (
	"encoding/json"
	"fmt"
	"net/http"
	"p2p-chat-app/internal/chat"
	"p2p-chat-app/internal/network"
	"strconv"
	"time"
)

type MobileAPI struct {
	chat    *chat.EnhancedChat
	network *network.EnhancedP2PNetwork
	port    string
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type MessageRequest struct {
	Content string `json:"content"`
	To      string `json:"to,omitempty"`
	Room    string `json:"room,omitempty"`
}

type ConnectRequest struct {
	Address string `json:"address"`
}

func NewMobileAPI(port string, chat *chat.EnhancedChat, network *network.EnhancedP2PNetwork) *MobileAPI {
	return &MobileAPI{
		chat:    chat,
		network: network,
		port:    port,
	}
}

func (api *MobileAPI) Start() error {
	http.HandleFunc("/api/messages", api.handleMessages)
	http.HandleFunc("/api/send", api.handleSend)
	http.HandleFunc("/api/rooms", api.handleRooms)
	http.HandleFunc("/api/join", api.handleJoin)
	http.HandleFunc("/api/peers", api.handlePeers)
	http.HandleFunc("/api/connect", api.handleConnect)
	http.HandleFunc("/api/discover", api.handleDiscover)
	http.HandleFunc("/api/status", api.handleStatus)
	http.HandleFunc("/api/search", api.handleSearch)

	// cors middleware
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			return
		}
		
		http.NotFound(w, r)
	})

	return http.ListenAndServe(":"+api.port, nil)
}

func (api *MobileAPI) handleMessages(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	room := r.URL.Query().Get("room")
	limitStr := r.URL.Query().Get("limit")
	
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	
	if room == "" {
		room = "general"
	}
	
	messages, err := api.chat.GetMessages("room:"+room, limit)
	if err != nil {
		api.sendError(w, err.Error())
		return
	}
	
	api.sendSuccess(w, messages)
}

func (api *MobileAPI) handleSend(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	if r.Method != "POST" {
		api.sendError(w, "method not allowed")
		return
	}
	
	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, "invalid json")
		return
	}
	
	err := api.chat.SendMessage(req.Content, req.To)
	if err != nil {
		api.sendError(w, err.Error())
		return
	}
	
	api.sendSuccess(w, map[string]string{"status": "sent"})
}

func (api *MobileAPI) handleRooms(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	rooms := api.chat.GetRooms()
	api.sendSuccess(w, rooms)
}

func (api *MobileAPI) handleJoin(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	if r.Method != "POST" {
		api.sendError(w, "method not allowed")
		return
	}
	
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, "invalid json")
		return
	}
	
	room := req["room"]
	if room == "" {
		api.sendError(w, "room name required")
		return
	}
	
	api.chat.JoinRoom(room)
	api.sendSuccess(w, map[string]string{"room": room})
}

func (api *MobileAPI) handlePeers(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	connected := api.network.GetConnectedPeers()
	discovered := api.network.GetDiscoveredPeers()
	
	result := map[string]interface{}{
		"connected":  connected,
		"discovered": discovered,
	}
	
	api.sendSuccess(w, result)
}

func (api *MobileAPI) handleConnect(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	if r.Method != "POST" {
		api.sendError(w, "method not allowed")
		return
	}
	
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, "invalid json")
		return
	}
	
	err := api.network.Connect(req.Address)
	if err != nil {
		api.sendError(w, err.Error())
		return
	}
	
	api.sendSuccess(w, map[string]string{"status": "connected"})
}

func (api *MobileAPI) handleDiscover(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	peers := api.network.GetDiscoveredPeers()
	api.sendSuccess(w, peers)
}

func (api *MobileAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	status := map[string]interface{}{
		"timestamp":        time.Now(),
		"connected_peers":  len(api.network.GetConnectedPeers()),
		"discovered_peers": len(api.network.GetDiscoveredPeers()),
	}
	
	api.sendSuccess(w, status)
}

func (api *MobileAPI) handleSearch(w http.ResponseWriter, r *http.Request) {
	api.setCORSHeaders(w)
	
	query := r.URL.Query().Get("q")
	room := r.URL.Query().Get("room")
	
	if query == "" {
		api.sendError(w, "query required")
		return
	}
	
	if room == "" {
		room = "general"
	}
	
	messages, err := api.chat.SearchMessages(query, "room:"+room)
	if err != nil {
		api.sendError(w, err.Error())
		return
	}
	
	api.sendSuccess(w, messages)
}

func (api *MobileAPI) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (api *MobileAPI) sendSuccess(w http.ResponseWriter, data interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	json.NewEncoder(w).Encode(response)
}

func (api *MobileAPI) sendError(w http.ResponseWriter, message string) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}

// mobile app configuration
type AppConfig struct {
	APIBaseURL      string `json:"api_base_url"`
	WebSocketURL    string `json:"websocket_url"`
	DiscoveryPort   int    `json:"discovery_port"`
	ChatPort        int    `json:"chat_port"`
	EnablePushNotif bool   `json:"enable_push_notifications"`
	Theme           string `json:"theme"`
}

func (api *MobileAPI) GetAppConfig() *AppConfig {
	return &AppConfig{
		APIBaseURL:      fmt.Sprintf("http://localhost:%s/api", api.port),
		WebSocketURL:    fmt.Sprintf("ws://localhost:8080/ws"),
		DiscoveryPort:   9001,
		ChatPort:        9000,
		EnablePushNotif: true,
		Theme:           "dark",
	}
}
