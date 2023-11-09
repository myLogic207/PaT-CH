package system

import (
	"context"
	"os"
	"testing"
)

var USERDB *UserIMDB

func TestMain(m *testing.M) {
	USERDB = NewUserIMDB()
	code := m.Run()
	// teardown
	USERDB = nil
	os.Exit(code)
}

func TestUserCRUD(t *testing.T) {
	userPass := "Password123"
	userName := "test"
	ctx := context.TODO()
	// Create
	testUser, err := USERDB.Create(ctx, userName, "test@example.net", userPass)
	if err != nil {
		t.Error(err)
	}
	// Read
	user, err := USERDB.GetByName(ctx, testUser.Name)
	if err != nil {
		t.Error(err)
	}
	if user.Name != testUser.Name {
		t.Error("User name mismatch")
	}
	if user.Email != testUser.Email {
		t.Error("User email mismatch")
	}
	// Update
	newMail := "newMail@test.org"
	user.Email = newMail
	_, err = USERDB.Update(ctx, user)
	if err != nil {
		t.Error(err)
	}
	// Read
	user, err = USERDB.GetById(ctx, testUser.ID())
	if err != nil {
		t.Error(err)
	}
	if user.Email != newMail {
		t.Error("User email mismatch")
	}
	// Delete
	err = USERDB.DeleteByName(ctx, testUser.Name)
	if err != nil {
		t.Error(err)
	}
}

func TestUserAuth(t *testing.T) {
	userPass := "Password123"
	userName := "test"
	ctx := context.TODO()
	// Create
	testUser, err := USERDB.Create(ctx, userName, "example@test.net", userPass)
	if err != nil {
		t.Error(err)
	}
	// Authenticate
	user, err := USERDB.Authenticate(ctx, userName, userPass)
	if err != nil {
		t.Error(err)
	}
	if user.Name != testUser.Name {
		t.Error("User name mismatch")
	}
	if user.Email != testUser.Email {
		t.Error("User email mismatch")
	}
	// Delete
	err = USERDB.DeleteByName(ctx, testUser.Name)
	if err != nil {
		t.Error(err)
	}
}
