package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mylogic207/PaT-CH/system"
)

var (
	ErrNoUser       = errors.New("no user found")
	ErrPassMismatch = errors.New("username or password incorrect")
	ErrUserExists   = errors.New("username or email already exists")
)

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

func (udb *UserDB) Create(ctx context.Context, name string, email string, password string) (*system.User, error) {
	now := time.Now().UTC()
	if _, err := udb.GetByName(ctx, name); err == nil {
		return nil, ErrUserExists
	}
	if _, err := udb.GetByEmail(ctx, email); err == nil {
		return nil, ErrUserExists
	}
	if strings.ContainsAny(name, " \t\r ") {
		return nil, fmt.Errorf("username cannot contain spaces")
	}
	hash := system.HashPassword(password)
	if err := udb.p.Insert(ctx, udb.usertable, []FieldName{"name", "email", "password", "created_at", "updated_at"}, [][]interface{}{{name, email, hash, now, now}}); err != nil {
		return nil, err
	}
	user, err := udb.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	go udb.updateCache(ctx, user)
	return user, nil
}

func (udb *UserDB) GetAll(ctx context.Context) ([]*system.User, error) {
	val, err := udb.p.Select(ctx, udb.usertable, []FieldName{"*"}, nil, "")
	if err != nil {
		return nil, err
	}
	users := make([]*system.User, len(val))
	for i, v := range val {
		users[i] = &system.User{
			Name:      v["name"].(string),
			Email:     v["email"].(string),
			CreatedAt: v["created_at"].(time.Time),
		}
		go udb.updateCache(ctx, users[i])
	}
	return users, nil
}

func (udb *UserDB) GetByName(ctx context.Context, name string) (*system.User, error) {
	if val, ok := udb.GetFromCache(ctx, name); ok {
		return val, nil
	}
	val, err := udb.p.Select(ctx, udb.usertable, []FieldName{"*"}, NewWhereMap(map[FieldName]interface{}{"name": name}), "LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrNoUser
	}
	user := &system.User{
		Name:      val[0]["name"].(string),
		Email:     val[0]["email"].(string),
		CreatedAt: val[0]["created_at"].(time.Time),
	}
	go udb.updateCache(ctx, user)
	return user, nil
}

func (udb *UserDB) GetByEmail(ctx context.Context, email string) (*system.User, error) {
	if val, ok := udb.GetFromCache(ctx, email); ok {
		return val, nil
	}
	val, err := udb.p.Select(ctx, udb.usertable, []FieldName{"*"}, NewWhereMap(map[FieldName]interface{}{"email": email}), "LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrNoUser
	}
	user := &system.User{
		Name:      val[0]["name"].(string),
		Email:     val[0]["email"].(string),
		CreatedAt: val[0]["created_at"].(time.Time),
	}
	go udb.updateCache(ctx, user)
	return user, nil
}

func (udb *UserDB) GetById(ctx context.Context, id int64) (*system.User, error) {
	if val, ok := udb.GetFromCache(ctx, id); ok {
		return val, nil
	}
	val, err := udb.p.Select(ctx, udb.usertable, []FieldName{"*"}, NewWhereMap(map[FieldName]interface{}{"id": id}), "LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrNoUser
	}
	user := &system.User{
		Name:      val[0]["name"].(string),
		Email:     val[0]["email"].(string),
		CreatedAt: val[0]["created_at"].(time.Time),
	}
	go udb.updateCache(ctx, user)
	return user, nil
}

func (udb *UserDB) Update(ctx context.Context, user *system.User) (*system.User, error) {
	timestamp := fmt.Sprint(time.Now().UTC())
	timestamp = timestamp[:len(timestamp)-9]
	usermap := map[FieldName]DBValue{
		"name":       user.Name,
		"email":      user.Email,
		"password":   user.Password(),
		"updated_at": timestamp,
	}
	if err := udb.p.Update(ctx, udb.usertable, usermap, NewWhereMap(map[FieldName]interface{}{"id": user.ID()})); err != nil {
		return nil, err
	}
	go udb.updateCache(ctx, user)
	return user, nil
}

func (udb *UserDB) DeleteById(ctx context.Context, id int64) error {
	user, err := udb.GetById(ctx, id)
	if err != nil {
		return ErrNoUser
	}
	go udb.clearCache(ctx, user)
	return udb.p.Delete(ctx, udb.usertable, NewWhereMap(map[FieldName]interface{}{"id": id}))
}

func (udb *UserDB) DeleteByName(ctx context.Context, name string) error {
	user, err := udb.GetByName(ctx, name)
	if err != nil {
		return ErrNoUser
	}
	go udb.clearCache(ctx, user)
	return udb.p.Delete(ctx, udb.usertable, NewWhereMap(map[FieldName]interface{}{"name": name}))
}

// Tests a user against the database, return no error if the user is found and the password matches
func (udb *UserDB) Authenticate(ctx context.Context, nameORemail string, password string) (*system.User, error) {
	if strings.Contains(nameORemail, "@") {
		return udb.AuthenticateByEmail(ctx, nameORemail, password)
	} else {
		return udb.AuthenticateByName(ctx, nameORemail, password)
	}
}

func (udb *UserDB) AuthenticateByName(ctx context.Context, name string, password string) (*system.User, error) {
	user, err := udb.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if user.CheckPassword(password) {
		return nil, ErrPassMismatch
	}
	return user, nil
}

func (udb *UserDB) AuthenticateByEmail(ctx context.Context, email string, password string) (*system.User, error) {
	user, err := udb.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user.CheckPassword(password) {
		return nil, ErrPassMismatch
	}
	return user, nil
}

func (udb *UserDB) GetFromCache(ctx context.Context, key interface{}) (*system.User, bool) {
	// if val, ok := usb.p.cache.Get(ctx, name); ok {
	// 	return val.(*system.User), nil
	// }
	// return nil, ErrNoUser
	if val, ok := udb.p.cache.Get(ctx, fmt.Sprint("user_", key)); ok {
		user := system.User{}
		if s, ok := val.(string); ok {
			if err := json.Unmarshal([]byte(s), &user); err != nil {
				logger.Println(err)
				return nil, false
			}
		}
		return &user, true
	}
	return nil, false
}

func (udb *UserDB) updateCache(ctx context.Context, user *system.User) {
	udb.p.cache.Set(ctx, fmt.Sprint("user_", user.Name), user)
	udb.p.cache.Set(ctx, fmt.Sprint("user_", user.Email), user)
	udb.p.cache.Set(ctx, fmt.Sprint("user_", user.ID()), user)
}

func (udb *UserDB) clearCache(ctx context.Context, user *system.User) {
	udb.p.cache.Delete(ctx, fmt.Sprint("user_", user.Name))
	udb.p.cache.Delete(ctx, fmt.Sprint("user_", user.Email))
	udb.p.cache.Delete(ctx, fmt.Sprint("user_", user.ID()))
}
