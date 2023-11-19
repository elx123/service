package ws

import (
	"github.com/ardanlabs/service/foundation/lockfreemap"
	"github.com/google/uuid"
	"go.uber.org/atomic"
)

type LocalSessionRegistry struct {
	userIds      map[string]string //userid - sessionid 用来临时记录
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

/*
func (r *LocalSessionRegistry) SingleSession(ctx context.Context, userID, sessionID uuid.UUID) {
	sessionIDs := tracker.ListLocalSessionIDByStream(PresenceStream{Mode: StreamModeNotifications, Subject: userID})
	for _, foundSessionID := range sessionIDs {
		if foundSessionID == sessionID {
			// Allow the current session, only disconnect any older ones.
			continue
		}
		session, ok := r.sessions.Load(foundSessionID)
		if ok {
			// No need to remove the session from the map, session.Close() will do that.
			session.Close("server-side session disconnect", runtime.PresenceReasonDisconnect,
				&rtapi.Envelope{Message: &rtapi.Envelope_Notifications{
					Notifications: &rtapi.Notifications{
						Notifications: []*api.Notification{
							{
								Id:         uuid.Must(uuid.NewV4()).String(),
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
}
*/
