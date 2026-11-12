package domain

const (
	RoleUser          = "user"
	RoleAdmin         = "admin"
	MaxPasswordBytes  = 72
	MaxUsernameLength = 255
)

type User struct {
	ID       uint
	Username string
	Password string
	Role     string
}

type CreateUserInput struct {
	Username string
	Password string
	GroupIDs []uint
}

type UpdateUserInput struct {
	Username *string
	Password *string
}

func ValidatePassword(password string) error {
	if password == "" {
		return ErrPasswordEmpty
	}
	if len([]byte(password)) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}

func IsValidRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}
