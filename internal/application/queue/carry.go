package queue

import (
	"context"
	"fmt"
	"slices"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (s *Service) PlanCarry(ctx context.Context, fromID int64, fromPos int, toID int64) (CarryPlan, error) {
	c, err := s.carry(ctx, fromID, fromPos, toID)
	if err != nil {
		return CarryPlan{}, err
	}

	return CarryPlan{
		From:  c.from,
		To:    c.to,
		Lines: s.lines(c.entries, fromPos),
	}, nil
}

func (s *Service) Carry(ctx context.Context, fromID int64, fromPos int, toID int64) (int, error) {
	c, err := s.carry(ctx, fromID, fromPos, toID)
	if err != nil {
		return 0, err
	}

	if err := s.entries.Prepend(ctx, c.to.ID, c.entries); err != nil {
		return 0, fmt.Errorf("prepend entries: %w", err)
	}

	s.scheduler.Schedule(c.to.ID)

	return len(c.entries), nil
}

type carryInput struct {
	from    domain.Event
	to      domain.Event
	entries []domain.Entry
}

func (s *Service) carry(ctx context.Context, fromID int64, fromPos int, toID int64) (carryInput, error) {
	from, err := s.events.Get(ctx, fromID)
	if err != nil {
		return carryInput{}, err
	}

	to, err := s.events.Get(ctx, toID)
	if err != nil {
		return carryInput{}, err
	}

	if err := domain.CanCarry(from, to); err != nil {
		return carryInput{}, err
	}

	entries, err := s.entries.List(ctx, fromID)
	if err != nil {
		return carryInput{}, fmt.Errorf("list entries: %w", err)
	}

	places := domain.Places(entries)

	if fromPos < 1 || fromPos > len(places) {
		return carryInput{}, domain.ErrCarryPosition
	}

	return carryInput{
		from:    from,
		to:      to,
		entries: slices.Concat(places[fromPos-1:]...),
	}, nil
}
