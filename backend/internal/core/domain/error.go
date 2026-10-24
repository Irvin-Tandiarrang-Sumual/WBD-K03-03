package domain

import "errors"

var (
	ErrInvalidGroupsIDs      = errors.New("Group ids are not valid")
	ErrGroupNotFound         = errors.New("Group is not found")
	ErrGroupMustHaveDatabase = errors.New("Group must have access to at least one database")
	ErrUserNotFound          = errors.New("User is not found")
	ErrUserMustHaveGroup     = errors.New("User must have at least one group")
	ErrDatabaseNotFound      = errors.New("Database is not found")
	ErrInvalidInput          = errors.New("Invalid input")
	ErrGroupNameTaken        = errors.New("Group name already used")
	ErrInvalidDatabaseAccess = errors.New("Invalid database access")
	ErrMembershipExists      = errors.New("User is already a group member")
	ErrMembershipNotFound    = errors.New("User is not a group member")
	ErrDatabaseAccessDenied  = errors.New("Database access denied")
	ErrDatabaseAlreadyExists = errors.New("Database already registered")
	ErrDatabaseInUse	 	 = errors.New("Database cannot be removed because a group would have no database")
	ErrUsernameTaken		 = errors.New("Username is already used")
	ErrInvalidRole  		 = errors.New("Invalid role")
	ErrInvalidSession = errors.New("Invalid or expired session")
	ErrInvalidCredentials = errors.New("Invalid username or password")
)
