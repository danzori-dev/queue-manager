package access

import "errors"

var (
	ErrCannotAddSelf      = errors.New("cannot add self as admin")
	ErrCannotRemoveOwner  = errors.New("cannot remove owner")
	ErrAdminAlreadyExists = errors.New("admin already exists")
	ErrAdminNotFound      = errors.New("admin not found")
)
