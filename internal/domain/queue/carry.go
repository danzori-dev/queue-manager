package queue

type CarrySelection struct {
	FromID  int64
	FromPos int
	Places  [][]int64
}

func CanCarry(from, to Event) error {
	switch {
	case from.ID == to.ID:
		return ErrCarrySameEvent

	case from.Status != StatusClosed:
		return ErrCarrySourceOpen

	case to.Status != StatusOpen:
		return ErrCarryTargetClosed

	default:
		return nil
	}
}
