package data

import (
	"errors"
	"testing"
	"time"

	"github.com/myLogic207/PaT-CH/internal/system"
)

func inTimeSpan(start, end, check time.Time) bool {
	start = start.Truncate(time.Microsecond)
	end = end.Truncate(time.Microsecond)
	// fmt.Printf("start:\t%v\ncheck:\t%v\nend:\t%v\n", start, check, end)
	return check.After(start) && check.Before(end)
}

func TestUserDB(t *testing.T) {
	testBeginTime := time.Now().UTC()
	tableName := tableName("test_user")
	TEST_DB.CreateTable(ctx, tableName, []DBField{
		{"id", "serial", 0},
		{"name", "text", 0},
		{"email", "text", 0},
		{"password", "text", 0},
		{"created_at", "timestamp", 0},
		{"updated_at", "timestamp", 0},
	}, DBConstraint{
		PrimaryKey:  []FieldName{"id"},
		ForeignKeys: nil,
	})
	defer TEST_DB.DeleteTable(ctx, tableName)
	TEST_DB.users.SetTableName(tableName)
	TEST_DB.users.Create(ctx, "test", "test@example.net", "test123")

	defer TEST_DB.Delete(ctx, tableName, nil)
	if err := testUserDBSelect(t, testBeginTime); err != nil {
		t.Error(err)
		t.FailNow()
	}

	user, err := testUserAuthenticate(t)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	user, err = testUserDBUpdate(t, user)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if err := TEST_DB.users.DeleteById(ctx, user.ID()); err != nil {
		t.Error(err)
		t.FailNow()
	}
	if _, err := TEST_DB.users.GetById(ctx, user.ID()); err == nil {
		t.Error("user not deleted")
		t.FailNow()
	}
}

func testUserDBSelect(t *testing.T, testBeginTime time.Time) error {
	user, err := TEST_DB.users.GetByName(ctx, "test")
	if err != nil {
		return err
	}
	if user.Name != "test" {
		return errors.New("name not set")
	}
	if user.Email != "test@example.net" {
		t.Error("email not set")
	}
	if !inTimeSpan(testBeginTime, time.Now().UTC(), user.CreatedAt) {
		t.Log("Now:", time.Now().UTC())
		t.Error("created not recently:", user.CreatedAt)
	}
	return nil
}

func testUserAuthenticate(t *testing.T) (*system.User, error) {
	user, err := TEST_DB.users.AuthenticateByName(ctx, "test", "test123")
	if err != nil {
		t.Error("authenticate failed:")
		return nil, err
	}
	return user, nil
}

func testUserDBUpdate(t *testing.T, user *system.User) (*system.User, error) {
	user.Name = "test2"
	user.Email = "example3@test.net"
	if _, err := TEST_DB.users.Update(ctx, user); err != nil {
		t.Log("\n")
		t.Error(err)
		t.Log("\n")
		t.FailNow()
	}
	user2, err := TEST_DB.users.GetById(ctx, user.ID())
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if user2.Name != "test2" {
		t.Error("name not updated")
		t.FailNow()
	}
	if user2.Email != "example3@test.net" {
		t.Error("email not updated")
		t.FailNow()
	}
	return user2, nil
}
