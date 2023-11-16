package pipeline

import (
	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/ardanlabs/service/business/ws/sessionws"
	"go.uber.org/zap"
)

func (p *Pipeline) GameServerCreateSucceed(logger *zap.Logger, session *sessionws.SessionWS, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	incoming := envelope.GetGameServerCreateSucceed()

	out := &rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_GameServerCreateSucceed{GameServerCreateSucceed: &rtapi.GameServerCreateSucceed{
		IpAddress: incoming.IpAddress,
		Port:      incoming.Port,
	}}}
	session.Send(out, true)

	return true, out
}
