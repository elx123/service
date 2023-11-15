package wsbroadcastgrp

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardanlabs/service/business/config"
	"github.com/ardanlabs/service/business/ws"
	"github.com/ardanlabs/service/business/ws/schema"
	"github.com/ardanlabs/service/business/ws/sessionws"
	"github.com/ardanlabs/service/business/ws/sessionregistry"
	"github.com/ardanlabs/service/business/sys/auth"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)



type Handlers struct {
	SessionRegistry *sessionws.LocalSessionRegistry
	Config          *config.Config
	Auth            *auth.Auth
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

func (h Handlers) NewSocketWsAcceptor(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  h.Config.GetSocket().ReadBufferSizeBytes,
		WriteBufferSize: h.Config.GetSocket().WriteBufferSizeBytes,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// Check format.
	var format SessionFormat
	switch r.URL.Query().Get("format") {
	case "protobuf":
		format = SessionFormatProtobuf
	case "json":
		fallthrough
	case "":
		format = SessionFormatJson
	default:
		// Invalid values are rejected.
		http.Error(w, "Invalid format parameter", 400)
		return
	}

	claim,err := auth.GetClaims(ctx)
	if err!= nil{
		return fmt.Errorf("err: %w", err)
	}

	// Upgrade to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// http.Error is invoked automatically from within the Upgrade function.
		logger.Debug("Could not upgrade to WebSocket", zap.Error(err))
		return
	}

	//clientIP, clientPort := extractClientAddressFromRequest(logger, r)
	//status, _ := strconv.ParseBool(r.URL.Query().Get("status"))
	sessionID := uuid.NewString()

	// Wrap the connection for application handling.
	session := sessionws.NewSessionWS(logger, h.Config, format, sessionID, claim.UserId, claim.Username, claim.ExpiresAt,conn,h.SessionRegistry)

	// Add to the session registry.
	h.SessionRegistry.Add(session)


	// Register initial status tracking and presence(s) for this session.
	statusRegistry.Follow(sessionID, map[uuid.UUID]struct{}{userID: {}})
	if status {
		// Both notification and status presence.
		tracker.TrackMulti(session.Context(), sessionID, []*TrackerOp{
				{
					Stream: PresenceStream{Mode: StreamModeNotifications, Subject: userID},
					Meta:   PresenceMeta{Format: format, Username: username, Hidden: true},
				},
				{
					Stream: PresenceStream{Mode: StreamModeStatus, Subject: userID},
					Meta:   PresenceMeta{Format: format, Username: username, Status: ""},
				},
			}, userID, true)
		} else {
			// Only notification presence.
			tracker.Track(session.Context(), sessionID, PresenceStream{Mode: StreamModeNotifications, Subject: userID}, userID, PresenceMeta{Format: format, Username: username, Hidden: true}, true)
		}

		if config.GetSession().SingleSocket {
			// Kick any other sockets for this user.
			go sessionRegistry.SingleSession(session.Context(), tracker, userID, sessionID)
		}

		// Allow the server to begin processing incoming messages from this session.
		session.Consume()

		// Mark the end of the session.
		metrics.CountWebsocketClosed(1)
	}
}
