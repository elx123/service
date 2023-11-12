package ws

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/ardanlabs/service/business/ws/schema"
	"github.com/gorilla/websocket"
)

var Sessions sync.Map

// var Clients = make(map[schema.WebSocketConnection]string)
var WsChan = make(chan schema.WsPayload)

// upgradeConnection is the upgraded connection
var UpgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func NotifyClientMatchmakerResult(players []string, ip string, port int) error {
	var response schema.WsJsonResponse
	response.Action = "Notify"
	response.IP = ip
	response.Port = port

	sessions := make([]*schema.WebSocketConnection, 2)
	for _, v := range players {
		session, ok := Sessions.Load(v)
		if !ok {
			return fmt.Errorf("the session of player:%v not exist", v)
		}
		sessionConn, _ := session.(*schema.WebSocketConnection)
		sessions = append(sessions, sessionConn)
	}

	for k := range sessions {
		err := sessions[k].WriteJSON(response)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListenForWS is the goroutine that listens for our channels
func ListenForWS(conn *schema.WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	var payload schema.WsPayload

	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			// do nothing
		} else {
			payload.Conn = *conn
			WsChan <- payload
		}
	}
}

func ListenToWsChannel() {

	var response schema.WsJsonResponse
	for {
		e := <-WsChan

		switch e.Action {
		case "ping":
			response.Action = "pong"
			//response.SkipSender = false
			//response.Message = fmt.Sprintf("<strong>%s:</strong> %s", e.UserName, e.Message)
			e.Conn.WriteJSON(response)
		}
	}
}
