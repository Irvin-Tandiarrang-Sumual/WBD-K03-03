package errors

import "errors"

var (
	ErrInvalidGroupsIDs      = errors.New("Group ids are not valid")
	ErrGroupNotFound         = errors.New("Group is not found")
	ErrGroupMustHaveDatabase = errors.New("Group must have access to at least one database")
)
