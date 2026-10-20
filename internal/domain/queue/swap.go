package queue

import "time"

const SwapTTL = 10 * time.Minute

type SwapRequest struct {
	ID            int64
	Event         Event
	FromID        int64
	ToID          int64
	FromPos       int
	ToPos         int
	FromMessageID int
	ToMessageID   int
	CreatedAt     time.Time
}

func (r SwapRequest) Expired() bool {
	return time.Since(r.CreatedAt) > SwapTTL
}

func (r SwapRequest) Involves(userID int64) bool {
	return r.FromID == userID || r.ToID == userID
}
