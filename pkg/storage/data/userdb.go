package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/myLogic207/PaT-CH/internal/system"
)

var (
	ErrNoUser       = errors.New("no user found")
	ErrPassMismatch = errors.New("username or password incorrect")
	ErrUserExists   = errors.New("username or email already exists")
	ErrCreateUser   = errors.New("error creating user")
)

type UserDB struct {
	p         *DataBase
	userTable string
	logger    *log.Logger
}

func NewUserDB(p *DataBase, userTable string, logger *log.Logger) *UserDB {
	userTable = strings.ToLower(userTable)
	userTable = strings.TrimSpace(userTable)
	if logger == nil {
		logger = log.Default()
	}
	return &UserDB{
		p:         p,
		userTable: userTable,
		logger:    logger,
	}
}

func (udb *UserDB) SetTableName(userTable string) {
	udb.userTable = userTable
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
	if err := udb.p.Insert(ctx, udb.userTable, []FieldName{"name", "email", "password", "created_at", "updated_at"}, [][]interface{}{{name, email, hash, now, now}}); err != nil {
		return nil, err
	}
	user, err := udb.GetByName(ctx, name)
	if err != nil {
		udb.logger.Println(err)
		return nil, ErrCreateUser
	}
	udb.logger.Printf("Created user %s\n", name)
	go udb.updateCache(ctx, user)
	return user, nil
}

func (udb *UserDB) GetAll(ctx context.Context) ([]*system.User, error) {
	udb.logger.Println("Getting all users")
	val, err := udb.p.Select(ctx, udb.userTable, []FieldName{"*"}, nil, "")
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

var USER_FIELDS = []FieldName{"id", "name", "email", "created_at", "updated_at"}

func (udb *UserDB) GetByName(ctx context.Context, name string) (*system.User, error) {
	if val, ok := udb.GetFromCache(ctx, name); ok {
		return val, nil
	}
	whereClause := NewWhereMap(map[FieldName]interface{}{"name": name})
	return udb.getUserWithWhere(ctx, whereClause)
}

func (udb *UserDB) GetByEmail(ctx context.Context, email string) (*system.User, error) {
	if val, ok := udb.GetFromCache(ctx, email); ok {
		return val, nil
	}
	whereClause := NewWhereMap(map[FieldName]interface{}{"email": email})
	return udb.getUserWithWhere(ctx, whereClause)
}

func (udb *UserDB) GetById(ctx context.Context, id int64) (*system.User, error) {
	if val, ok := udb.GetFromCache(ctx, id); ok {
		return val, nil
	}
	whereClause := NewWhereMap(map[FieldName]interface{}{"id": id})
	return udb.getUserWithWhere(ctx, whereClause)
}

func (udb *UserDB) getUserWithWhere(ctx context.Context, whereMap *WhereMap) (*system.User, error) {
	raw_db_user, err := udb.p.Select(ctx, udb.userTable, USER_FIELDS, whereMap, "LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(raw_db_user) == 0 {
		return nil, ErrNoUser
	}
	var created, updated time.Time
	if val, ok := raw_db_user[0]["created_at"].(time.Time); ok {
		created = val
	}
	if val, ok := raw_db_user[0]["updated_at"].(time.Time); ok {
		updated = val
	}
	user := system.LoadUser(raw_db_user[0]["id"].(int64), raw_db_user[0]["name"].(string), raw_db_user[0]["email"].(string), &created, &updated)
	udb.logger.Println("Got user", user.Name)
	go udb.updateCache(ctx, user)
	return user, nil
}

var USER_PASSWORD_FIELDS = []FieldName{"password"} // salt, etc.

func (udb *UserDB) verifyPasswordById(ctx context.Context, id int64, password string) bool {
	val, err := udb.p.Select(ctx, udb.userTable, USER_PASSWORD_FIELDS, NewWhereMap(map[FieldName]interface{}{"id": id}), "LIMIT 1")
	if err != nil {
		return false
	}
	if len(val) == 0 {
		return false
	}
	user_password_hash := val[0]["password"].(string)
	if user_password_hash == "" {
		return false
	}
	if system.HashPassword(password) == user_password_hash {
		return true
	}
	return false
}

func (udb *UserDB) Update(ctx context.Context, user *system.User) (*system.User, error) {
	timestamp := fmt.Sprint(time.Now().UTC())
	timestamp = timestamp[:len(timestamp)-9]
	userMap := map[FieldName]DBValue{
		"name":       user.Name,
		"email":      user.Email,
		"updated_at": timestamp,
	}
	if err := udb.p.Update(ctx, udb.userTable, userMap, NewWhereMap(map[FieldName]interface{}{"id": user.ID()})); err != nil {
		return nil, err
	}
	go udb.updateCache(ctx, user)
	return user, nil
}

func (udb *UserDB) UpdateUserPassword(ctx context.Context, user *system.User, new_password string) (*system.User, error) {
	timestamp := fmt.Sprint(time.Now().UTC())
	timestamp = timestamp[:len(timestamp)-9]
	userMap := map[FieldName]DBValue{
		"updated_at": timestamp,
		"password":   system.HashPassword(new_password),
	}
	if err := udb.p.Update(ctx, udb.userTable, userMap, NewWhereMap(map[FieldName]interface{}{"id": user.ID()})); err != nil {
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
	return udb.p.Delete(ctx, udb.userTable, NewWhereMap(map[FieldName]interface{}{"id": id}))
}

func (udb *UserDB) DeleteByName(ctx context.Context, name string) error {
	user, err := udb.GetByName(ctx, name)
	if err != nil {
		return ErrNoUser
	}
	go udb.clearCache(ctx, user)
	return udb.p.Delete(ctx, udb.userTable, NewWhereMap(map[FieldName]interface{}{"name": name}))
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
	if !udb.verifyPasswordById(ctx, user.ID(), password) {
		return nil, ErrPassMismatch
	}
	return user, nil
}

func (udb *UserDB) AuthenticateByEmail(ctx context.Context, email string, password string) (*system.User, error) {
	user, err := udb.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if !udb.verifyPasswordById(ctx, user.ID(), password) {
		return nil, ErrPassMismatch
	}
	return user, nil
}

func (udb *UserDB) GetFromCache(ctx context.Context, key interface{}) (*system.User, bool) {
	// if val, ok := usb.p.cache.Get(ctx, name); ok {
	// 	return val.(*system.User), nil
	// }
	// return nil, ErrNoUser
	if !udb.p.cache.Is_active() {
		return nil, false
	}
	if val, ok := udb.p.cache.Get(ctx, fmt.Sprint("user_", key)); ok {
		user := system.User{}
		if s, ok := val.(string); ok {
			if err := json.Unmarshal([]byte(s), &user); err != nil {
				udb.logger.Println(err)
				return nil, false
			}
		}
		return &user, true
	}
	return nil, false
}

func (udb *UserDB) updateCache(ctx context.Context, user *system.User) {
	if !udb.p.cache.Is_active() {
		return
	}
	udb.p.cache.Set(ctx, fmt.Sprint("user_", user.Name), user)
	udb.p.cache.Set(ctx, fmt.Sprint("user_", user.Email), user)
	udb.p.cache.Set(ctx, fmt.Sprint("user_", user.ID()), user)
}

func (udb *UserDB) clearCache(ctx context.Context, user *system.User) {
	if !udb.p.cache.Is_active() {
		return
	}
	udb.p.cache.Delete(ctx, fmt.Sprint("user_", user.Name))
	udb.p.cache.Delete(ctx, fmt.Sprint("user_", user.Email))
	udb.p.cache.Delete(ctx, fmt.Sprint("user_", user.ID()))
}
