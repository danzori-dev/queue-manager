package admin

import (
	"context"
)

type AdminService interface {
	Add(ctx context.Context, targetID int64) error
	Remove(ctx context.Context, targetID int64) error
	List(ctx context.Context) ([]int64, error)
}

type StudentService interface {
	Name(userID int64) (string, bool)
	Count() int
}
