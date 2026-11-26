package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type Deps struct {
	Events       EventRepository
	Entries      EntryRepository
	Targets      TargetRepository
	Students     StudentDirectory
	Messages     Messenger
	Scheduler    RenderScheduler
	RequestLimit RequestLimit
	Log          *slog.Logger
	Activity     ActivityRecorder
}

type Service struct {
	events        EventRepository
	entries       EntryRepository
	targets       TargetRepository
	students      StudentDirectory
	messages      Messenger
	scheduler     RenderScheduler
	renderLocksMu sync.Mutex
	renderLocks   map[int64]*eventRenderLock
	swaps         *swapStore
	invites       *inviteStore
	throttle      *requestThrottle
	log           *slog.Logger
	activity      ActivityRecorder
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
		invites:   newInviteStore(),
		throttle:  newRequestThrottle(d.RequestLimit),
		log:       d.Log,
		activity:  d.Activity,
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

func (s *Service) audit(ctx context.Context, userID int64, action string, attrs ...slog.Attr) {
	s.log.LogAttrs(ctx, slog.LevelInfo, action, attrs...)
	s.activity.Record(userID, action, attrs)
}

type eventRenderLock struct {
	mu   sync.Mutex
	refs int
}

func (s *Service) lockEventRender(eventID int64) func() {
	s.renderLocksMu.Lock()

	if s.renderLocks == nil {
		s.renderLocks = make(map[int64]*eventRenderLock)
	}

	lock := s.renderLocks[eventID]
	if lock == nil {
		lock = &eventRenderLock{}
		s.renderLocks[eventID] = lock
	}

	lock.refs++

	s.renderLocksMu.Unlock()

	lock.mu.Lock()

	return func() {
		lock.mu.Unlock()

		s.renderLocksMu.Lock()
		defer s.renderLocksMu.Unlock()

		lock.refs--
		if lock.refs == 0 {
			delete(s.renderLocks, eventID)
		}
	}
}
