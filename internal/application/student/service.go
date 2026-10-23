package student

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

type Service struct {
	repo StudentRepository

	mu       sync.RWMutex
	students map[int64]string
}

func NewService(repo StudentRepository) *Service {
	return &Service{
		repo:     repo,
		students: make(map[int64]string),
	}
}

func (s *Service) Load(ctx context.Context) error {
	students, err := s.repo.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("load students: %w", err)
	}

	s.mu.Lock()
	s.students = students
	s.mu.Unlock()

	return nil
}

func (s *Service) Import(ctx context.Context, list map[int64]string) error {
	if err := student.Validate(list); err != nil {
		return err
	}

	next := maps.Clone(list)

	if err := s.repo.ReplaceAll(ctx, next); err != nil {
		return fmt.Errorf("replace students: %w", err)
	}

	s.mu.Lock()
	s.students = next
	s.mu.Unlock()

	return nil
}

func (s *Service) List() []student.Student {
	s.mu.RLock()
	list := make([]student.Student, 0, len(s.students))

	for id, name := range s.students {
		list = append(
			list, student.Student{
				ID:   id,
				Name: name,
			},
		)
	}

	s.mu.RUnlock()

	slices.SortFunc(
		list, func(a, b student.Student) int {
			if c := strings.Compare(a.Name, b.Name); c != 0 {
				return c
			}

			return cmp.Compare(a.ID, b.ID)
		},
	)

	return list
}

func (s *Service) Name(userID int64) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name, ok := s.students[userID]

	return name, ok
}

func (s *Service) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.students)
}
