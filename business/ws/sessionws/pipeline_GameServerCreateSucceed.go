package pipeline

import (
	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/ardanlabs/service/business/ws/sessionws"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (p *Pipeline) GameServerCreateSucceed(logger *zap.Logger, session *sessionws.SessionWS, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	incoming := envelope.GetGameServerCreateSucceed()

	out := &rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_GameServerCreateSucceed{GameServerCreateSucceed: &rtapi.GameServerCreateSucceed{
		IpAddress: incoming.IpAddress,
		Port:      incoming.Port,
	}}}
	err := session.Send(out)
	if err != nil && err != sessionws.ErrSessionQueueFull {
		envelope := &rtapi.Envelope{Cid: uuid.NewString(), Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_RUNTIME_EXCEPTION),
			Message: "Server internel error",
		}}}
		session.Close(envelope)
		return false, envelope
	}
	return true, out
}
