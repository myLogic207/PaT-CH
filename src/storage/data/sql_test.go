package data

import (
	"context"
	"errors"
	"testing"
)

var ctx context.Context = context.Background()
var TESTDB *DataBase

func TestMain(m *testing.M) {
	// config := DefaultConfig()
	config := &DataConfig{
		host:     "patch-test-6310.7tc.cockroachlabs.cloud",
		port:     26257,
		user:     "patch",
		password: "MDuKbW__xZs3guKrlK-AdA",
		dbname:   "patch_db",
		sslmode:  "verify-full",
	}
	db, err := NewDataBase(config, ctx)
	if err != nil {
		panic(err)
	}
	TESTDB = db
	m.Run()
}

func TestEverything(t *testing.T) {
	if err := TESTDB.CreateTable(ctx, "test", []DBfields{
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
	if err := TESTDB.Insert(ctx, "test", []string{"testKey", "testValue", "testData"}, testDataSet); err != nil {
		t.Error("error inserting into table: ", err)
		t.FailNow()
	}
	t.Log("inserted data")
	wm := NewWhereMap(map[DBField]interface{}{"testKey": "testK3"})
	if val, err := TESTDB.Select(ctx, "test", []DBField{"testKey", "TestData"}, wm, ""); err == nil {
		for _, v := range val {
			t.Log("row: ", v)
		}
	} else {
		t.Error("error getting values back from table: ", err)
		t.FailNow()
	}
	t.Log("got data back")
	if err := TESTDB.DeleteTable(ctx, "test"); err == nil {
		t.Error("error: deleted full table")
		t.FailNow()
	} else if errors.Is(err, ErrTableNotEmpty) {
		t.Log("ok: table not empty")
	} else {
		t.Error("error: unexpected error: ", err)
		t.FailNow()
	}
	// clear table
	if err := TESTDB.Delete(ctx, "test", nil); err != nil {
		t.Error("error deleting from table: ", err)
		t.FailNow()
	}
	t.Log("cleared table")
	if err := TESTDB.DeleteTable(ctx, "test"); err != nil {
		t.Error("error deleting table: ", err)
		t.FailNow()
	}
	t.Log("deleted table")
	t.Log("test complete")
}

func TestTable(t *testing.T) {
	if err := TESTDB.CreateTable(ctx, "delete_test", []DBfields{
		{"testKey", "text", 0, "PRIMARY KEY"},
		{"testValue", "text", 0, ""},
		{"testData", "text", 0, ""},
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("created table")
	if err := TESTDB.DeleteTable(ctx, "delete_test"); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("deleted table")
}
