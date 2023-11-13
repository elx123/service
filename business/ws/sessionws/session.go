package sessionws

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"github.com/ardanlabs/conf/v2"
)

type SessionWS struct {
	sync.Mutex
	logger *zap.Logger
	//config     Config
	id uuid.UUID
	//format     SessionFormat
	userID     uuid.UUID
	username   *atomic.String
	vars       map[string]string
	expiry     int64
	clientIP   string
	clientPort string
	lang       string

	ctx         context.Context
	ctxCancelFn context.CancelFunc

	//protojsonMarshaler   *protojson.MarshalOptions
	//protojsonUnmarshaler *protojson.UnmarshalOptions
	wsMessageType      int
	pingPeriodDuration time.Duration
	pongWaitDuration   time.Duration
	writeWaitDuration  time.Duration

	sessionRegistry *LocalSessionRegistry
	//statusRegistry  *StatusRegistry
	//matchmaker      Matchmaker
	//tracker         Tracker
	//metrics         Metrics
	//pipeline        *Pipeline
	//runtime         *Runtime

	stopped bool
	conn    *websocket.Conn
	//receivedMessageCounter int
	pingTimer    *time.Timer
	pingTimerCAS *atomic.Uint32
	outgoingCh   chan []byte
}

func NewSessionWS(logger *zap.Logger, format SessionFormat, sessionID, userID uuid.UUID, username string, vars map[string]string, expiry int64, clientIP, clientPort, lang string, conn *websocket.Conn, sessionRegistry *LocalSessionRegistry) *SessionWS {
	sessionLogger := logger.With(zap.String("uid", userID.String()), zap.String("sid", sessionID.String()))

	sessionLogger.Info("New WebSocket session connected", zap.Uint8("format", uint8(format)))

	ctx, ctxCancelFn := context.WithCancel(context.Background())

	wsMessageType := websocket.TextMessage
	if format == SessionFormatProtobuf {
		wsMessageType = websocket.BinaryMessage
	}

	return &SessionWS{
		logger:     sessionLogger,
		id:         sessionID,
		userID:     userID,
		username:   atomic.NewString(username),
		vars:       vars,
		expiry:     expiry,
		clientIP:   clientIP,
		clientPort: clientPort,
		lang:       lang,

		ctx:         ctx,
		ctxCancelFn: ctxCancelFn,

		wsMessageType:      wsMessageType,
		pingPeriodDuration: time.Duration(asdfsfasdf) * time.Millisecond,
		pongWaitDuration:   time.Duration(config.GetSocket().PongWaitMs) * time.Millisecond,
		writeWaitDuration:  time.Duration(config.GetSocket().WriteWaitMs) * time.Millisecond,

		sessionRegistry: sessionRegistry,
		//statusRegistry:  statusRegistry,
		//matchmaker:      matchmaker,
		//tracker:         tracker,
		//metrics:         metrics,
		//pipeline:        pipeline,
		//runtime:         runtime,

		stopped: false,
		conn:    conn,
		//receivedMessageCounter: config.GetSocket().PingBackoffThreshold,
		pingTimer:    time.NewTimer() * time.Millisecond),
		pingTimerCAS: atomic.NewUint32(1),
		outgoingCh:   make(chan []byte, config.GetSocket().OutgoingQueueSize),
	}
}

func (s *SessionWS) processOutgoing() {

OutgoingLoop:
	for {
		select {
		case <-s.ctx.Done():
			// Session is closing, close the outgoing process routine.
			break OutgoingLoop
		case <-s.pingTimer.C:
			// Periodically send pings.
			if _, ok := s.pingNow(); !ok {
				// If ping fails the session will be stopped, clean up the loop.
				break OutgoingLoop
			}
		case payload := <-s.outgoingCh:
			s.Lock()
			if s.stopped {
				// The connection may have stopped between the payload being queued on the outgoing channel and reaching here.
				// If that's the case then abort outgoing processing at this point and exit.
				s.Unlock()
				break OutgoingLoop
			}
			// Process the outgoing message queue.
			if err := s.conn.SetWriteDeadline(time.Now().Add(s.writeWaitDuration)); err != nil {
				s.Unlock()
				s.logger.Warn("Failed to set write deadline", zap.Error(err))

				break OutgoingLoop
			}
			if err := s.conn.WriteMessage(s.wsMessageType, payload); err != nil {
				s.Unlock()
				s.logger.Warn("Could not write message", zap.Error(err))

				break OutgoingLoop
			}
			s.Unlock()

		}
	}

	s.Close()
}

func (s *SessionWS) pingNow() (string, bool) {
	s.Lock()
	if s.stopped {
		s.Unlock()
		return "", false
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(s.writeWaitDuration)); err != nil {
		s.Unlock()
		s.logger.Warn("Could not set write deadline to ping", zap.Error(err))
		return err.Error(), false
	}
	err := s.conn.WriteMessage(websocket.PingMessage, []byte{})
	s.Unlock()
	if err != nil {
		s.logger.Warn("Could not send ping", zap.Error(err))
		return err.Error(), false
	}

	return "", true
}

func (s *SessionWS) Close() {
	s.Lock()
	if s.stopped {
		s.Unlock()
		return
	}
	s.stopped = true
	s.Unlock()

	// Cancel any ongoing operations tied to this session.
	s.ctxCancelFn()

	s.sessionRegistry.Remove(s.id)
	if s.logger.Core().Enabled(zap.DebugLevel) {
		s.logger.Info("Cleaned up closed connection session registry")
	}

	// Clean up internals.
	s.pingTimer.Stop()
	close(s.outgoingCh)

	// Send close message.
	if err := s.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(s.writeWaitDuration)); err != nil {
		// This may not be possible if the socket was already fully closed by an error.
		s.logger.Debug("Could not send close message", zap.Error(err))
	}
	// Close WebSocket.
	if err := s.conn.Close(); err != nil {
		s.logger.Debug("Could not close", zap.Error(err))
	}

	s.logger.Info("Closed client connection")
}

func (s *sessionWS) Consume() {
	// Fire an event for session start.
	if fn := s.runtime.EventSessionStart(); fn != nil {
		fn(s.userID.String(), s.username.Load(), s.vars, s.expiry, s.id.String(), s.clientIP, s.clientPort, s.lang, time.Now().UTC().Unix())
	}

	s.conn.SetReadLimit(s.config.GetSocket().MaxMessageSizeBytes)
	if err := s.conn.SetReadDeadline(time.Now().Add(s.pongWaitDuration)); err != nil {
		s.logger.Warn("Failed to set initial read deadline", zap.Error(err))
		s.Close("failed to set initial read deadline", runtime.PresenceReasonDisconnect)
		return
	}
	// 这里我猜测，作为全双工协议，我们也需要设置对应的handler
	s.conn.SetPongHandler(func(string) error {
		// 在接受到pong的 message以后，重置ping pong定时器
		s.maybeResetPingTimer()
		return nil
	})

	// Start a routine to process outbound messages.
	go s.processOutgoing()

	var reason string
	var data []byte

IncomingLoop:
	for {
		messageType, data, err := s.conn.ReadMessage()
		if err != nil {
			// Ignore "normal" WebSocket errors.
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				// Ignore underlying connection being shut down while read is waiting for data.
				// 这里比较字符串 应该是历史遗留问题了
				// https://github.com/golang/go/issues/4373
				if e, ok := err.(*net.OpError); !ok || e.Err.Error() != "use of closed network connection" {
					s.logger.Debug("Error reading message from client", zap.Error(err))
					reason = err.Error()
				}
			}
			break
		}
		if messageType != s.wsMessageType {
			// Expected text but received binary, or expected binary but received text.
			// Disconnect client if it attempts to use this kind of mixed protocol mode.
			s.logger.Debug("Received unexpected WebSocket message type", zap.Int("expected", s.wsMessageType), zap.Int("actual", messageType))
			reason = "received unexpected WebSocket message type"
			break
		}

		s.receivedMessageCounter--
		if s.receivedMessageCounter <= 0 {
			s.receivedMessageCounter = s.config.GetSocket().PingBackoffThreshold
			if !s.maybeResetPingTimer() {
				// Problems resetting the ping timer indicate an error so we need to close the loop.
				reason = "error updating ping timer"
				break
			}
		}

		request := &rtapi.Envelope{}
		switch s.format {
		case SessionFormatProtobuf:
			err = proto.Unmarshal(data, request)
		case SessionFormatJson:
			fallthrough
		default:
			err = s.protojsonUnmarshaler.Unmarshal(data, request)
		}
		if err != nil {
			// If the payload is malformed the client is incompatible or misbehaving, either way disconnect it now.
			s.logger.Warn("Received malformed payload", zap.Binary("data", data))
			reason = "received malformed payload"
			break
		}

		switch request.Cid {
		case "":
			if !s.pipeline.ProcessRequest(s.logger, s, request) {
				reason = "error processing message"
				break IncomingLoop
			}
		default:
			requestLogger := s.logger.With(zap.String("cid", request.Cid))
			if !s.pipeline.ProcessRequest(requestLogger, s, request) {
				reason = "error processing message"
				break IncomingLoop
			}
		}

		// Update incoming message metrics.
		s.metrics.Message(int64(len(data)), false)
	}

	if reason != "" {
		// Update incoming message metrics.
		s.metrics.Message(int64(len(data)), true)
	}

	s.Close(reason, runtime.PresenceReasonDisconnect)
}