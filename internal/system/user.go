package system

import (
	"encoding/json"
	"time"
)

// TODO: Hash password
func HashPassword(password string) string {
	return password
}

type User struct {
	id        int64     `json:"-"`
	Name      string    `json:"name" binding:"required"`
	Email     string    `json:"email" binding:"optional"`
	CreatedAt time.Time `json:"created_at" binding:"optional"`
	UpdatedAt time.Time `json:"updated_at" binding:"optional"`
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
