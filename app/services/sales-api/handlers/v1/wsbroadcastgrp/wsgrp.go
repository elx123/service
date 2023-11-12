package wsbroadcastgrp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ardanlabs/service/business/ws"
	"github.com/ardanlabs/service/business/ws/schema"
)

type Handlers struct {
}

func (h Handlers) WsEndPoint(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	wsConn, err := ws.UpgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	//log.Println(fmt.Sprintf("Client Connected from %s", r.RemoteAddr))
	var response schema.WsJsonResponse
	response.Message = "<em><small>Connected to server ... </small></em>"

	err = wsConn.WriteJSON(response)
	if err != nil {
		log.Println(err)
		return err
	}

	conn := schema.WebSocketConnection{Conn: wsConn}
	ws.Clients[conn] = ""

	go ListenForWS(&conn)
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
			ws.WsChan <- payload
		}
	}
}
