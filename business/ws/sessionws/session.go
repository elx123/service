package sessionws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ardanlabs/service/business/config"
	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ErrSessionQueueFull = errors.New("session outgoing queue full")

type SessionFormat uint8

const (
	SessionFormatJson SessionFormat = iota
	SessionFormatProtobuf
)

type SessionWS struct {
	sync.Mutex
	logger   *zap.Logger
	config   *config.Config
	id       uuid.UUID
	format   SessionFormat
	userID   uuid.UUID
	username *atomic.String
	//vars       map[string]string
	expiry int64
	//clientIP   string
	//clientPort string
	lang string

	ctx         context.Context
	ctxCancelFn context.CancelFunc

	protojsonMarshaler   *protojson.MarshalOptions
	protojsonUnmarshaler *protojson.UnmarshalOptions
	wsMessageType        int
	pingPeriodDuration   time.Duration
	pongWaitDuration     time.Duration
	writeWaitDuration    time.Duration

	sessionRegistry *LocalSessionRegistry
	//statusRegistry  *StatusRegistry
	//matchmaker      Matchmaker
	//tracker         Tracker
	//metrics         Metrics
	//pipeline        *Pipeline
	//runtime         *Runtime

	stopped                bool
	conn                   *websocket.Conn
	receivedMessageCounter int
	pingTimer              *time.Timer
	pingTimerCAS           *atomic.Uint32
	outgoingCh             chan []byte
}

func NewSessionWS(logger *zap.Logger, config *config.Config, format SessionFormat, sessionID, userID uuid.UUID, username string, expiry int64, conn *websocket.Conn, sessionRegistry *LocalSessionRegistry, protojsonMarshaler *protojson.MarshalOptions, protojsonUnmarshaler *protojson.UnmarshalOptions) *SessionWS {
	sessionLogger := logger.With(zap.String("uid", userID.String()), zap.String("sid", sessionID.String()))

	sessionLogger.Info("New WebSocket session connected", zap.Uint8("format", uint8(format)))

	ctx, ctxCancelFn := context.WithCancel(context.Background())

	wsMessageType := websocket.TextMessage
	if format == SessionFormatProtobuf {
		wsMessageType = websocket.BinaryMessage
	}

	return &SessionWS{
		logger:   sessionLogger,
		id:       sessionID,
		config:   config,
		format:   format,
		userID:   userID,
		username: atomic.NewString(username),
		expiry:   expiry,

		ctx:         ctx,
		ctxCancelFn: ctxCancelFn,

		protojsonMarshaler:   protojsonMarshaler,
		protojsonUnmarshaler: protojsonUnmarshaler,
		wsMessageType:        wsMessageType,
		pingPeriodDuration:   time.Duration(config.GetSocket().PingPeriodMs) * time.Millisecond,
		pongWaitDuration:     time.Duration(config.GetSocket().PongWaitMs) * time.Millisecond,
		writeWaitDuration:    time.Duration(config.GetSocket().WriteWaitMs) * time.Millisecond,

		sessionRegistry: sessionRegistry,
		//statusRegistry:  statusRegistry,
		//matchmaker:      matchmaker,
		//tracker:         tracker,
		//metrics:         metrics,
		//pipeline:        pipeline,
		//runtime:         runtime,

		stopped:                false,
		conn:                   conn,
		receivedMessageCounter: config.GetSocket().PingBackoffThreshold,
		pingTimer:              time.NewTimer(time.Duration(config.GetSocket().PingPeriodMs) * time.Millisecond),
		pingTimerCAS:           atomic.NewUint32(1),
		outgoingCh:             make(chan []byte, config.GetSocket().OutgoingQueueSize),
	}
}

func (s *SessionWS) ID() uuid.UUID {
	return s.id
}

func (s *SessionWS) Format() SessionFormat {
	return s.format
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
			if msg, ok := s.pingNow(); !ok {
				// If ping fails the session will be stopped, clean up the loop.
				//reason = msg
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
				reason = err.Error()
				break OutgoingLoop
			}
			if err := s.conn.WriteMessage(s.wsMessageType, payload); err != nil {
				s.Unlock()
				s.logger.Warn("Could not write message", zap.Error(err))
				reason = err.Error()
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

func (s *SessionWS) maybeResetPingTimer() bool {
	// If there's already a reset in progress there's no need to wait.
	if !s.pingTimerCAS.CompareAndSwap(1, 0) {
		return true
	}
	defer s.pingTimerCAS.CompareAndSwap(0, 1)

	s.Lock()
	if s.stopped {
		s.Unlock()
		return false
	}
	// CAS ensures concurrency is not a problem here.
	if !s.pingTimer.Stop() {
		select {
		case <-s.pingTimer.C:
		default:
		}
	}
	s.pingTimer.Reset(s.pingPeriodDuration)
	err := s.conn.SetReadDeadline(time.Now().Add(s.pongWaitDuration))
	s.Unlock()
	if err != nil {
		s.logger.Warn("Failed to set read deadline", zap.Error(err))
		s.Close()
		return false
	}
	return true
}

func (s *SessionWS) Close(envelopes ...*rtapi.Envelope) {
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

	// Clean up internals.
	s.pingTimer.Stop()
	close(s.outgoingCh)

	// Send final messages, if any are specified.
	for _, envelope := range envelopes {
		var payload []byte
		var err error
		switch s.format {
		case SessionFormatProtobuf:
			payload, err = proto.Marshal(envelope)
		case SessionFormatJson:
			fallthrough
		default:
			if buf, err := s.protojsonMarshaler.Marshal(envelope); err == nil {
				payload = buf
			}
		}
		if err != nil {
			s.logger.Info("Could not marshal envelope", zap.Error(err))
			continue
		}

		switch envelope.Message.(type) {
		case *rtapi.Envelope_Error:
			s.logger.Info("Sending error message", zap.Binary("payload", payload))
		default:
			s.logger.Info(fmt.Sprintf("Sending %T message", envelope.Message), zap.Any("envelope", envelope))
		}

		s.Lock()
		if err := s.conn.SetWriteDeadline(time.Now().Add(s.writeWaitDuration)); err != nil {
			s.Unlock()
			s.logger.Info("Failed to set write deadline", zap.Error(err))
			continue
		}
		if err := s.conn.WriteMessage(s.wsMessageType, payload); err != nil {
			s.Unlock()
			s.logger.Info("Could not write message", zap.Error(err))
			continue
		}
		s.Unlock()
	}

	// Send close message.
	if err := s.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(s.writeWaitDuration)); err != nil {
		// This may not be possible if the socket was already fully closed by an error.
		s.logger.Info("Could not send close message", zap.Error(err))
	}
	// Close WebSocket.
	if err := s.conn.Close(); err != nil {
		s.logger.Info("Could not close", zap.Error(err))
	}

	s.logger.Info("Closed client connection")
}

func (s *SessionWS) Consume() {

	s.conn.SetReadLimit(s.config.GetSocket().MaxMessageSizeBytes)
	if err := s.conn.SetReadDeadline(time.Now().Add(s.pongWaitDuration)); err != nil {
		s.logger.Info("Failed to set initial read deadline", zap.Error(err))
		s.Close()
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
			s.logger.Info("Received malformed payload", zap.Binary("data", data))
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

	s.Close()
}

//在网络编程和游戏服务器的上下文中，忽略发送消息时的错误并直接丢弃消息在某些情况下是可接受的，尤其是在以下情况：

//非关键性消息：如果消息不是业务逻辑的关键部分，比如一些状态更新或非关键的通知，丢弃这些消息可能不会对游戏的核心体验产生显著影响。

//高频率消息：在高频率通信中，例如实时多人游戏，丢失个别消息可能是可接受的，特别是当下一次更新能够快速补充丢失信息时。

//性能考虑：处理每个错误并尝试重新发送可能会对性能产生负面影响，特别是在高负载或高并发的情况下。

//容错性设计：系统可能设计为对丢失的消息具有容错性，例如通过后续的消息来纠正或更新状态。

// 然而，这种处理方式可能不适用于所有类型的消息。对于某些关键性的业务逻辑，确保消息的可靠传递可能更为重要。因此，这是否是一个好的做法取决于具体的应用场景和消息的重要性。
func (s *SessionWS) Send(envelope *rtapi.Envelope, reliable bool) error {
	var payload []byte
	var err error
	switch s.format {
	case SessionFormatProtobuf:
		payload, err = proto.Marshal(envelope)
	case SessionFormatJson:
		fallthrough
	default:
		if buf, err := s.protojsonMarshaler.Marshal(envelope); err == nil {
			payload = buf
		}
	}
	if err != nil {
		s.logger.Warn("Could not marshal envelope", zap.Error(err))
		return err
	}

	switch envelope.Message.(type) {
	case *rtapi.Envelope_Error:
		s.logger.Info("Sending error message", zap.Binary("payload", payload))
	default:
		s.logger.Info(fmt.Sprintf("Sending %T message", envelope.Message), zap.Any("envelope", envelope))
	}

	return s.SendBytes(payload, reliable)
}

func (s *SessionWS) SendBytes(payload []byte, reliable bool) error {
	s.Lock()
	if s.stopped {
		s.Unlock()
		return nil
	}

	// Attempt to queue messages and observe failures.
	select {
	case s.outgoingCh <- payload:
		s.Unlock()
		return nil
	default:
		// The outgoing queue is full, likely because the remote client can't keep up.
		// Terminate the connection immediately because the only alternative that doesn't block the server is
		// to start dropping messages, which might cause unexpected behaviour.
		s.Unlock()
		s.logger.Warn("Could not write message, session outgoing queue full")
		s.Close(ErrSessionQueueFull.Error())
		return ErrSessionQueueFull
	}
}
