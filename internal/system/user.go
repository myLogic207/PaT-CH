package system

import (
	"encoding/json"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TODO: Hash password
func EncryptPassword(password string) (string, error) {
	return hashPassword(password)
}

func hashPassword(password string) (string, error) {
	rawPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(rawPass), err
}

func CheckPasswords(hashedPassword string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

type User struct {
	id        int64     `json:"-"`
	Name      string    `json:"name" binding:"required"`
	Email     string    `json:"email" binding:"optional"`
	CreatedAt time.Time `json:"created_at" binding:"optional"`
	UpdatedAt time.Time `json:"updated_at" binding:"optional"`
}

type RawUser struct {
	Username string `json:"username"`
	// Email    string `json:"email"`
	Password string `json:"password"`
}

func NewUser(name string, email string) *User {
	return &User{
		Name:      name,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}
}

func LoadUser(id int64, name string, email string, created *time.Time, updated *time.Time) *User {
	user := &User{
		id:    id,
		Name:  name,
		Email: email,
	}

	if created != nil {
		user.CreatedAt = *created
	}

	if updated != nil {
		user.UpdatedAt = *updated
	}

	return user
}

func (u *User) ID() int64 {
	return u.id
}

func (u *User) SetID(id int64) {
	u.id = id
}

func (u User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}
