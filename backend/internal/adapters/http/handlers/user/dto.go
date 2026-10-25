package user

import "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	GroupIDs []uint `json:"groupIds" binding:"required"`
}

type UpdateUserRequest struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
}

type SetRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func toUserResponse(user domain.User) UserResponse {
	return UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}
