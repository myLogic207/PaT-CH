package system

import "context"

type UserTable interface {
	Authenticate(ctx context.Context, name string, password string) (*User, error)
	Create(ctx context.Context, name string, email string, password string) (*User, error)
	GetByName(ctx context.Context, name string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	DeleteByName(ctx context.Context, user string) error
}

type UserIMDB struct {
	// UserTable
	Users       map[int64]*User
	IDPasswords map[int64]string
	idCounter   int64
}

func NewUserIMDB() *UserIMDB {
	return &UserIMDB{
		Users:       make(map[int64]*User),
		IDPasswords: make(map[int64]string),
	}
}

func (u *UserIMDB) Authenticate(ctx context.Context, name string, password string) (*User, error) {
	user, err := u.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if HashPassword(password) == u.IDPasswords[user.ID()] {
		return user, nil
	}
	return user, nil
}

func (u *UserIMDB) GetByName(ctx context.Context, name string) (*User, error) {
	for _, user := range u.Users {
		if user.Name == name {
			return user, nil
		}
	}

	return nil, nil
}

func (u *UserIMDB) GetByEmail(ctx context.Context, email string) (*User, error) {
	// panic("not implemented")
	return nil, nil
}

func (u *UserIMDB) GetById(ctx context.Context, id int64) (*User, error) {
	if u.Users[id] == nil {
		return nil, nil
	}
	return u.Users[id], nil
}

func (u *UserIMDB) GetUsers(ctx context.Context) ([]*User, error) {
	allUsers := make([]*User, 0, len(u.Users))
	for _, user := range u.Users {
		allUsers = append(allUsers, user)
	}
	return allUsers, nil
}

func (u *UserIMDB) Create(ctx context.Context, name string, email string, password string) (*User, error) {
	user := NewUser(name, email)
	id := u.nextId()
	user.SetID(id)
	u.Users[id] = user
	u.IDPasswords[id] = HashPassword(password)
	return user, nil
}

func (u *UserIMDB) Update(ctx context.Context, user *User) (*User, error) {
	u.Users[user.ID()] = user
	return user, nil
}

func (u *UserIMDB) DeleteById(ctx context.Context, id int64) error {
	delete(u.Users, id)
	delete(u.IDPasswords, id)
	return nil
}

func (u *UserIMDB) DeleteByName(ctx context.Context, name string) error {
	for id, user := range u.Users {
		if user.Name == name {
			delete(u.Users, id)
			delete(u.IDPasswords, id)
			return nil
		}
	}
	return nil
}

func (u *UserIMDB) nextId() int64 {
	i := u.idCounter
	u.idCounter++
	return i
}
