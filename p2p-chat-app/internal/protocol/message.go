package protocol

import (
	"encoding/json"
	"time"
)

// MessageType represents different types of messages
type MessageType string

const (
	TextMessage       MessageType = "text"
	FileMessage       MessageType = "file"
	TypingMessage     MessageType = "typing"
	JoinMessage       MessageType = "join"
	LeaveMessage      MessageType = "leave"
	HandshakeMessage  MessageType = "handshake"
	DeliveredMessage  MessageType = "delivered"
	ReadMessage       MessageType = "read"
	UserListMessage   MessageType = "userlist"
)

type Message struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	From      string      `json:"from"`
	To        string      `json:"to,omitempty"`        
	Room      string      `json:"room,omitempty"`      
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
	Encrypted bool        `json:"encrypted"`
	FileInfo  *FileInfo   `json:"file_info,omitempty"`
}

type FileInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Checksum string `json:"checksum"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
	Address   string `json:"address"`
	Online    bool   `json:"online"`
}

type HandshakeData struct {
	User      User   `json:"user"`
	Version   string `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

func SerializeMessage(msg *Message) ([]byte, error) {
	return json.Marshal(msg)
}

func DeserializeMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

func NewTextMessage(from, content string) *Message {
	return &Message{
		ID:        GenerateMessageID(),
		Type:      TextMessage,
		From:      from,
		Content:   content,
		Timestamp: time.Now(),
	}
}

func NewFileMessage(from, filename string, size int64, mimeType, checksum string) *Message {
	return &Message{
		ID:        GenerateMessageID(),
		Type:      FileMessage,
		From:      from,
		Content:   "File: " + filename,
		Timestamp: time.Now(),
		FileInfo: &FileInfo{
			Name:     filename,
			Size:     size,
			MimeType: mimeType,
			Checksum: checksum,
		},
	}
}

func GenerateMessageID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
