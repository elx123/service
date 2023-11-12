package wsbroadcastgrp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[WebSocketConnection]string)
var wsChan = make(chan WsPayload)

type WebSocketConnection struct {
	*websocket.Conn
}

// upgradeConnection is the upgraded connection
var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WsJsonResponse defines the json we send back to client
type WsJsonResponse struct {
	Action         string              `json:"action"`
	Message        string              `json:"message"`
	MessageType    string              `json:"message_type"`
	SkipSender     bool                `json:"-"`
	CurrentConn    WebSocketConnection `json:"-"`
	ConnectedUsers []string            `json:"connected_users"`
}

// WsPayload defines the data we receive from the client
type WsPayload struct {
	Action      string              `json:"action"`
	Message     string              `json:"message"`
	UserName    string              `json:"username"`
	MessageType string              `json:"message_type"`
	Conn        WebSocketConnection `json:"-"`
}

type Handlers struct {
}

func (h Handlers) WsEndPoint(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	//log.Println(fmt.Sprintf("Client Connected from %s", r.RemoteAddr))
	var response WsJsonResponse
	response.Message = "<em><small>Connected to server ... </small></em>"

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
		return err
	}

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	go ListenForWS(&conn)
	return nil
}

// ListenForWS is the goroutine that listens for our channels
func ListenForWS(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload

	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			// do nothing
		} else {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}
