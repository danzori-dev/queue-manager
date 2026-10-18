package student

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEmptyList = errors.New("student list is empty")
	ErrInvalidID = errors.New("student id must be positive")
	ErrEmptyName = errors.New("student name is empty")
)

type Student struct {
	ID   int64
	Name string
}

func Validate(list map[int64]string) error {
	if len(list) == 0 {
		return ErrEmptyList
	}

	for id, name := range list {
		if id <= 0 {
			return fmt.Errorf("%w: %d", ErrInvalidID, id)
		}

		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%w: %d", ErrEmptyName, id)
		}
	}

	return nil
}
