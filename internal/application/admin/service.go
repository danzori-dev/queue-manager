package admin

import (
	"context"
	"fmt"
	"sync"

	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type Service struct {
	repo    AdminRepository
	ownerID int64

	mu     sync.RWMutex
	admins map[int64]struct{}
}

func NewService(repo AdminRepository, ownerID int64) *Service {
	return &Service{
		repo:    repo,
		ownerID: ownerID,
		admins:  make(map[int64]struct{}),
	}
}

func (s *Service) Load(ctx context.Context) error {
	ids, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("load admins: %w", err)
	}

	admins := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		admins[id] = struct{}{}
	}

	s.mu.Lock()
	s.admins = admins
	s.mu.Unlock()

	return nil
}

func (s *Service) RoleOf(userID int64) access.Role {
	if userID == s.ownerID {
		return access.RoleOwner
	}

	s.mu.RLock()
	_, ok := s.admins[userID]
	s.mu.RUnlock()

	if ok {
		return access.RoleAdmin
	}

	return access.RoleOutsider
}

func (s *Service) Add(ctx context.Context, targetID int64) error {
	if targetID == s.ownerID {
		return access.ErrCannotAddSelf
	}

	if s.isAdmin(targetID) {
		return access.ErrAdminAlreadyExists
	}

	if err := s.repo.Add(ctx, targetID); err != nil {
		return fmt.Errorf("add admin: %w", err)
	}

	s.mu.Lock()
	s.admins[targetID] = struct{}{}
	s.mu.Unlock()

	return nil
}

func (s *Service) Remove(ctx context.Context, targetID int64) error {
	if targetID == s.ownerID {
		return access.ErrCannotRemoveOwner
	}

	if !s.isAdmin(targetID) {
		return access.ErrAdminNotFound
	}

	if err := s.repo.Remove(ctx, targetID); err != nil {
		return fmt.Errorf("remove admin: %w", err)
	}

	s.mu.Lock()
	delete(s.admins, targetID)
	s.mu.Unlock()

	return nil
}

func (s *Service) List(ctx context.Context) ([]int64, error) {
	ids, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}

	return ids, nil
}

func (s *Service) isAdmin(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.admins[userID]

	return ok
}
