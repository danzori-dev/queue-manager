package queue

import (
	"context"
	"fmt"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type Deps struct {
	Events    EventRepository
	Entries   EntryRepository
	Targets   TargetRepository
	Students  StudentDirectory
	Messages  Messenger
	Scheduler RenderScheduler
}

type Service struct {
	events    EventRepository
	entries   EntryRepository
	targets   TargetRepository
	students  StudentDirectory
	messages  Messenger
	scheduler RenderScheduler
	swaps     *swapStore
}

func New(d Deps) *Service {
	return &Service{
		events:    d.Events,
		entries:   d.Entries,
		targets:   d.Targets,
		students:  d.Students,
		messages:  d.Messages,
		scheduler: d.Scheduler,
		swaps:     newSwapStore(),
	}
}

func (s *Service) view(ctx context.Context, event domain.Event) (View, error) {
	entries, err := s.entries.List(ctx, event.ID)
	if err != nil {
		return View{}, fmt.Errorf("list entries: %w", err)
	}

	return View{
		Event: event,
		Lines: s.lines(entries, 1),
	}, nil
}

func (s *Service) studentName(userID int64) string {
	if name, ok := s.students.Name(userID); ok {
		return name
	}

	return fmt.Sprintf("id%d", userID)
}

func (s *Service) loadOpenEvent(ctx context.Context, eventID int64) (domain.Event, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return domain.Event{}, err
	}

	if err := event.CanJoin(); err != nil {
		return domain.Event{}, err
	}

	return event, nil
}

func (s *Service) lines(entries []domain.Entry, firstPos int) []Line {
	lines := make([]Line, 0, len(entries))

	for i, place := range domain.Places(entries) {
		for _, e := range place {
			lines = append(
				lines, Line{
					Position: firstPos + i,
					UserID:   e.UserID,
					Name:     s.studentName(e.UserID),
					UserName: e.Username,
				},
			)
		}
	}

	return lines
}
