package wsbroadcastgrp

import (
	"context"
	"log"
	"net/http"

	"github.com/ardanlabs/service/business/ws"
	"github.com/ardanlabs/service/business/ws/schema"
	"github.com/google/uuid"
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
	sessionID := uuid.NewString()
	ws.Sessions.Store(sessionID, &conn)

	go ws.ListenForWS(&conn)

	go ws.ListenToWsChannel()
	return nil
}
