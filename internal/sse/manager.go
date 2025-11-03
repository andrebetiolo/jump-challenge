package sse

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
)

// SSEManager manages Server-Sent Event connections
type SSEManager struct {
	clients    map[string]map[chan []byte]bool // userID -> connection channels
	clientsMux sync.RWMutex
	
	broadcast chan []byte
	logger    *logger.Logger
	
	// Context for managing the SSE service lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSSEManager creates a new SSE manager
func NewSSEManager(logger *logger.Logger) *SSEManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &SSEManager{
		clients:   make(map[string]map[chan []byte]bool),
		broadcast: make(chan []byte, 100), // Buffered channel for broadcasting
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
	
	// Start the broadcaster goroutine
	go manager.broadcastEvents()
	
	return manager
}

// AddClient adds a new client connection for a specific user
func (s *SSEManager) AddClient(userID string) chan []byte {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	
	// Create user-specific clients map if it doesn't exist
	if s.clients[userID] == nil {
		s.clients[userID] = make(map[chan []byte]bool)
	}
	
	// Create a new channel for this client
	channel := make(chan []byte, 10) // Buffered channel for this specific client
	s.clients[userID][channel] = true
	
	s.logger.Info("Added SSE client for user:", userID, "total clients:", len(s.clients[userID]))
	
	return channel
}

// RemoveClient removes a client connection
func (s *SSEManager) RemoveClient(userID string, channel chan []byte) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	
	if userClients, exists := s.clients[userID]; exists {
		delete(userClients, channel)
		
		// Close the channel to free resources
		close(channel)
		
		s.logger.Info("Removed SSE client for user:", userID, "remaining clients:", len(userClients))
		
		// If this was the last client for the user, remove the user's map
		if len(userClients) == 0 {
			delete(s.clients, userID)
			s.logger.Info("Removed empty user SSE map for user:", userID)
		}
	}
}

// BroadcastEmailToUser broadcasts an email to a specific user
func (s *SSEManager) BroadcastEmailToUser(userID string, email *model.Email) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	
	userClients, exists := s.clients[userID]
	if !exists {
		return // No active connections for this user
	}
	
	// Prepare the event data
	event := map[string]interface{}{
		"type":  "new_email",
		"data":  email,
		"time":  time.Now().Unix(),
	}
	
	jsonData, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("Failed to marshal email event:", err)
		return
	}
	
	// Send to all active connections for this user
	for channel := range userClients {
		select {
		case channel <- jsonData:
			// Message sent successfully
		case <-time.After(5 * time.Second):
			// Timeout - client might be disconnected
			s.logger.Warn("Timeout sending message to user:", userID)
		}
	}
}

// BroadcastToUser broadcasts a generic message to a specific user
func (s *SSEManager) BroadcastToUser(userID string, eventType string, data interface{}) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	
	userClients, exists := s.clients[userID]
	if !exists {
		return // No active connections for this user
	}
	
	// Prepare the event data
	event := map[string]interface{}{
		"type": eventType,
		"data": data,
		"time": time.Now().Unix(),
	}
	
	jsonData, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("Failed to marshal broadcast event:", err)
		return
	}
	
	// Send to all active connections for this user
	for channel := range userClients {
		select {
		case channel <- jsonData:
			// Message sent successfully
		case <-time.After(5 * time.Second):
			// Timeout - client might be disconnected
			s.logger.Warn("Timeout sending broadcast to user:", userID)
		}
	}
}

// broadcastEvents handles the global broadcast channel
func (s *SSEManager) broadcastEvents() {
	for {
		select {
		case <-s.broadcast:
			// This handles global broadcasts, though for user-specific notifications
			// we use the direct user broadcast methods
			s.logger.Info("Global broadcast event received")
		case <-s.ctx.Done():
			// Context cancelled, exit the broadcaster
			return
		}
	}
}

// Close shuts down the SSE manager
func (s *SSEManager) Close() {
	s.cancel()
	
	// Close all client channels
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	
	for userID, userClients := range s.clients {
		for channel := range userClients {
			close(channel)
		}
		delete(s.clients, userID)
	}
}

// GetUserConnectionCount returns the number of active connections for a user
func (s *SSEManager) GetUserConnectionCount(userID string) int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	
	userClients, exists := s.clients[userID]
	if !exists {
		return 0
	}
	
	return len(userClients)
}

// HasUserConnection checks if a user has active SSE connections
func (s *SSEManager) HasUserConnection(userID string) bool {
	return s.GetUserConnectionCount(userID) > 0
}