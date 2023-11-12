package user

import (
	"fmt"
	"log"
	"sort"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Core manages the set of API's for user access.
type Core struct {
	log *zap.SugaredLogger
}

// NewCore constructs a core for user api access.
func NewCore(log *zap.SugaredLogger, db *sqlx.DB) Core {
	return Core{
		log: log,
	}
}

// ListenToWsChannel listens to all channels and pushes data to broadcast function
func ListenToWsChannel() {
	var response WsJsonResponse
	for {
		e := <-wsChan

		switch e.Action {
		case "broadcast":
			response.Action = "broadcast"
			response.SkipSender = false
			response.Message = fmt.Sprintf("<strong>%s:</strong> %s", e.UserName, e.Message)
			broadcastToAll(response)

		case "alert":
			response.Action = "alert"
			response.SkipSender = false
			response.Message = e.Message
			response.MessageType = e.MessageType
			broadcastToAll(response)

		case "list_users":
			response.Action = "list"
			response.SkipSender = false
			response.Message = e.Message
			broadcastToAll(response)

		case "connect":
			response.Action = "connected"
			response.SkipSender = false
			response.Message = e.Message
			broadcastToAll(response)

		case "entered":
			response.SkipSender = true
			response.CurrentConn = e.Conn
			response.Action = "entered"
			response.Message = `<small class="text-muted"><em>New user in room</em></small>`
			broadcastToAll(response)

		case "left":
			response.SkipSender = false
			response.CurrentConn = e.Conn
			response.Action = "left"
			response.Message = fmt.Sprintf(`<small class="text-muted"><em>%s left</em></small>`, e.UserName)
			broadcastToAll(response)

			delete(clients, e.Conn)
			userList := getUserNameList()
			response.Action = "list_users"
			response.ConnectedUsers = userList
			response.SkipSender = false
			broadcastToAll(response)

		case "username":
			userList := addToUserList(e.Conn, e.UserName)
			response.Action = "list_users"
			response.ConnectedUsers = userList
			response.SkipSender = false
			broadcastToAll(response)
		}
	}
}

func getUserNameList() []string {
	var userNames []string
	for _, value := range clients {
		if value != "" {
			userNames = append(userNames, value)
		}
	}
	sort.Strings(userNames)

	return userNames
}

func addToUserList(conn WebSocketConnection, u string) []string {
	var userNames []string
	clients[conn] = u
	for _, value := range clients {
		if value != "" {
			userNames = append(userNames, value)
		}
	}
	sort.Strings(userNames)

	return userNames
}

// broadcastToAll sends a response to all connected clients, as JSON
// note that the JSON will show up as part of the WS default json,
// under "data"
func broadcastToAll(response WsJsonResponse) {
	for client := range clients {
		// skip sender, if appropriate
		if response.SkipSender && response.CurrentConn == client {
			continue
		}

		// broadcast to every connected client
		err := client.WriteJSON(response)
		if err != nil {
			log.Printf("Websocket error on %s: %s", response.Action, err)
			_ = client.Close()
			delete(clients, client)
		}
	}
}
