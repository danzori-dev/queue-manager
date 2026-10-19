package queue

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
