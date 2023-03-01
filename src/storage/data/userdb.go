package data

import (
	"context"
	"log"
)

type User struct {
	id        int
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	password  string
}

func (u *User) Password() string {
	log.Println("warn! User.Password() called")
	return u.password
}

func (u *User) SetPassword(password string) {
	log.Println("warn! User.SetPassword() called")
	u.password = password
}

type UserDB struct {
	p *DataBase
}

func (udb *UserDB) Create(ctx context.Context, name string, email string, password string) error {
	// TODO: Hash password
	passwordHash := password
	return udb.p.Insert(ctx, "users", []FieldName{"name", "email", "password"}, [][]interface{}{{name, email, passwordHash}})
}

func (udb *UserDB) GetById(ctx context.Context, id int) (*User, error) {
	val, err := udb.p.Select(ctx, "users", []FieldName{"id", "name", "email", "created_at"}, NewWhereMap(map[FieldName]interface{}{"id": id}), "LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, nil
	}
	return &User{
		id:        val[0]["id"].(int),
		Name:      val[0]["name"].(string),
		Email:     val[0]["email"].(string),
		CreatedAt: val[0]["created_at"].(string),
	}, nil
}

func (udb *UserDB) Update(ctx context.Context, user *User) error {
	usermap := map[FieldName]DBValue{
		"name":     user.Name,
		"email":    user.Email,
		"password": user.Password(),
	}
	return udb.p.Update(ctx, "users", usermap, NewWhereMap(map[FieldName]interface{}{"id": user.id}))
}

func (udb *UserDB) DeleteById(ctx context.Context, id int) error {
	return udb.p.Delete(ctx, "users", NewWhereMap(map[FieldName]interface{}{"id": id}))
}
