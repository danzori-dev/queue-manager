package invite

import (
	"context"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type InviteService interface {
	InviteTargets(ctx context.Context, userID int64) ([]domain.Event, error)
	InviteCandidates(ctx context.Context, eventID, userID int64) (appqueue.InviteCandidates, error)
	RequestInvite(ctx context.Context, eventID, fromID, toID int64, fromMessageID int) (appqueue.Invite, error)
	AcceptInvite(ctx context.Context, requestID, userID int64, username string) (appqueue.Invite, error)
	DeclineInvite(_ context.Context, requestID, userID int64) (appqueue.Invite, error)
	CancelInvite(_ context.Context, requestID, userID int64) (appqueue.Invite, error)
}
