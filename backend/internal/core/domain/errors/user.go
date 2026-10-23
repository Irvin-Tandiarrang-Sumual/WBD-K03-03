package errors

import "errors"

var (
	ErrUserNotFound      = errors.New("User is not found")
	ErrUserMustHaveGroup = errors.New("User must have at least one group")
)
