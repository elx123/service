package pipeline

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/ardanlabs/service/business/ws/schema/sessionws"
	"github.com/ardanlabs/service/business/ws/sessionws"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

type Pipeline struct {
	logger               *zap.Logger
	config               Config
	db                   *sql.DB
	protojsonMarshaler   *protojson.MarshalOptions
	protojsonUnmarshaler *protojson.UnmarshalOptions
	sessionRegistry      SessionRegistry
	statusRegistry       *StatusRegistry
	matchRegistry        MatchRegistry
	partyRegistry        PartyRegistry
	matchmaker           Matchmaker
	tracker              Tracker
	router               MessageRouter
}

func NewPipeline(logger *zap.Logger, config Config, db *sql.DB, protojsonMarshaler *protojson.MarshalOptions, protojsonUnmarshaler *protojson.UnmarshalOptions, sessionRegistry SessionRegistry, statusRegistry *StatusRegistry, matchRegistry MatchRegistry, partyRegistry PartyRegistry, matchmaker Matchmaker, tracker Tracker, router MessageRouter, runtime *Runtime) *Pipeline {
	return &Pipeline{
		logger:               logger,
		config:               config,
		db:                   db,
		protojsonMarshaler:   protojsonMarshaler,
		protojsonUnmarshaler: protojsonUnmarshaler,
		sessionRegistry:      sessionRegistry,
		statusRegistry:       statusRegistry,
		matchRegistry:        matchRegistry,
		partyRegistry:        partyRegistry,
		matchmaker:           matchmaker,
		tracker:              tracker,
		router:               router,
		runtime:              runtime,
		node:                 config.GetName(),
	}
}

func (p *Pipeline) ProcessRequest(logger *zap.Logger, session sessionws.SessionWS, in *rtapi.Envelope) bool {
	if in.Message == nil {
		session.Send(&rtapi.Envelope{Cid: in.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_MISSING_PAYLOAD),
			Message: "Missing message.",
		}}}, true)
		return false
	}

	var pipelineFn func(*zap.Logger, sessionws.SessionWS, *rtapi.Envelope) (bool, *rtapi.Envelope)

	switch in.Message.(type) {
	case *rtapi.Envelope_Ping:
		pipelineFn = p.ping
	case *rtapi.Envelope_Pong:
		pipelineFn = p.pong

	default:
		// If we reached this point the envelope was valid but the contents are missing or unknown.
		// Usually caused by a version mismatch, and should cause the session making this pipeline request to close.
		logger.Error("Unrecognizable payload received.", zap.Any("payload", in))
		session.Send(&rtapi.Envelope{Cid: in.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_UNRECOGNIZED_PAYLOAD),
			Message: "Unrecognized message.",
		}}}, true)
		return false
	}

	var messageName, messageNameID string

	switch in.Message.(type) {
	case *rtapi.Envelope_Rpc:
		// No before/after hooks on RPC.
	default:
		messageName = fmt.Sprintf("%T", in.Message)
		messageNameID = strings.ToLower(messageName)

		if fn := p.runtime.BeforeRt(messageNameID); fn != nil {
			hookResult, hookErr := fn(session.Context(), logger, session.UserID().String(), session.Username(), session.Vars(), session.Expiry(), session.ID().String(), session.ClientIP(), session.ClientPort(), session.Lang(), in)

			if hookErr != nil {
				// Errors from before hooks do not close the session.
				session.Send(&rtapi.Envelope{Cid: in.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
					Code:    int32(rtapi.Error_RUNTIME_FUNCTION_EXCEPTION),
					Message: hookErr.Error(),
				}}}, true)
				return true
			} else if hookResult == nil {
				// If result is nil, requested resource is disabled. Sessions calling disabled resources will be closed.
				logger.Warn("Intercepted a disabled resource.", zap.String("resource", messageName))
				session.Send(&rtapi.Envelope{Cid: in.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
					Code:    int32(rtapi.Error_UNRECOGNIZED_PAYLOAD),
					Message: "Requested resource was not found.",
				}}}, true)
				return false
			}

			in = hookResult
		}
	}

	success, out := pipelineFn(logger, session, in)

	if success && messageName != "" {
		// Unsuccessful operations do not trigger after hooks.
		if fn := p.runtime.AfterRt(messageNameID); fn != nil {
			fn(session.Context(), logger, session.UserID().String(), session.Username(), session.Vars(), session.Expiry(), session.ID().String(), session.ClientIP(), session.ClientPort(), session.Lang(), out, in)
		}
	}

	return true
}
