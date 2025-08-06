package repo

import (
	"errors"
	"katkam/internal/config"
)

type UserRepository struct {
	users []config.User
}

func NewUserRepository(users []config.User) *UserRepository {
	return &UserRepository{
		users: users,
	}
}

func (r *UserRepository) GetByUsername(username string) (*config.User, error) {
	for _, user := range r.users {
		if user.Username == username {
			return &user, nil
		}
	}
	return nil, errors.New("user not found")
}
