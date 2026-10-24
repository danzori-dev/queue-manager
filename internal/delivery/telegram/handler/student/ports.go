package student

import (
	"context"

	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

type StudentService interface {
	Import(ctx context.Context, list map[int64]string) error
	List() []student.Student
}
