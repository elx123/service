package sessionws

import (
	"database/sql"

	"github.com/ardanlabs/service/business/config"
	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/ardanlabs/service/business/ws/sessionws"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

type Pipeline struct {
	logger               *zap.Logger
	config               *config.Config
	protojsonMarshaler   *protojson.MarshalOptions
	protojsonUnmarshaler *protojson.UnmarshalOptions
	router               *LocalMessageRouter
}

func NewPipeline(logger *zap.Logger, config *config.Config, db *sql.DB, protojsonMarshaler *protojson.MarshalOptions, protojsonUnmarshaler *protojson.UnmarshalOptions, router *LocalMessageRouter) *Pipeline {
	return &Pipeline{
		logger:               logger,
		config:               config,
		protojsonMarshaler:   protojsonMarshaler,
		protojsonUnmarshaler: protojsonUnmarshaler,
		router:               router,
	}
}

func (p *Pipeline) ProcessRequest(logger *zap.Logger, session *sessionws.SessionWS, in *rtapi.Envelope) bool {
	if in.Message == nil {
		session.Send(&rtapi.Envelope{Cid: in.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_MISSING_PAYLOAD),
			Message: "Missing message.",
		}}})
		return false
	}

	var pipelineFn func(*zap.Logger, *sessionws.SessionWS, *rtapi.Envelope) (bool, *rtapi.Envelope)

	switch in.Message.(type) {
	case *rtapi.Envelope_Ping:
		pipelineFn = p.ping
	case *rtapi.Envelope_Pong:
		pipelineFn = p.pong
	case *rtapi.Envelope_GameServerCreateSucceed:
		pipelineFn = p.GameServerCreateSucceed
	default:
		// If we reached this point the envelope was valid but the contents are missing or unknown.
		// Usually caused by a version mismatch, and should cause the session making this pipeline request to close.
		logger.Error("Unrecognizable payload received.", zap.Any("payload", in))
		session.Send(&rtapi.Envelope{Cid: in.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_UNRECOGNIZED_PAYLOAD),
			Message: "Unrecognized message.",
		}}})
		return false
	}

	success, _ := pipelineFn(logger, session, in)

	if !success {
		return false
	}
	return true
}
