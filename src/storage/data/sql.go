package data

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var logger = log.New(log.Writer(), "data: ", log.Flags())

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
		logger.Println("error prrof-selecting from table: ", err)
		return err
	}
	if row != nil {
		return errors.New("table not empty")
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

func (db *DataBase) buildSelect(table string, fields []string, whereMap map[string]interface{}, args string) string {
	sb := strings.Builder{}
	sb.WriteString("SELECT ")
	if len(fields) == 0 {
		sb.WriteString("*")
	} else {
		for i, v := range fields {
			fields[i] = strings.ToLower(v)
		}
		sb.WriteString(strings.Join(fields, ", "))
	}
	sb.WriteString(fmt.Sprintf(" FROM %s", table))
	if len(whereMap) > 0 {
		counter := 0
		for k, v := range whereMap {
			if counter != 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(fmt.Sprintf(" WHERE %s = '%s'", k, v))
			counter++
		}
	}
	if args != "" {
		sb.WriteString(fmt.Sprintf(" %s", args))
	}
	sb.WriteString(";")
	return sb.String()
}

func (db *DataBase) Select(ctx context.Context, table string, fields []string, whereMap map[string]interface{}, args string) ([][]interface{}, error) {
	logger.Println("selecting from table: ", table)
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly, DeferrableMode: pgx.NotDeferrable})
	if err != nil {
		return nil, err
	}
	result, err := db.getData(tx, ctx, table, fields, whereMap, args)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	err = tx.Commit(ctx)
	if err != nil {
		logger.Println("error committing transaction: ", err)
		return nil, err
	}
	return result, nil
}

func (db *DataBase) getData(tx pgx.Tx, ctx context.Context, table string, fields []string, whereMap map[string]interface{}, args string) ([][]interface{}, error) {
	queryString := db.buildSelect(table, fields, whereMap, args)
	fmt.Println(queryString)
	rows, err := tx.Query(ctx, queryString)
	if err != nil {
		return nil, err
	}
	// There might be a way to parse/Marhsal into a struct
	// dbFields := rows.FieldDescriptions()
	defer rows.Close()
	var result [][]interface{}
	for rows.Next() {
		row := make([]interface{}, len(fields))
		err = rows.Scan(row...)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
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
