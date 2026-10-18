package student

import "context"

type StudentRepository interface {
	ReplaceAll(ctx context.Context, students map[int64]string) error
	LoadAll(ctx context.Context) (map[int64]string, error)
}
