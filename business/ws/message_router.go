package ws

import (
	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type LocalMessageRouter struct {
	protojsonMarshaler *protojson.MarshalOptions
	sessionRegistry    *LocalSessionRegistry
}

func NewLocalMessageRouter(sessionRegistry *LocalSessionRegistry, protojsonMarshaler *protojson.MarshalOptions) *LocalMessageRouter {
	return &LocalMessageRouter{
		protojsonMarshaler: protojsonMarshaler,
		sessionRegistry:    sessionRegistry,
	}
}

func (r *LocalMessageRouter) SendToPresenceIDs(logger *zap.Logger, sessionIDs []uuid.UUID, envelope *rtapi.Envelope) {
	if len(sessionIDs) == 0 {
		return
	}

	// Prepare payload variables but do not initialize until we hit a session that needs them to avoid unnecessary work.
	var payloadProtobuf []byte
	var payloadJSON []byte

	for _, presenceID := range sessionIDs {
		session := r.sessionRegistry.Get(presenceID)
		if session == nil {
			logger.Debug("No session to route to", zap.String("sid", presenceID.String()))
			continue
		}

		var err error
		switch session.Format() {
		case SessionFormatProtobuf:
			if payloadProtobuf == nil {
				// Marshal the payload now that we know this format is needed.
				payloadProtobuf, err = proto.Marshal(envelope)
				if err != nil {
					logger.Error("Could not marshal message", zap.Error(err))
					return
				}
			}
			err = session.SendBytes(payloadProtobuf)
		case SessionFormatJson:
			fallthrough
		default:
			if payloadJSON == nil {
				// Marshal the payload now that we know this format is needed.
				if buf, err := r.protojsonMarshaler.Marshal(envelope); err == nil {
					payloadJSON = buf
				} else {
					logger.Error("Could not marshal message", zap.Error(err))
					return
				}
			}
			err = session.SendBytes(payloadJSON)
		}
		if err != nil {
			logger.Info("Failed to route message", zap.String("sid", presenceID.String()), zap.Error(err))
		}
	}
}
