package wsbroadcastgrp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ardanlabs/service/business/config"
	"github.com/ardanlabs/service/business/sys/auth"
	"github.com/ardanlabs/service/business/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Handlers struct {
	SessionRegistry *ws.LocalSessionRegistry
	Config          *config.Config
	Auth            *auth.Auth
	logger          *zap.Logger
}

func (h Handlers) NewSocketWsAcceptor(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  h.Config.GetSocket().ReadBufferSizeBytes,
		WriteBufferSize: h.Config.GetSocket().WriteBufferSizeBytes,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// Check format.
	var format ws.SessionFormat
	switch r.URL.Query().Get("format") {
	case "protobuf":
		format = ws.SessionFormatProtobuf
	case "json":
		fallthrough
	case "":
		format = ws.SessionFormatJson
	default:
		// Invalid values are rejected.
		http.Error(w, "Invalid format parameter", 400)
		return errors.New("Invalid format parameter")
	}

	claim, err := auth.GetClaims(ctx)
	if err != nil {
		return fmt.Errorf("err: %w", err)
	}

	// Upgrade to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// http.Error is invoked automatically from within the Upgrade function.
		h.logger.Error("Could not upgrade to WebSocket", zap.Error(err))
		return errors.New("Could not upgrade to WebSocket")
	}

	//clientIP, clientPort := extractClientAddressFromRequest(logger, r)
	//status, _ := strconv.ParseBool(r.URL.Query().Get("status"))
	sessionID := uuid.NewString()

	// Wrap the connection for application handling.
	session := ws.NewSessionWS(h.logger, h.Config, format, sessionID, claim.UserId, claim.Username, claim.ExpiresAt, conn, h.SessionRegistry)

	// Add to the session registry.
	h.SessionRegistry.Add(session)

	if status {

		if config.GetSession().SingleSocket {
			// Kick any other sockets for this user.
			go sessionRegistry.SingleSession(session.Context(), tracker, userID, sessionID)
		}

		// Allow the server to begin processing incoming messages from this session.
		session.Consume()
	}
	return nil
}
