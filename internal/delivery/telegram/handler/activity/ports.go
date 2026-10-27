package activity

import (
	"github.com/danzori-dev/queue-manager/internal/application/activity"
	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

type Journal interface {
	Users() []int64
	Events(userID int64) []activity.Event
}

type StudentService interface {
	List() []student.Student
	Name(userID int64) (string, bool)
}
