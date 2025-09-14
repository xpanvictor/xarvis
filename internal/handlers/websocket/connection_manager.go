package websocket

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// ConnectionManager manages WebSocket connections and sessions
type ConnectionManager struct {
	logger         *Logger.Logger
	sessions       map[uuid.UUID]*Session
	mutex          sync.RWMutex
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	sessionTimeout time.Duration
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(logger *Logger.Logger) *ConnectionManager {
	cm := &ConnectionManager{
		logger:         logger,
		sessions:       make(map[uuid.UUID]*Session),
		stopCleanup:    make(chan struct{}),
		sessionTimeout: 30 * time.Minute, // 30 minutes default timeout
	}

	// Start cleanup goroutine
	cm.startCleanupRoutine()

	return cm
}

// RegisterConnection registers a new session
func (cm *ConnectionManager) RegisterConnection(session *Session) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.sessions[session.UserID] = session
	cm.logger.Infof("Registered session for user %s (session: %s)",
		session.UserID, session.SessionID)
}

// UnregisterConnection removes a session
func (cm *ConnectionManager) UnregisterConnection(userID uuid.UUID) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if session, exists := cm.sessions[userID]; exists {
		cm.logger.Infof("Unregistering session for user %s (session: %s)",
			userID, session.SessionID)

		// Close the session
		if err := session.Close(); err != nil {
			cm.logger.Errorf("Error closing session for user %s: %v", userID, err)
		}

		delete(cm.sessions, userID)
	}
}

// GetSession retrieves a session by user ID
func (cm *ConnectionManager) GetSession(userID uuid.UUID) (*Session, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	session, exists := cm.sessions[userID]
	return session, exists
}

// GetAllSessions returns all active sessions
func (cm *ConnectionManager) GetAllSessions() map[uuid.UUID]*Session {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Create a copy to avoid race conditions
	sessions := make(map[uuid.UUID]*Session)
	for userID, session := range cm.sessions {
		sessions[userID] = session
	}

	return sessions
}

// GetSessionCount returns the number of active sessions
func (cm *ConnectionManager) GetSessionCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.sessions)
}

// BroadcastMessage broadcasts a message to all connected clients
func (cm *ConnectionManager) BroadcastMessage(msgType MessageType, data interface{}) {
	cm.mutex.RLock()
	sessions := make([]*Session, 0, len(cm.sessions))
	for _, session := range cm.sessions {
		sessions = append(sessions, session)
	}
	cm.mutex.RUnlock()

	// Send to all sessions without holding the lock
	for _, session := range sessions {
		if err := session.SendWebSocketMessage(msgType, data); err != nil {
			cm.logger.Errorf("Failed to broadcast message to user %s: %v",
				session.UserID, err)
		}
	}
}

// SendMessageToUser sends a message to a specific user
func (cm *ConnectionManager) SendMessageToUser(userID uuid.UUID, msgType MessageType, data interface{}) error {
	session, exists := cm.GetSession(userID)
	if !exists {
		return fmt.Errorf("no active session for user %s", userID)
	}

	return session.SendWebSocketMessage(msgType, data)
}

// SetSessionTimeout sets the session timeout duration
func (cm *ConnectionManager) SetSessionTimeout(timeout time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.sessionTimeout = timeout
}

// startCleanupRoutine starts a goroutine to clean up expired sessions
func (cm *ConnectionManager) startCleanupRoutine() {
	cm.cleanupTicker = time.NewTicker(5 * time.Minute) // Check every 5 minutes

	go func() {
		for {
			select {
			case <-cm.cleanupTicker.C:
				cm.cleanupExpiredSessions()
			case <-cm.stopCleanup:
				cm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupExpiredSessions removes expired sessions
func (cm *ConnectionManager) cleanupExpiredSessions() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	expiredUsers := make([]uuid.UUID, 0)

	for userID, session := range cm.sessions {
		if session.IsExpired(cm.sessionTimeout) {
			expiredUsers = append(expiredUsers, userID)
		}
	}

	// Remove expired sessions
	for _, userID := range expiredUsers {
		cm.logger.Infof("Cleaning up expired session for user %s", userID)
		if session := cm.sessions[userID]; session != nil {
			session.Close()
		}
		delete(cm.sessions, userID)
	}

	if len(expiredUsers) > 0 {
		cm.logger.Infof("Cleaned up %d expired sessions", len(expiredUsers))
	}
}

// Close shuts down the connection manager
func (cm *ConnectionManager) Close() error {
	// Stop cleanup routine
	close(cm.stopCleanup)

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Close all sessions
	for userID, session := range cm.sessions {
		cm.logger.Infof("Closing session for user %s", userID)
		if err := session.Close(); err != nil {
			cm.logger.Errorf("Error closing session for user %s: %v", userID, err)
		}
	}

	// Clear sessions map
	cm.sessions = make(map[uuid.UUID]*Session)

	cm.logger.Infof("Connection manager closed")
	return nil
}

// GetStats returns connection manager statistics
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := map[string]interface{}{
		"active_sessions": len(cm.sessions),
		"session_timeout": cm.sessionTimeout.String(),
	}

	// Add per-session stats
	sessionStats := make([]map[string]interface{}, 0, len(cm.sessions))
	for _, session := range cm.sessions {
		sessionStats = append(sessionStats, map[string]interface{}{
			"user_id":      session.UserID.String(),
			"session_id":   session.SessionID.String(),
			"connected_at": session.ConnectedAt,
			"last_active":  session.LastActive,
			"is_active":    session.IsActive,
		})
	}
	stats["sessions"] = sessionStats

	return stats
}
