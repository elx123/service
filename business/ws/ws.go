package ws

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/ardanlabs/service/business/ws/schema"
	"github.com/gorilla/websocket"
)

var Clients = make(map[schema.WebSocketConnection]string)
var WsChan = make(chan schema.WsPayload)

// upgradeConnection is the upgraded connection
var UpgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func ListenToWsChannel() {
	var response schema.WsJsonResponse
	for {
		e := <-WsChan

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

			delete(Clients, e.Conn)
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
	for _, value := range Clients {
		if value != "" {
			userNames = append(userNames, value)
		}
	}
	sort.Strings(userNames)

	return userNames
}

func addToUserList(conn schema.WebSocketConnection, u string) []string {
	var userNames []string
	Clients[conn] = u
	for _, value := range Clients {
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
func broadcastToAll(response schema.WsJsonResponse) {
	for client := range Clients {
		// skip sender, if appropriate
		if response.SkipSender && response.CurrentConn == client {
			continue
		}

		// broadcast to every connected client
		err := client.WriteJSON(response)
		if err != nil {
			log.Printf("Websocket error on %s: %s", response.Action, err)
			_ = client.Close()
			delete(Clients, client)
		}
	}
}
