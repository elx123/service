package sessionws

import (
	"go.uber.org/atomic"

	"github.com/ardanlabs/service/foundation/lockfreemap"
	"github.com/google/uuid"
)

type LocalSessionRegistry struct {
	sessions     *lockfreemap.MapOf[uuid.UUID, SessionWS]
	sessionCount *atomic.Int32
}

func NewLocalSessionRegistry() *LocalSessionRegistry {
	return &LocalSessionRegistry{

		sessions:     &lockfreemap.MapOf[uuid.UUID, Session]{},
		sessionCount: atomic.NewInt32(0),
	}
}

func (r *LocalSessionRegistry) Remove(sessionID uuid.UUID) {
	r.sessions.Delete(sessionID)
	r.sessionCount.Dec()
}
