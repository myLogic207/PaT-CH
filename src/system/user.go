package system

import (
	"log"
	"time"
)

// TODO: Hash password
func HashPassword(password string) string {
	return password
}

type User struct {
	id        int64
	Name      string    `json:"name" binding:"required"`
	Email     string    `json:"email" binding:"optional"`
	CreatedAt time.Time `json:"created_at" binding:"optional"`
	UpdatedAt string    `json:"updated_at" binding:"optional"`
	password  string
}

func NewUser(name, email, password string) *User {
	return &User{
		Name:      name,
		Email:     email,
		CreatedAt: time.Now().UTC(),
		password:  HashPassword(password),
	}
}

func (u *User) ID() int64 {
	return u.id
}

func (u *User) SetID(id int64) {
	u.id = id
}

func (u *User) CheckPassword(password string) bool {
	return HashPassword(password) == u.password
}

func (u *User) SetPassword(password string) {
	log.Println("warn! User.SetPassword() called")
	u.password = HashPassword(password)
}

func (u *User) Password() string {
	log.Println("warn! User.Password() called")
	return u.password
}
