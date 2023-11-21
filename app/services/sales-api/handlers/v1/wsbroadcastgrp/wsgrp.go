package wsbroadcastgrp

import (
	"context"
	"errors"
	"net/http"

	"github.com/ardanlabs/service/business/config"
	"github.com/ardanlabs/service/business/sys/auth"
	"github.com/ardanlabs/service/business/sys/validate"
	"github.com/ardanlabs/service/business/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

type Handlers struct {
	Config               *config.Config
	Auth                 *auth.Auth
	logger               *zap.Logger
	Pipeline             *ws.Pipeline
	MessageRouter        *ws.LocalMessageRouter
	SessionRegistry      *ws.LocalSessionRegistry
	ProtojsonMarshaler   *protojson.MarshalOptions
	ProtojsonUnmarshaler *protojson.UnmarshalOptions
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
		//http.Error(w, "Invalid format parameter", 400)
		return validate.NewRequestError(errors.New("invalid format parameter"), http.StatusBadRequest)
	}

	claim, err := auth.GetClaims(ctx)
	if err != nil {
		return errors.New("claims missing from context")
	}

	// Upgrade to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// http.Error is invoked automatically from within the Upgrade function.
		h.logger.Error("Could not upgrade to WebSocket", zap.Error(err))
		return nil
	}

	//clientIP, clientPort := extractClientAddressFromRequest(logger, r)
	//status, _ := strconv.ParseBool(r.URL.Query().Get("status"))
	sessionID := uuid.New()

	userid, err := uuid.Parse(claim.UserId)
	if err != nil {
		h.logger.Error("userid Parse fail", zap.String("userid", claim.UserId), zap.Error(err))
		return err
	}

	// Wrap the connection for application handling.
	session := ws.NewSessionWS(h.logger, h.Config, format, sessionID, userid, claim.Username, conn, h.SessionRegistry, h.ProtojsonMarshaler, h.ProtojsonUnmarshaler, h.Pipeline)

	// Add to the session registry.
	h.SessionRegistry.Add(session)

	if h.Config.GetSession().SingleSocket {
		// Kick any other sockets for this user.
		go h.SessionRegistry.SingleSession(ctx, userid, sessionID)
	}

	session.Consume()

	return nil
}
