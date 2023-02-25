package data

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	logger           = log.New(log.Writer(), "data: ", log.Flags())
	ErrTableNotEmpty = errors.New("table is not empty")
)

const (
	INSERT DBMethod = "INSERT"
	SELECT DBMethod = "SELECT"
	UPDATE DBMethod = "UPDATE"
	DELETE DBMethod = "DELETE"
	CREATE DBMethod = "CREATETABLE"
	DROP   DBMethod = "DROPTABLE"
)

type (
	FieldName string
	DBMethod  string
	DBResult  []map[string]interface{}
	DBValue   interface {
		string | int | float64 | bool | []byte | time.Time | any
	}
)

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
	dbConn := &DataBase{
		pool:    db,
		config:  config,
		context: ctx,
	}
	return dbConn, nil
}

func (db *DataBase) Disconnect() error {
	db.pool.Close()
	return nil
}

func (db *DataBase) Init() error {
	logger.Println("initializing database")
	return nil
}

// DBFunction
func (db *DataBase) CreateTable(ctx context.Context, table string, fields []DBField) error {
	logger.Println("creating table: ", table)
	_, err := db.transactionWrapper(ctx, CREATE, table, fields)
	if err != nil {
		logger.Println(err)
		return errors.New("error creating table: " + table)
	}
	return nil
}

func (db *DataBase) DeleteTable(ctx context.Context, table string) error {
	logger.Println("deleting table: ", table)

	// selecting to check if table is empty
	row, err := db.Select(ctx, table, nil, nil, "LIMIT 1")
	if err != nil {
		logger.Println("error proof-selecting from table: ", err)
		return err
	}
	if len(row) > 0 {
		return ErrTableNotEmpty
	}

	_, err = db.transactionWrapper(ctx, DROP, table)
	if err != nil {
		logger.Println(err)
		return errors.New("error deleting table: " + table)
	}
	return nil
}

func (db *DataBase) Select(ctx context.Context, table string, fields []FieldName, wm *WhereMap, args string) (DBResult, error) {
	logger.Println("selecting from table: ", table)
	result, err := db.transactionWrapper(ctx, SELECT, table, fields, wm, args)
	if err != nil {
		logger.Println(err)
		return nil, errors.New("error selecting from table: " + table)
	}
	fmt.Printf("got data: %+v\n", result)
	return result, nil
}

func (db *DataBase) Insert(ctx context.Context, table string, fields []FieldName, values [][]interface{}) error {
	logger.Println("inserting into table: ", table)
	for _, v := range values {
		if len(v) != len(fields) {
			logger.Println("warning: number of values does not match number of fields: ", values[0])
		}
	}
	_, err := db.transactionWrapper(ctx, INSERT, table, fields, values)
	if err != nil {
		logger.Println(err)
		return errors.New("error inserting into table: " + table)
	}
	return nil
}

func (db *DataBase) Update(ctx context.Context, table string, updates map[FieldName]DBValue, wm *WhereMap) error {
	logger.Println("updating table: ", table)
	_, err := db.transactionWrapper(ctx, UPDATE, table, updates, wm)
	if err != nil {
		return err
	}
	logger.Println("updated successfully")
	return nil
}

func (db *DataBase) Delete(ctx context.Context, table string, wm *WhereMap) error {
	logger.Println("deleting from table: ", table)
	_, err := db.transactionWrapper(ctx, DELETE, table, wm)
	if err != nil {
		return err
	}
	logger.Println("deleted successfully")
	return nil
}

// transactionWrapper
func (db *DataBase) transactionWrapper(ctx context.Context, dbFunction DBMethod, table string, args ...any) (DBResult, error) {
	logger.Printf("wrapping %s transaction", dbFunction)
	parsedArgs, err := parseArgs(dbFunction, args)
	if err != nil {
		logger.Println(err)
		if !errors.Is(err, ErrNoArgs) {
			return nil, errors.New("error parsing arguments")
		}
		logger.Println("no arguments passed")
	}
	query, err := buildQuery(dbFunction, table, parsedArgs)
	if err != nil {
		logger.Println(err)
		return nil, errors.New("error building query")
	}
	if os.Getenv("ENVIROMENT") == "development" {
		logger.Printf("query: %s", query)
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		logger.Println(err)
		return nil, errors.New("error starting transaction")
	}
	var tag pgconn.CommandTag
	var rows DBResult
	switch dbFunction {
	case SELECT:
		logger.Println("getting data")
		rows, err = db.getData(ctx, tx, query)
	case INSERT:
		logger.Println("inserting data")
		err = db.insert(ctx, tx, table, parsedArgs.fields, parsedArgs.values)
	default:
		tag, err = tx.Exec(ctx, query)
		logger.Printf("executed query: %s", query)
		logger.Printf("command tag: %s", tag)
	}
	if err != nil {
		logger.Printf("error acting on table, rolling back;\n%s", err)
		if e := tx.Rollback(ctx); e != nil {
			logger.Println("error rolling back transaction: ", e)
			return nil, e
		}
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		logger.Println("error committing transaction: ", err)
		return nil, err
	}
	logger.Println("transaction committed")
	return rows, nil
}

func (db *DataBase) getData(ctx context.Context, tx pgx.Tx, query string) (DBResult, error) {
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	dbFields := rows.FieldDescriptions()
	defer rows.Close()
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		row := make(map[string]interface{}, len(dbFields))
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

func (db *DataBase) insert(ctx context.Context, tx pgx.Tx, table string, fields []FieldName, values [][]interface{}) error {
	dbFields := make([]string, len(fields))
	for i, v := range fields {
		dbFields[i] = strings.ToLower(string(v))
	}
	copyCount, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{table},
		dbFields,
		pgx.CopyFromRows(values),
	)
	if err != nil {
		return err
	}
	if copyCount != int64(len(values)) {
		return errors.New("error inserting all values into table: " + table)
	}
	logger.Printf("inserted %d rows into %s", copyCount, table)
	return nil
}

// query builder
type QueryArgs struct {
	fields    []FieldName
	newFields []DBField
	wm        *WhereMap
	args      string
	updates   map[FieldName]DBValue
	values    [][]interface{}
}

// Argument parser and query builder

func buildQuery(dbFunction DBMethod, table string, args *QueryArgs) (string, error) {
	sb := strings.Builder{}
	switch dbFunction {
	case CREATE:
		return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", strings.ToLower(table), buildCreate(args.newFields)), nil
	case DROP:
		return fmt.Sprintf("DROP TABLE IF EXISTS %s;", table), nil
	case DELETE:
		sb.WriteString(fmt.Sprintf("DELETE FROM %s ", table))
		if args.wm != nil {
			sb.WriteString(args.wm.String())
		}
	case SELECT:
		sb.WriteString("SELECT ")
		if len(args.fields) == 0 {
			sb.WriteString("*")
		} else {
			for i, v := range args.fields {
				sb.WriteString(strings.ToLower(string(v)))
				if i != len(args.fields)-1 {
					sb.WriteString(", ")
				}
			}
		}
		sb.WriteString(fmt.Sprintf(" FROM %s ", table))
		if args.wm != nil {
			sb.WriteString("WHERE ")
			sb.WriteString(args.wm.String())
		}
		if args.args != "" {
			sb.WriteString(args.args)
		}
	case UPDATE:
		sb.WriteString(fmt.Sprintf("UPDATE %s SET ", table))
		sb.WriteString(buildUpdate(args.updates))
		if args.wm != nil {
			sb.WriteString(" WHERE ")
			sb.WriteString(args.wm.String())
		}
	case INSERT:
		return "", nil
	}
	sb.WriteString(";")
	return sb.String(), nil
}

var (
	ErrInvalidField    = errors.New("invalid field type")
	ErrNoArgs          = errors.New("no arguments provided")
	ErrInvalidArg      = errors.New("invalid argument type")
	ErrInvalidFunction = errors.New("invalid db function type")
)

func parseArgs(dbFunction DBMethod, args []any) (*QueryArgs, error) {
	logger.Println("parsing args")
	if len(args) == 0 {
		return nil, ErrNoArgs
	}

	switch dbFunction {
	case UPDATE:
		// INSERT INTO table (fields) VALUES (values)
		updates, uOk := args[0].(map[FieldName]DBValue)
		wm, wOk := args[1].(*WhereMap)
		if !uOk || !wOk {
			return nil, ErrInvalidArg
		}
		return &QueryArgs{
			updates: updates,
			wm:      wm,
		}, nil
	case CREATE:
		// CREATE TABLE IF NOT EXISTS table (fields)
		if val, ok := args[0].([]DBField); ok {
			return &QueryArgs{
				newFields: val,
			}, nil
		}
		return nil, ErrInvalidArg
	case DROP:
		// DROP TABLE IF EXISTS table
		return nil, nil
	case DELETE:
		// DELETE FROM table WHERE statements
		if val, ok := args[0].(*WhereMap); ok {
			return &QueryArgs{
				wm: val,
			}, nil
		}
		return nil, nil
	case SELECT:
		// SELECT fields FROM table WHERE statements PARAMS
		fields, fOk := args[0].([]FieldName)
		wm, wOk := args[1].(*WhereMap)
		if !fOk || !wOk {
			return nil, ErrInvalidArg
		}
		if len(args) > 2 {
			if val, ok := args[2].(string); ok {
				return &QueryArgs{
					fields: fields,
					wm:     wm,
					args:   val,
				}, nil
			}
		}
		return &QueryArgs{
			fields: fields,
			wm:     wm,
		}, nil
	case INSERT:
		// Insert is done via COPY
		fields, fOk := args[0].([]FieldName)
		values, vOk := args[1].([][]interface{})
		if !fOk || !vOk {
			return nil, ErrInvalidArg
		}
		return &QueryArgs{
			fields: fields,
			values: values,
		}, nil

	}
	return nil, ErrInvalidFunction
}

func buildCreate(fields []DBField) string {
	if len(fields) == 1 {
		return fields[0].String()
	}
	sb := strings.Builder{}
	for i, f := range fields {
		sb.WriteString(f.String())
		if i < len(fields)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String()
}

func buildUpdate(values map[FieldName]DBValue) string {
	buffer := make([]string, len(values))
	counter := 0
	for k, v := range values {
		buffer[counter] = fmt.Sprintf("%s = '%s'", strings.ToLower(string(k)), v)
		counter++
	}
	return join(buffer, ", ")
}

// helper

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
