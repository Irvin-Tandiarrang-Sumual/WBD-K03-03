package testutil

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type UserRepository struct {
	ports.UserRepository
	FindAllFunc                    func(context.Context, pagination.PaginationParams) ([]domain.User, int64, error)
	FindByIDFunc                   func(context.Context, uint) (domain.User, error)
	CountByRoleFunc                func(context.Context, string) (int64, error)
	SaveFunc                       func(context.Context, domain.User, []uint) (domain.User, error)
	UpdateFunc                     func(context.Context, uint, *string, *string) (domain.User, error)
	DeleteFunc                     func(context.Context, uint) error
	FindByUsernameWithPasswordFunc func(context.Context, string) (domain.User, error)
	UpdateRoleFunc                 func(context.Context, uint, string) (domain.User, error)
}

func (f *UserRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	if f.FindAllFunc != nil {
		return f.FindAllFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *UserRepository) FindByID(ctx context.Context, id uint) (domain.User, error) {
	if f.FindByIDFunc != nil {
		return f.FindByIDFunc(ctx, id)
	}
	return domain.User{}, nil
}

func (f *UserRepository) CountByRole(ctx context.Context, role string) (int64, error) {
	if f.CountByRoleFunc != nil {
		return f.CountByRoleFunc(ctx, role)
	}
	return 0, nil
}

func (f *UserRepository) Save(ctx context.Context, user domain.User, groupIDs []uint) (domain.User, error) {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, user, groupIDs)
	}
	return domain.User{}, nil
}

func (f *UserRepository) Update(ctx context.Context, id uint, username *string, passwordHash *string) (domain.User, error) {
	if f.UpdateFunc != nil {
		return f.UpdateFunc(ctx, id, username, passwordHash)
	}
	return domain.User{}, nil
}

func (f *UserRepository) Delete(ctx context.Context, id uint) error {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, id)
	}
	return nil
}

func (f *UserRepository) FindByUsernameWithPassword(ctx context.Context, username string) (domain.User, error) {
	if f.FindByUsernameWithPasswordFunc != nil {
		return f.FindByUsernameWithPasswordFunc(ctx, username)
	}
	return domain.User{}, nil
}

func (f *UserRepository) UpdateRole(ctx context.Context, id uint, role string) (domain.User, error) {
	if f.UpdateRoleFunc != nil {
		return f.UpdateRoleFunc(ctx, id, role)
	}
	return domain.User{}, nil
}

type UserService struct {
	ports.UserService
	GetAllUsersFunc    func(context.Context, pagination.PaginationParams) ([]domain.User, int64, error)
	GetUserByIDFunc    func(context.Context, uint) (domain.User, error)
	CreateUserFunc     func(context.Context, domain.CreateUserInput) (domain.User, error)
	UpdateUserFunc     func(context.Context, uint, domain.UpdateUserInput) (domain.User, error)
	DeleteUserFunc     func(context.Context, uint) error
	UpdateUserRoleFunc func(context.Context, uint, string) (domain.User, error)
}

func (f *UserService) GetAllUsers(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	if f.GetAllUsersFunc != nil {
		return f.GetAllUsersFunc(ctx, params)
	}
	return nil, 0, nil
}

func (f *UserService) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	if f.GetUserByIDFunc != nil {
		return f.GetUserByIDFunc(ctx, id)
	}
	return domain.User{}, nil
}

func (f *UserService) CreateUser(ctx context.Context, input domain.CreateUserInput) (domain.User, error) {
	if f.CreateUserFunc != nil {
		return f.CreateUserFunc(ctx, input)
	}
	return domain.User{}, nil
}

func (f *UserService) UpdateUser(ctx context.Context, id uint, input domain.UpdateUserInput) (domain.User, error) {
	if f.UpdateUserFunc != nil {
		return f.UpdateUserFunc(ctx, id, input)
	}
	return domain.User{}, nil
}

func (f *UserService) DeleteUser(ctx context.Context, id uint) error {
	if f.DeleteUserFunc != nil {
		return f.DeleteUserFunc(ctx, id)
	}
	return nil
}

func (f *UserService) UpdateUserRole(ctx context.Context, id uint, role string) (domain.User, error) {
	if f.UpdateUserRoleFunc != nil {
		return f.UpdateUserRoleFunc(ctx, id, role)
	}
	return domain.User{}, nil
}
