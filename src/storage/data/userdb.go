package data

import (
	"context"
	"fmt"
	"log"
	"time"
)

var (
	ErrNoUser = fmt.Errorf("no user found")
)

type User struct {
	id        int64
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	updatedAt time.Time
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
	p         *DataBase
	usertable string
}

func NewUserDB(p *DataBase, usertable string) *UserDB {
	return &UserDB{p, usertable}
}

func (udb *UserDB) SetTableName(usertable string) {
	udb.usertable = usertable
}

func (udb *UserDB) Create(ctx context.Context, name string, email string, password string) error {
	// TODO: Hash password
	passwordHash := password
	now := time.Now().UTC()
	return udb.p.Insert(ctx, udb.usertable, []FieldName{"name", "email", "password", "created_at", "updated_at"}, [][]interface{}{{name, email, passwordHash, now, now}})
}

func (udb *UserDB) GetAll(ctx context.Context) ([]*User, error) {
	val, err := udb.p.Select(ctx, udb.usertable, []FieldName{"*"}, nil, "")
	if err != nil {
		return nil, err
	}
	users := make([]*User, len(val))
	for i, v := range val {
		users[i] = &User{
			id:        v["id"].(int64),
			Name:      v["name"].(string),
			Email:     v["email"].(string),
			CreatedAt: v["created_at"].(time.Time),
		}
	}
	return users, nil
}

func (udb *UserDB) GetByName(ctx context.Context, name string) (*User, error) {
	val, err := udb.p.Select(ctx, udb.usertable, []FieldName{"*"}, NewWhereMap(map[FieldName]interface{}{"name": name}), "LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrNoUser
	}
	return &User{
		id:        val[0]["id"].(int64),
		Name:      val[0]["name"].(string),
		Email:     val[0]["email"].(string),
		CreatedAt: val[0]["created_at"].(time.Time),
		updatedAt: val[0]["updated_at"].(time.Time),
		password:  val[0]["password"].(string),
	}, nil
}

func (udb *UserDB) GetById(ctx context.Context, id int64) (*User, error) {
	val, err := udb.p.Select(ctx, udb.usertable, []FieldName{"*"}, NewWhereMap(map[FieldName]interface{}{"id": id}), "LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrNoUser
	}
	return &User{
		id:        val[0]["id"].(int64),
		Name:      val[0]["name"].(string),
		Email:     val[0]["email"].(string),
		CreatedAt: val[0]["created_at"].(time.Time),
		updatedAt: val[0]["updated_at"].(time.Time),
		password:  val[0]["password"].(string),
	}, nil
}

func (udb *UserDB) Update(ctx context.Context, user *User) error {
	timestamp := fmt.Sprint(time.Now().UTC())
	timestamp = timestamp[:len(timestamp)-9]
	usermap := map[FieldName]DBValue{
		"name":       user.Name,
		"email":      user.Email,
		"password":   user.Password(),
		"updated_at": timestamp,
	}
	return udb.p.Update(ctx, udb.usertable, usermap, NewWhereMap(map[FieldName]interface{}{"id": user.id}))
}

func (udb *UserDB) DeleteById(ctx context.Context, id int64) error {
	return udb.p.Delete(ctx, udb.usertable, NewWhereMap(map[FieldName]interface{}{"id": id}))
}
