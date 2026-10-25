package queue

import "time"

const InviteTTL = 10 * time.Minute

type InviteRequest struct {
	ID            int64
	Event         Event
	FromID        int64
	ToID          int64
	Position      int
	ToPosition    int
	FromMessageID int
	ToMessageID   int
	CreatedAt     time.Time
}

func (r InviteRequest) Expired() bool {
	return time.Since(r.CreatedAt) > InviteTTL
}

func (r InviteRequest) Involves(userID int64) bool {
	return r.FromID == userID || r.ToID == userID
}
