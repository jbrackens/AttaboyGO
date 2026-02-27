package infra

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
)

// WSHub manages WebSocket connections and room-based message delivery.
// In production, backed by Redis pub/sub for multi-instance support.
type WSHub struct {
	mu       sync.RWMutex
	rooms    map[string]map[string]*WSConn // room -> connID -> conn
	logger   *slog.Logger
}

// WSConn represents a WebSocket connection (abstracted for testability).
type WSConn struct {
	ID       string
	PlayerID string
	Send     chan []byte
}

// WSMessage is the payload sent over WebSocket.
type WSMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// NewWSHub creates a new WebSocket hub.
func NewWSHub(logger *slog.Logger) *WSHub {
	return &WSHub{
		rooms:  make(map[string]map[string]*WSConn),
		logger: logger,
	}
}

// Join adds a connection to a room (typically player-scoped: "player:{id}").
func (h *WSHub) Join(room string, conn *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[string]*WSConn)
	}
	h.rooms[room][conn.ID] = conn
}

// Leave removes a connection from a room.
func (h *WSHub) Leave(room string, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.rooms[room]; ok {
		delete(conns, connID)
		if len(conns) == 0 {
			delete(h.rooms, room)
		}
	}
}

// Publish sends a message to all connections in a room.
func (h *WSHub) Publish(room string, event string, data interface{}) {
	msg := WSMessage{Event: event, Data: data}
	payload, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("ws marshal error", "error", err, "room", room, "event", event)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.rooms[room]
	if !ok {
		return
	}

	for _, conn := range conns {
		select {
		case conn.Send <- payload:
		default:
			h.logger.Warn("ws send buffer full", "connID", conn.ID, "room", room)
		}
	}
}

// PublishToPlayer is a convenience method to publish to a player-scoped room.
func (h *WSHub) PublishToPlayer(playerID string, event string, data interface{}) {
	h.Publish("player:"+playerID, event, data)
}

// ConnectionCount returns the total number of active connections.
func (h *WSHub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, conns := range h.rooms {
		count += len(conns)
	}
	return count
}

// RoomCount returns the number of active rooms.
func (h *WSHub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// Shutdown closes all connections gracefully.
func (h *WSHub) Shutdown(_ context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for room, conns := range h.rooms {
		for _, conn := range conns {
			close(conn.Send)
		}
		delete(h.rooms, room)
	}
}

// Placeholder for upgrade handler — gorilla/websocket integration deferred to G9 wiring.
var _ http.Handler // ensure net/http imported
