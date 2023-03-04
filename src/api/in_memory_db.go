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
	Users     map[string]*system.User
	UsersID   map[int64]string
	idcounter int64
}

func NewUserIMDB() *UserIMDB {
	return &UserIMDB{
		Users:   make(map[string]*system.User),
		UsersID: make(map[int64]string),
	}
}

func (u *UserIMDB) Authenticate(ctx context.Context, name string, password string) (*system.User, error) {
	user := u.Users[name]
	if user == nil {
		return nil, nil
	}
	if !user.CheckPassword(password) {
		return nil, nil
	}
	return user, nil
}

func (u *UserIMDB) GetByName(ctx context.Context, name string) (*system.User, error) {
	return u.Users[name], nil
}

func (u *UserIMDB) GetByEmail(ctx context.Context, email string) (*system.User, error) {
	return nil, nil
}

func (u *UserIMDB) GetById(ctx context.Context, id int64) (*system.User, error) {
	return u.Users[u.UsersID[id]], nil
}

func (u *UserIMDB) GetUsers(ctx context.Context) ([]*system.User, error) {
	allUsers := make([]*system.User, 0, len(u.Users))
	for _, user := range u.Users {
		allUsers = append(allUsers, user)
	}
	return allUsers, nil
}

func (u *UserIMDB) Create(ctx context.Context, name string, email string, password string) (*system.User, error) {
	user := system.NewUser(name, email, password)
	user.SetID(u.nextId())
	u.Users[name] = user
	u.UsersID[user.ID()] = name
	return user, nil
}

func (u *UserIMDB) Update(ctx context.Context, user *system.User) (*system.User, error) {
	u.Users[user.Name] = user
	u.UsersID[user.ID()] = user.Name
	return user, nil
}

func (u *UserIMDB) DeleteById(ctx context.Context, id int64) error {
	name := u.UsersID[id]
	delete(u.Users, name)
	delete(u.UsersID, id)
	return nil
}

func (u *UserIMDB) DeleteByName(ctx context.Context, name string) error {
	id := u.Users[name].ID()
	delete(u.Users, name)
	delete(u.UsersID, id)
	return nil
}

func (u *UserIMDB) nextId() int64 {
	i := u.idcounter
	u.idcounter++
	return i
}
