package admin

import "context"

type AdminRepository interface {
	Add(ctx context.Context, userID int64) error
	Remove(ctx context.Context, userID int64) error
	List(ctx context.Context) ([]int64, error)
}
