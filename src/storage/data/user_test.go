package data

import (
	"fmt"
	"testing"
	"time"
)

func inTimeSpan(start, end, check time.Time) bool {
	start = start.Truncate(time.Microsecond)
	end = end.Truncate(time.Microsecond)
	// fmt.Printf("start:\t%v\ncheck:\t%v\nend:\t%v\n", start, check, end)
	return check.Compare(start) <= 0 && check.Before(end)
}

func TestUserDB(t *testing.T) {
	tablename := tableName("test_user")
	TESTDB.CreateTable(ctx, tablename, []DBField{
		{"id", "serial", 0, "PRIMARY KEY"},
		{"name", "text", 0, ""},
		{"email", "text", 0, ""},
		{"password", "text", 0, ""},
		{"created_at", "timestamp", 0, ""},
		{"updated_at", "timestamp", 0, ""},
	})
	defer TESTDB.DeleteTable(ctx, tablename)
	TESTDB.Users.SetTableName(tablename)
	before := time.Now()
	TESTDB.Users.Create(ctx, "test", "test@example.net", "test123")
	all, err := TESTDB.Select(ctx, tablename, []FieldName{"*"}, nil, "")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	fmt.Printf("all: %+v", all)
	defer TESTDB.Delete(ctx, tablename, nil)
	user, err := TESTDB.Users.GetByName(ctx, "test")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if user.Name != "test" {
		t.Error("name not set")
		t.FailNow()
	}
	if user.Email != "test@example.net" {
		t.Error("email not set")
		t.FailNow()
	}
	if user.Password() != "test123" {
		t.Error("password not set")
		t.FailNow()
	}
	if !inTimeSpan(before, time.Now(), user.CreatedAt) {
		t.Error("created not recently")
		t.FailNow()
	}
	updateDate := user.updatedAt
	user.Name = "test2"
	user.Email = "example3@test.net"
	user.SetPassword("abcd1234")
	if err := TESTDB.Users.Update(ctx, user); err != nil {
		t.Log("\n")
		t.Error(err)
		t.Log("\n")
		t.FailNow()
	}
	user2, err := TESTDB.Users.GetById(ctx, user.id)
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
	if user2.Password() != "abcd1234" {
		t.Error("password updated")
		t.FailNow()
	}
	if user2.updatedAt == updateDate {
		t.Error("updated not updated")
		t.FailNow()
	}
	if err := TESTDB.Users.DeleteById(ctx, user.id); err != nil {
		t.Error(err)
		t.FailNow()
	}
	if _, err := TESTDB.Users.GetById(ctx, user.id); err == nil {
		t.Error("user not deleted")
		t.FailNow()
	}
}
