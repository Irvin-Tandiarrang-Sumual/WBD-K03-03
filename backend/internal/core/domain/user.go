package domain

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