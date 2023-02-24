package data

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var logger = log.New(log.Writer(), "data: ", log.Flags())

var ErrTableNotEmpty = errors.New("table is not empty")

type DataConfig struct {
	host     string
	port     uint16
	user     string
	password string
	dbname   string
	sslmode  string
}

func DefaultConfig() *DataConfig {
	return &DataConfig{
		host:     "localhost",
		port:     5432,
		user:     "postgres",
		password: "",
		dbname:   "patch_db",
	}
}

func (c *DataConfig) init() error {
	if c.host == "" {
		c.host = "localhost"
	}
	if c.port == 0 {
		c.port = 5432
	}
	if c.user == "" {
		c.user = "postgres"
	}
	if c.dbname == "" {
		c.dbname = "patch_db"
	}
	return nil
}

func (c *DataConfig) ConnString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s", c.user, c.password, c.host, c.port, c.dbname, c.sslmode)
}

type DBField string
type DBValue interface {
	string | int | float64 | bool | []byte | time.Time | any
}

type DBResult []map[string]interface{}

type WhereMap struct {
	clauses map[DBField]any
}

func (m *WhereMap) String() string {
	if len(m.clauses) < 1 {
		return ""
	}
	if len(m.clauses) == 1 {
		for k, v := range m.clauses {
			return fmt.Sprintf("%s = '%s'", k, v)
		}
	}
	sb := strings.Builder{}
	counter := 0
	for k, v := range m.clauses {
		sb.WriteString(fmt.Sprintf("%s = '%s' AND", k, v))
		counter++
	}
	return sb.String()
}

func NewWhereMap(rawMap map[DBField]interface{}) *WhereMap {
	return &WhereMap{
		clauses: rawMap,
	}
}

func (m *WhereMap) Set(field DBField, value DBValue) {
	if m.clauses == nil {
		m.clauses = make(map[DBField]any)
	}
	m.clauses[field] = value
}

type DBfields struct {
	name       string
	typ        string
	len        int    // optional
	constraint string // optional
}

func (f *DBfields) String() string {
	sb := strings.Builder{}
	sb.WriteString(strings.ToLower(f.name))
	sb.WriteString(" ")
	sb.WriteString(f.typ)
	if f.len > 0 {
		sb.WriteString(fmt.Sprintf("(%d)", f.len))
	}
	if f.constraint != "" {
		sb.WriteString(" ")
		sb.WriteString(f.constraint)
	}
	return sb.String()
}

type DataBase struct {
	pool    *pgxpool.Pool
	context context.Context
	config  *DataConfig
}

type DBFunction func(ctx context.Context, tx pgx.Tx, table string, fields []DBField, whereMap *WhereMap, args string) (DBResult, error)

func NewDataBase(config *DataConfig, ctx context.Context) (*DataBase, error) {
	if err := config.init(); err != nil {
		return nil, err
	}
	// fmt.Println(config.ConnString())
	poolConfig, err := pgxpool.ParseConfig(config.ConnString())
	if err != nil {
		logger.Println("error parsing database config: ", err)
		return nil, err
	}
	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	err = db.Ping(ctx)
	if err != nil {
		logger.Println("error connecting to database: ", err)
		return nil, err
	}
	logger.Println("connected to database")
	return &DataBase{
		pool:    db,
		config:  config,
		context: ctx,
	}, nil
}

func (db *DataBase) Disconnect() error {
	db.pool.Close()
	return nil
}

func (db *DataBase) Init() error {
	logger.Println("initializing database")
	return nil
}

func buildCreate(table string, fields []DBfields) string {
	sb := strings.Builder{}
	sb.WriteString("CREATE TABLE IF NOT EXISTS ")
	sb.WriteString(table)
	sb.WriteString(" (")
	for i, f := range fields {
		sb.WriteString(f.String())
		if i < len(fields)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(");")
	return sb.String()
}

func (db *DataBase) CreateTable(ctx context.Context, table string, fields []DBfields) error {
	logger.Println("creating table: ", table)
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	createQuery := buildCreate(table, fields)
	_, err = tx.Exec(ctx, createQuery)
	if err != nil {
		logger.Println("error creating table: ", err)
		tx.Rollback(ctx)
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		logger.Println("error committing transaction: ", err)
		return err
	}
	return nil
}

func (db *DataBase) DeleteTable(ctx context.Context, table string) error {
	logger.Println("deleting table: ", table)
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}

	row, err := db.Select(ctx, table, nil, nil, "LIMIT 1")
	if err != nil {
		logger.Println("error proof-selecting from table: ", err)
		return err
	}
	if len(row) > 0 {
		return ErrTableNotEmpty
	}
	err = db.dropTable(ctx, tx, table)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		logger.Println("error committing transaction: ", err)
		return err
	}
	return nil
}

func (db *DataBase) dropTable(ctx context.Context, transaction pgx.Tx, table string) error {
	logger.Println("warn: dropping table: ", table)
	tx, err := transaction.Begin(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	if err != nil {
		err := tx.Rollback(ctx)
		if err != nil {
			logger.Println("error rolling back transaction: ", err)
		}
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		logger.Println("error committing transaction: ", err)
		return err
	}
	return nil
}

func (db *DataBase) Insert(ctx context.Context, table string, fields []string, values [][]interface{}) error {
	logger.Println("inserting into table: ", table)
	for _, v := range values {
		if len(v) != len(fields) {
			logger.Println("warning: number of values does not match number of fields: ", values[0])
		}
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	err = db.insert(ctx, tx, table, fields, values)
	if err != nil {
		logger.Printf("error inserting into table: %s\nrolling back\n", err)
		err := tx.Rollback(ctx)
		if err != nil {
			logger.Println("error rolling back transaction: ", err)
		}
		return err
	}
	err = tx.Commit(ctx)
	errors.Is(err, pgx.ErrTxCommitRollback)
	if err != nil {
		logger.Println("error committing transaction: ", err)
		return err
	}
	return nil
}

func (db *DataBase) insert(ctx context.Context, tx pgx.Tx, table string, fields []string, values [][]interface{}) error {
	for i, v := range fields {
		fields[i] = strings.ToLower(v)
	}
	copyCount, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{table},
		fields,
		pgx.CopyFromRows(values),
	)
	if err != nil {
		return err
	}
	logger.Printf("inserted %d rows into %s", copyCount, table)
	return nil
}

func (db *DataBase) buildSelect(table string, fields []DBField, wm *WhereMap, args string) string {
	sb := strings.Builder{}
	sb.WriteString("SELECT ")
	if len(fields) == 0 {
		sb.WriteString("*")
	} else if len(fields) == 1 {
		sb.WriteString(string(fields[0]))
	} else {
		sb.WriteString(join(fields, ", "))
	}
	sb.WriteString(fmt.Sprintf(" FROM %s", table))
	if wm != nil {
		sb.WriteString(" WHERE ")
		sb.WriteString(wm.String())
	}
	if args != "" {
		sb.WriteString(fmt.Sprintf(" %s", args))
	}
	sb.WriteString(";")
	return sb.String()
}

func (db *DataBase) Select(ctx context.Context, table string, fields []DBField, wm *WhereMap, args string) (DBResult, error) {
	logger.Println("selecting from table: ", table)
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly, DeferrableMode: pgx.NotDeferrable})
	if err != nil {
		return nil, err
	}
	result, err := db.getData(tx, ctx, table, fields, wm, args)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	err = tx.Commit(ctx)
	if err != nil {
		logger.Println("error committing transaction: ", err)
		return nil, err
	}
	fmt.Printf("got data: %+v\n", result)
	return result, nil
}

func (db *DataBase) getData(tx pgx.Tx, ctx context.Context, table string, fields []DBField, wm *WhereMap, args string) (DBResult, error) {
	queryString := db.buildSelect(table, fields, wm, args)
	rows, err := tx.Query(ctx, queryString)
	if err != nil {
		return nil, err
	}
	// There might be a way to parse/Marhsal into a struct
	dbFields := rows.FieldDescriptions()
	defer rows.Close()
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		row := make(map[string]interface{}, len(fields))
		val, err := rows.Values()
		if err != nil {
			return nil, err
		}
		for i, v := range val {
			row[dbFields[i].Name] = v
		}
		result = append(result, row)
	}
	return result, nil
}

func (db *DataBase) transactionWrapper(ctx context.Context, dbFunction DBFunction, table string, fields []DBField, wm *WhereMap, args string) (DBResult, error) {
	logger.Println("wrapping transaction")
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	val, err := dbFunction(ctx, tx, table, fields, wm, args)
	if err != nil {
		logger.Printf("error acting on table: %s\nrolling back\n", err)
		err := tx.Rollback(ctx)
		if err != nil {
			logger.Println("error rolling back transaction: ", err)
		}
		return nil, err
	}
	err = tx.Commit(ctx)
	if err != nil {
		logger.Println("error committing transaction: ", err)
		return nil, err
	}
	return val, nil
}

func (db *DataBase) Delete(ctx context.Context, table string, wm *WhereMap) error {
	logger.Println("deleting from table: ", table)
	_, err := db.transactionWrapper(ctx, db.delete, table, nil, wm, "")
	if err != nil {
		return err
	}
	logger.Println("deleted successfully")
	return nil
}

func (db *DataBase) delete(ctx context.Context, tx pgx.Tx, table string, fields []DBField, wm *WhereMap, args string) (DBResult, error) {
	queryString := db.buildDelete(table, fields, wm, args)
	logger.Println("deleting with query: ", queryString)
	_, err := tx.Exec(ctx, queryString)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (db *DataBase) buildDelete(table string, fields []DBField, wm *WhereMap, args string) string {
	sb := strings.Builder{}
	sb.WriteString("DELETE FROM ")
	sb.WriteString(table)
	if wm != nil {
		sb.WriteString(" WHERE ")
		sb.WriteString(wm.String())
	}
	sb.WriteString(";")
	return sb.String()
}

func join[K any](a []K, sep string) string {
	sb := strings.Builder{}
	for i, v := range a {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(fmt.Sprint(v))
	}
	return sb.String()
}
