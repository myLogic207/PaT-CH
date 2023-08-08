package data

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

var ctx context.Context = context.Background()
var TEST_DB *DataBase

func nameExt() string {
	namExt, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		panic(err)
	}
	return namExt.String()
}

func tableName(name string) string {
	return fmt.Sprintf("%s_%s", name, nameExt())
}

func TestMain(m *testing.M) {
	var db_test_password string

	if file, ok := os.LookupEnv("PATCHTEST_DB_PASSWORD_FILE"); !ok {
		panic("PATCHTEST_DB_PASSWORD_FILE not set")
	} else if _, err := os.Stat(file); err != nil {
		panic(err)
	} else if raw_db_test_password, err := os.ReadFile(file); err != nil {
		panic(err)
	} else {
		db_test_password = strings.Trim(string(raw_db_test_password), "\r\n")
	}

	config := &DataConfig{
		Host:     "patch-test-6310.7tc.cockroachlabs.cloud",
		Port:     26257,
		User:     "patch_test",
		password: db_test_password,
		DBname:   "patch_db_test",
		SSLmode:  "verify-full",
	}
	db, err := NewConnectorWithConf(ctx, config)
	if err != nil {
		panic(err)
	}
	TEST_DB = db
	m.Run()
}

func TestTableInsert(t *testing.T) {
	tableName := tableName("test_table")
	if err := TEST_DB.CreateTable(ctx, tableName, []DBField{
		{"testKey", "text", 0, "PRIMARY KEY"},
		{"testValue", "text", 0, ""},
		{"testData", "text", 0, ""},
	}); err != nil {
		t.Error("error creating table: ", err)
		t.FailNow()
	}
	t.Log("created table")
	testDataSet := [][]interface{}{
		{"testK1", "testV1", "testD1"},
		{"testK2", "testV3", "testD4"},
		{"testK3", "testV3", "testD3"},
		{"testK4", "testV4", "testD4"},
	}
	if err := TEST_DB.Insert(ctx, tableName, []FieldName{"testKey", "testValue", "testData"}, testDataSet); err != nil {
		t.Error("error inserting into table: ", err)
		t.FailNow()
	}
	t.Log("inserted data")
	wm := NewWhereMap(map[FieldName]interface{}{"testKey": "testK3"})
	if val, err := TEST_DB.Select(ctx, tableName, []FieldName{"testKey", "TestData"}, wm, ""); err == nil {
		for _, v := range val {
			t.Log("row: ", v)
		}
	} else {
		t.Error("error getting values back from table: ", err)
		t.FailNow()
	}
	t.Log("got data back")
	if err := TEST_DB.DeleteTable(ctx, tableName); err == nil {
		t.Error("error: deleted full table")
		t.FailNow()
	} else if errors.Is(err, ErrTableNotEmpty) {
		t.Log("ok: table not empty")
	} else {
		t.Error("error: unexpected error: ", err)
		t.FailNow()
	}
	// clear table
	if err := TEST_DB.Delete(ctx, tableName, nil); err != nil {
		t.Error("error deleting from table: ", err)
		t.FailNow()
	}
	t.Log("cleared table")
	if err := TEST_DB.DeleteTable(ctx, tableName); err != nil {
		t.Error("error deleting table: ", err)
		t.FailNow()
	}
	t.Log("deleted table")
	t.Log("test complete")
}

func TestUpdate(t *testing.T) {
	tableName := tableName("test_table")
	if err := TEST_DB.CreateTable(ctx, tableName, []DBField{
		{"testKey", "text", 0, "PRIMARY KEY"},
		{"testValue", "text", 0, ""},
		{"testData", "text", 0, ""},
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("created table")
	testDataSet := [][]interface{}{
		{"testK1", "testV1", "testD1"},
	}
	if err := TEST_DB.Insert(ctx, tableName, []FieldName{"testKey", "testValue", "testData"}, testDataSet); err != nil {
		t.Error("error inserting into table: ", err)
		t.FailNow()
	}
	t.Log("inserted data")
	wm := NewWhereMap(map[FieldName]interface{}{"testKey": "testK1"})
	if err := TEST_DB.Update(ctx, tableName, map[FieldName]DBValue{
		"testValue": "testV2",
		"testData":  "testD2",
	}, wm); err != nil {
		t.Error("error updating table: ", err)
		t.FailNow()
	}
	t.Log("updated table")
	if val, err := TEST_DB.Select(ctx, tableName, []FieldName{"testKey", "testValue", "testData"}, wm, ""); err == nil {
		for _, v := range val {
			t.Log("row: ", v)
		}
	} else {
		t.Error("error getting values back from table: ", err)
		t.FailNow()
	}
	t.Log("got data back")
	if err := TEST_DB.Delete(ctx, tableName, nil); err != nil {
		t.Error("error deleting from table: ", err)
		t.FailNow()
	}
	t.Log("cleared table")
	if err := TEST_DB.DeleteTable(ctx, tableName); err != nil {
		t.Error("error deleting table: ", err)
		t.FailNow()
	}
	t.Log("deleted table")
	t.Log("test complete")
}

func TestTable(t *testing.T) {
	tableName := fmt.Sprintf("test_delete_%s", nameExt())
	if err := TEST_DB.CreateTable(ctx, tableName, []DBField{
		{"testKey", "text", 0, "PRIMARY KEY"},
		{"testValue", "text", 0, ""},
		{"testData", "text", 0, ""},
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("created table")
	if err := TEST_DB.DeleteTable(ctx, tableName); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("deleted table")
	// add insert, test if not delete_able, clear, delete
}
