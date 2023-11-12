package schema

import (
	"github.com/gorilla/websocket"
)

type WebSocketConnection struct {
	*websocket.Conn
}

// WsJsonResponse defines the json we send back to client
type WsJsonResponse struct {
	Action         string              `json:"action"`
	Message        string              `json:"message"`
	MessageType    string              `json:"message_type"`
	SkipSender     bool                `json:"-"`
	CurrentConn    WebSocketConnection `json:"-"`
	ConnectedUsers []string            `json:"connected_users"`
	IP             string              `json:"IP"`
	Port           int                 `json:"Port"`
}

// WsPayload defines the data we receive from the client
type WsPayload struct {
	Action      string              `json:"action"`
	Message     string              `json:"message"`
	UserName    string              `json:"username"`
	MessageType string              `json:"message_type"`
	Conn        WebSocketConnection `json:"-"`
}
