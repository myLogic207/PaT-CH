package data

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/myLogic207/PaT-CH/pkg/util"
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

	if file, ok := os.LookupEnv("PATCHTESTDB_PASSWORD_FILE"); !ok {
		panic("PATCHTESTDB_PASSWORD_FILE not set")
	} else if _, err := os.Stat(file); err != nil {
		panic(err)
	} else if raw_db_test_password, err := os.ReadFile(file); err != nil {
		panic(err)
	} else {
		db_test_password = strings.Trim(string(raw_db_test_password), "\r\n")
	}

	config := util.NewConfig(map[string]interface{}{
		"Host":     "patch-test-6310.7tc.cockroachlabs.cloud",
		"Port":     26257,
		"User":     "patch_test",
		"password": db_test_password,
		"DBname":   "patch_db_test",
		"SSLmode":  "verify-full",
	}, nil)
	db, err := NewConnector(ctx, nil, config)
	if err != nil {
		panic(err)
	}
	TEST_DB = db
	m.Run()
}

func TestTableInsert(t *testing.T) {
	tableName := tableName("test_table")
	if err := TEST_DB.CreateTable(ctx, tableName, []DBField{
		{"testKey", "text", 0},
		{"testValue", "text", 0},
		{"testData", "text", 0},
	}, DBConstraint{
		PrimaryKey:  []FieldName{"testKey"},
		ForeignKeys: nil,
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
	if val := TEST_DB.Select(ctx, tableName, []string{"testKey", "TestData"}, wm, ""); len(val) > 0 {
		for _, v := range val {
			t.Log("row: ", v)
		}
	} else {
		t.Error("error getting values back from table")
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
		{"testKey", "text", 0},
		{"testValue", "text", 0},
		{"testData", "text", 0},
	}, DBConstraint{
		PrimaryKey:  []FieldName{"testKey"},
		ForeignKeys: nil,
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
	if val := TEST_DB.Select(ctx, tableName, []string{"testKey", "testValue", "testData"}, wm, ""); len(val) > 0 {
		for _, v := range val {
			t.Log("row: ", v)
		}
	} else {
		t.Error("error getting values back from table")
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
		{"testKey", "text", 0},
		{"testValue", "text", 0},
		{"testData", "text", 0},
	}, DBConstraint{
		PrimaryKey:  []FieldName{"testKey"},
		ForeignKeys: nil,
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

func TestForeignConstraint(t *testing.T) {
	tableName1 := fmt.Sprintf("test_foreign_%s", nameExt())
	tableName2 := fmt.Sprintf("test_foreign_%s", nameExt())
	if err := TEST_DB.CreateTable(ctx, tableName1, []DBField{
		{"testKey", "text", 0},
		{"testValue", "text", 0},
	}, DBConstraint{
		PrimaryKey:  []FieldName{"testKey"},
		ForeignKeys: nil,
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("created table1")

	if err := TEST_DB.CreateTable(ctx, tableName2, []DBField{
		{"testKey", "text", 0},
		{"testForeign", "text", 0},
		{"testData", "text", 0},
	}, DBConstraint{
		PrimaryKey: []FieldName{"testKey"},
		ForeignKeys: []DBForeignConstraint{
			{
				Fields: []FieldName{"testForeign"},
				Reference: DBConstraintReference{
					Table:  tableName1,
					Fields: []FieldName{"testKey"},
				},
			},
		}, // foreign key to itself
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("created table2")

	testDataSet1 := [][]interface{}{
		{"testK1", "testV1"},
		{"testK2", "testV2"},
	}

	if err := TEST_DB.Insert(ctx, tableName1, []FieldName{"testKey", "testValue"}, testDataSet1); err != nil {
		t.Error("error inserting into table: ", err)
		t.FailNow()
	}

	testDataSet2 := [][]interface{}{
		{"testK1", "testK1", "testD1"},
		{"testK2", "testK1", "testD2"},
	}

	if err := TEST_DB.Insert(ctx, tableName2, []FieldName{"testKey", "testForeign", "testData"}, testDataSet2); err != nil {
		t.Error("error inserting into table: ", err)
		t.FailNow()
	}

	// test if they connected

	// clear tables

	// drop tables in order to not violate foreign key constraints
	if err := TEST_DB.Delete(ctx, tableName2, nil); err != nil {
		t.Error("error deleting from table: ", err)
		t.FailNow()
	}

	if err := TEST_DB.Delete(ctx, tableName1, nil); err != nil {
		t.Error("error deleting from table: ", err)
		t.FailNow()
	}

	if err := TEST_DB.DeleteTable(ctx, tableName2); err != nil {
		t.Error(err)
		t.FailNow()
	}

	if err := TEST_DB.DeleteTable(ctx, tableName1); err != nil {
		t.Error(err)
		t.FailNow()
	}
}
