package ws

import (
	"context"
	"sync"
	"time"

	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/ardanlabs/service/foundation/lockfreemap"
	"github.com/google/uuid"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type LocalSessionRegistry struct {
	sync.RWMutex
	userIds      map[uuid.UUID][]uuid.UUID //userid - sessionid 用来临时记录
	sessions     *lockfreemap.MapOf[uuid.UUID, *SessionWS]
	sessionCount *atomic.Int32
}

func NewLocalSessionRegistry() *LocalSessionRegistry {
	return &LocalSessionRegistry{

		sessions:     &lockfreemap.MapOf[uuid.UUID, *SessionWS]{},
		sessionCount: atomic.NewInt32(0),
	}
}

func (r *LocalSessionRegistry) Add(session *SessionWS) {
	r.sessions.Store(session.ID(), session)
	r.sessionCount.Inc()
}

func (r *LocalSessionRegistry) Remove(sessionID uuid.UUID) {
	r.sessions.Delete(sessionID)
	r.sessionCount.Dec()
}

func (r *LocalSessionRegistry) Get(sessionID uuid.UUID) *SessionWS {
	session, ok := r.sessions.Load(sessionID)
	if !ok {
		return nil
	}
	return session
}

func (r *LocalSessionRegistry) SingleSession(ctx context.Context, userID, sessionid uuid.UUID) error {
	r.Lock()
	sessionids, ok := r.userIds[userID]
	if !ok {
		return nil
	}

	if len(sessionids) == 0 {
		return nil
	}

	for _, foundSessionID := range sessionids {

		if foundSessionID == sessionid {
			continue
		}

		session, ok := r.sessions.Load(foundSessionID)
		if ok {
			// No need to remove the session from the map, session.Close() will do that.
			session.Close(&rtapi.Envelope{Message: &rtapi.Envelope_Notifications{
				Notifications: &rtapi.Notifications{
					Notifications: []*rtapi.Notification{
						{
							Id:         uuid.NewString(),
							Subject:    "single_socket",
							Content:    "{}",
							Code:       NotificationCodeSingleSocket,
							SenderId:   "",
							CreateTime: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
							Persistent: false,
						},
					},
				},
			}})
		}
	}
	return nil
}
