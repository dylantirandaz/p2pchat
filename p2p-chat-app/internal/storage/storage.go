package storage

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"p2p-chat-app/internal/protocol"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type MessageStore struct {
	dataDir  string
	messages map[string][]*protocol.Message 
	mu       sync.RWMutex
}

func NewMessageStore(dataDir string) (*MessageStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	store := &MessageStore{
		dataDir:  dataDir,
		messages: make(map[string][]*protocol.Message),
	}

	if err := store.loadMessages(); err != nil {
		return nil, err
	}

	return store, nil
}

func (ms *MessageStore) StoreMessage(msg *protocol.Message) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	key := ms.getStorageKey(msg)
	ms.messages[key] = append(ms.messages[key], msg)

	sort.Slice(ms.messages[key], func(i, j int) bool {
		return ms.messages[key][i].Timestamp.Before(ms.messages[key][j].Timestamp)
	})

	return ms.saveMessages(key)
}

func (ms *MessageStore) GetMessages(roomOrUser string, limit int) ([]*protocol.Message, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	messages := ms.messages[roomOrUser]
	if messages == nil {
		return []*protocol.Message{}, nil
	}

	start := 0
	if limit > 0 && len(messages) > limit {
		start = len(messages) - limit
	}

	result := make([]*protocol.Message, len(messages)-start)
	copy(result, messages[start:])
	return result, nil
}

func (ms *MessageStore) GetAllRooms() []string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var rooms []string
	for room := range ms.messages {
		rooms = append(rooms, room)
	}
	sort.Strings(rooms)
	return rooms
}

func (ms *MessageStore) SearchMessages(query string, roomOrUser string) ([]*protocol.Message, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var results []*protocol.Message
	messages := ms.messages[roomOrUser]

	for _, msg := range messages {
		if contains(msg.Content, query) {
			results = append(results, msg)
		}
	}

	return results, nil
}

func (ms *MessageStore) DeleteOldMessages(olderThan time.Duration) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	for key, messages := range ms.messages {
		var kept []*protocol.Message
		for _, msg := range messages {
			if msg.Timestamp.After(cutoff) {
				kept = append(kept, msg)
			}
		}
		ms.messages[key] = kept
		if err := ms.saveMessages(key); err != nil {
			return err
		}
	}

	return nil
}

func (ms *MessageStore) getStorageKey(msg *protocol.Message) string {
	if msg.Room != "" {
		return "room:" + msg.Room
	}
	if msg.To != "" {
		if msg.From < msg.To {
			return "private:" + msg.From + ":" + msg.To
		}
		return "private:" + msg.To + ":" + msg.From
	}
	return "global"
}

func (ms *MessageStore) saveMessages(key string) error {
	filename := filepath.Join(ms.dataDir, sanitizeFilename(key)+".json")
	data, err := json.MarshalIndent(ms.messages[key], "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func (ms *MessageStore) loadMessages() error {
	files, err := ioutil.ReadDir(ms.dataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filename := filepath.Join(ms.dataDir, file.Name())
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			continue 
		}

		var messages []*protocol.Message
		if err := json.Unmarshal(data, &messages); err != nil {
			continue 
		}

		key := file.Name()[:len(file.Name())-5] 
		key = unsanitizeFilename(key)
		ms.messages[key] = messages
	}

	return nil
}

func sanitizeFilename(s string) string {
	safe := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			safe += string(r)
		} else {
			safe += "_"
		}
	}
	return safe
}

func unsanitizeFilename(s string) string {
	return s
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
