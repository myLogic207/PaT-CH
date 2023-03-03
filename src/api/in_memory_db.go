package api

import (
	"context"

	"github.com/mylogic207/PaT-CH/system"
)

type UserTable interface {
	Authenticate(ctx context.Context, name string, password string) (*system.User, error)
	Create(ctx context.Context, name string, email string, password string) (*system.User, error)
	GetByName(ctx context.Context, name string) (*system.User, error)
	GetByEmail(ctx context.Context, email string) (*system.User, error)
	Update(ctx context.Context, user *system.User) (*system.User, error)
	DeleteByName(ctx context.Context, user string) error
}

type UserIMDB struct {
	UserTable
	Users map[string]*system.User
}

func NewUserIMDB() *UserIMDB {
	return &UserIMDB{
		Users: make(map[string]*system.User),
	}
}

func (u *UserIMDB) Authenticate(ctx context.Context, name string, password string) (*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) GetByName(ctx context.Context, name string) (*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) GetByEmail(ctx context.Context, email string) (*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) GetById(ctx context.Context, id int64) (*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) GetUsers(ctx context.Context) ([]*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) Create(ctx context.Context, name string, email string, password string) (*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) Update(ctx context.Context, user *system.User) (*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) DeleteById(ctx context.Context, id int64) error {
	return nil
}

func (u *UserIMDB) DeleteByName(ctx context.Context, name string) error {
	return nil
}
