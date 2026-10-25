package queue

import (
	"errors"
	"time"
)

type RetryError struct {
	Err   error
	After time.Duration
}

func (e *RetryError) Error() string {
	return e.Err.Error()
}

func (e *RetryError) Unwrap() error {
	return e.Err
}

func RetryAfter(err error) time.Duration {
	if retry, ok := errors.AsType[*RetryError](err); ok {
		return retry.After
	}

	return 0
}
