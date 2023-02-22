package data

import (
	"context"
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

func TestInsertGet(t *testing.T) {
	if err := TESTDB.CreateTable(ctx, "test", []DBfields{
		{"testKey", "text", 0, "PRIMARY KEY"},
		{"testValue", "text", 0, ""},
		{"testData", "text", 0, ""},
	}); err != nil {
		t.Error("error creating table: ", err)
		t.FailNow()
	}
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

	if val, err := TESTDB.Select(ctx, "test", []string{"testKey", "TestData"}, map[string]interface{}{"testKey": "testK3"}, ""); err == nil {
		t.Log(val)
	} else {
		t.Error("error getting values back from table: ", err)
		t.FailNow()
	}
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
