package pipeline

import (
	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/ardanlabs/service/business/ws/sessionws"
	"go.uber.org/zap"
)

func (p *Pipeline) ping(logger *zap.Logger, session *sessionws.SessionWS, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	out := &rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Pong{Pong: &rtapi.Pong{}}}
	session.Send(out, true)

	return true, out
}

func (p *Pipeline) pong(logger *zap.Logger, session *sessionws.SessionWS, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	// No application-level action in response to a pong message.
	return true, nil
}
