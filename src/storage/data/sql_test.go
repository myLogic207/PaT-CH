package data

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"testing"
)

var ctx context.Context = context.Background()
var TESTDB *DataBase

func TestMain(m *testing.M) {
	// config := DefaultConfig()
	config := &DataConfig{
		Host:     "patch-test-6310.7tc.cockroachlabs.cloud",
		Port:     26257,
		User:     "patch",
		password: "MDuKbW__xZs3guKrlK-AdA",
		DBname:   "patch_db_test",
		SSLmode:  "verify-full",
	}
	db, err := NewConnectorWithConf(ctx, config)
	if err != nil {
		panic(err)
	}
	TESTDB = db
	m.Run()
}

func TestTableInsert(t *testing.T) {
	namext, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	tableName := fmt.Sprintf("test_%s", namext)
	if err := TESTDB.CreateTable(ctx, tableName, []DBField{
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
	if err := TESTDB.Insert(ctx, tableName, []FieldName{"testKey", "testValue", "testData"}, testDataSet); err != nil {
		t.Error("error inserting into table: ", err)
		t.FailNow()
	}
	t.Log("inserted data")
	wm := NewWhereMap(map[FieldName]interface{}{"testKey": "testK3"})
	if val, err := TESTDB.Select(ctx, tableName, []FieldName{"testKey", "TestData"}, wm, ""); err == nil {
		for _, v := range val {
			t.Log("row: ", v)
		}
	} else {
		t.Error("error getting values back from table: ", err)
		t.FailNow()
	}
	t.Log("got data back")
	if err := TESTDB.DeleteTable(ctx, tableName); err == nil {
		t.Error("error: deleted full table")
		t.FailNow()
	} else if errors.Is(err, ErrTableNotEmpty) {
		t.Log("ok: table not empty")
	} else {
		t.Error("error: unexpected error: ", err)
		t.FailNow()
	}
	// clear table
	if err := TESTDB.Delete(ctx, tableName, nil); err != nil {
		t.Error("error deleting from table: ", err)
		t.FailNow()
	}
	t.Log("cleared table")
	if err := TESTDB.DeleteTable(ctx, tableName); err != nil {
		t.Error("error deleting table: ", err)
		t.FailNow()
	}
	t.Log("deleted table")
	t.Log("test complete")
}

func TestUpdate(t *testing.T) {
	ext, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	tableName := fmt.Sprintf("test_update_%s", ext)
	if err := TESTDB.CreateTable(ctx, tableName, []DBField{
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
	if err := TESTDB.Insert(ctx, tableName, []FieldName{"testKey", "testValue", "testData"}, testDataSet); err != nil {
		t.Error("error inserting into table: ", err)
		t.FailNow()
	}
	t.Log("inserted data")
	wm := NewWhereMap(map[FieldName]interface{}{"testKey": "testK1"})
	if err := TESTDB.Update(ctx, tableName, map[FieldName]DBValue{
		"testValue": "testV2",
		"testData":  "testD2",
	}, wm); err != nil {
		t.Error("error updating table: ", err)
		t.FailNow()
	}
	t.Log("updated table")
	if val, err := TESTDB.Select(ctx, tableName, []FieldName{"testKey", "testValue", "testData"}, wm, ""); err == nil {
		for _, v := range val {
			t.Log("row: ", v)
		}
	} else {
		t.Error("error getting values back from table: ", err)
		t.FailNow()
	}
	t.Log("got data back")
	if err := TESTDB.Delete(ctx, tableName, nil); err != nil {
		t.Error("error deleting from table: ", err)
		t.FailNow()
	}
	t.Log("cleared table")
	if err := TESTDB.DeleteTable(ctx, tableName); err != nil {
		t.Error("error deleting table: ", err)
		t.FailNow()
	}
	t.Log("deleted table")
	t.Log("test complete")
}

func TestTable(t *testing.T) {
	ext, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	tableName := fmt.Sprintf("test_delete_%s", ext)
	if err := TESTDB.CreateTable(ctx, tableName, []DBField{
		{"testKey", "text", 0, "PRIMARY KEY"},
		{"testValue", "text", 0, ""},
		{"testData", "text", 0, ""},
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("created table")
	if err := TESTDB.DeleteTable(ctx, tableName); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("deleted table")
	// add insert, test if not delete_able, clear, delete
}
