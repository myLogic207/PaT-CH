package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/myLogic207/PaT-CH/pkg/storage/cache"
	"github.com/myLogic207/PaT-CH/pkg/util"
	"gopkg.in/yaml.v3"
)

var (
	ErrTableNotEmpty   = errors.New("table is not empty")
	ErrOpenInitFile    = errors.New("error opening init file")
	ErrInitDB          = errors.New("error initializing database")
	ErrReadFile        = errors.New("error reading init file")
	ErrConnect         = errors.New("error connecting to database")
	ErrConnectRedis    = errors.New("error connecting to redis")
	ErrDBCreate        = errors.New("error creating table")
	ErrDBDrop          = errors.New("error dropping table")
	ErrDBInsert        = errors.New("error inserting into database")
	ErrDBSelect        = errors.New("error selecting from database")
	ErrDBUpdate        = errors.New("error updating database")
	ErrDBDelete        = errors.New("error deleting from database")
	ErrTxStart         = errors.New("error starting transaction")
	ErrTxCommit        = errors.New("error committing transaction")
	ErrTxRollback      = errors.New("error rolling back transaction")
	ErrInvalidField    = errors.New("error invalid field type")
	ErrNoArgs          = errors.New("error no arguments provided")
	ErrInvalidArg      = errors.New("error invalid argument type")
	ErrInvalidFunction = errors.New("error invalid db function type")
	ErrParseConfig     = errors.New("error parsing config")
	ErrBuildQuery      = errors.New("error building query")
	ErrParseArgs       = errors.New("error parsing arguments")
)

const (
	INSERT DBMethod = "INSERT"
	SELECT DBMethod = "SELECT"
	UPDATE DBMethod = "UPDATE"
	DELETE DBMethod = "DELETE"
	CREATE DBMethod = "CREATE TABLE"
	DROP   DBMethod = "DROP TABLE"
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
	config  *util.Config
	cache   *cache.RedisConnector
	Users   *UserDB
	logger  *log.Logger
}

var defaultConfig = map[string]interface{}{
	"Host":     "localhost",
	"Port":     "5432",
	"user":     "postgres",
	"password": "postgres",
	"DBname":   "postgres",
	"sslmode":  "disable",
	"MaxConns": "10",
	"UseCache": false,
	"initFile": "db.init.d",
}

func NewConnector(ctx context.Context, logger *log.Logger, config *util.Config) (*DataBase, error) {
	if logger == nil {
		logger = log.Default()
	}
	config.MergeDefault(defaultConfig)
	connString := buildConnString(config)
	if connString == "" {
		return nil, ErrParseConfig
	}
	pgxConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logger.Println(err)
		return nil, ErrParseConfig
	}
	db, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		logger.Println(err)
		return nil, ErrConnect
	}
	err = db.Ping(ctx)
	if err != nil {
		logger.Println(err)
		return nil, ErrConnect
	}
	dbConn := &DataBase{
		pool:    db,
		config:  config,
		context: ctx,
		logger:  logger,
	}
	dbConn.Users = NewUserDB(dbConn, "users", logger)
	dbConn.cache = cache.NewStubConnector()
	if redisConfig, ok := config.Get("redis").(*util.Config); ok && redisConfig != nil {
		if redisConfig.GetBool("use") {
			logger.Println("Using redis cache")
			if err := dbConn.SetCache(config); err != nil {
				return nil, err
			}
		}
	} else if !ok {
		return nil, errors.New("error parsing redis config")
	}

	logger.Println("connected to database")
	return dbConn, nil
}

func buildConnString(config *util.Config) string {
	stringBuffer := strings.Builder{}
	stringBuffer.WriteString("postgres://")

	if user, ok := config.GetString("User"); ok {
		stringBuffer.WriteString(user + ":")
	} else {
		return ""
	}

	if password, ok := config.GetString("Password"); ok {
		stringBuffer.WriteString(password + "@")
	} else {
		return ""
	}

	if host, ok := config.GetString("Host"); ok {
		stringBuffer.WriteString(host + ":")
	} else {
		return ""
	}

	if port, ok := config.GetString("Port"); ok {
		stringBuffer.WriteString(port + "/")
	} else {
		return ""
	}

	if dbName, ok := config.GetString("DBname"); ok {
		stringBuffer.WriteString(dbName + "?")
	} else {
		return ""
	}

	if sslMode, ok := config.GetString("sslmode"); ok {
		stringBuffer.WriteString("sslmode=" + sslMode + "&")
	}

	if poolMaxConns, ok := config.GetString("MaxConns"); ok {
		stringBuffer.WriteString("pool_max_conns=" + poolMaxConns)
	}

	return stringBuffer.String()
}

func (db *DataBase) SetCache(config *util.Config) error {
	redisConf, ok := config.Get("redis").(*util.Config)
	if !ok {
		db.logger.Println("No redis config provided")
		return nil
	}
	cache, err := cache.NewConnector(redisConf, db.logger)
	if err != nil {
		db.logger.Println(err)
		return ErrConnectRedis
	}
	db.logger.Println("connected to redis")
	db.cache = cache
	return nil
}

func (db *DataBase) Disconnect() error {
	db.pool.Close()
	return nil
}

func (db *DataBase) Init() error {
	db.logger.Println("initializing database")
	initFile, ok := db.config.GetString("initFile")
	if !ok {
		db.logger.Println("No init file provided")
		return nil
	}

	if fileStat, err := os.Stat(initFile); err != nil {
		db.logger.Println("Init file does not exist")
		return nil
	} else {
		if fileStat.IsDir() {
			db.logger.Println("Init file is a directory")
			return db.loadInitDir(initFile)
		}
		return db.loadInitFile(initFile)
	}
}

func (db *DataBase) loadInitDir(folderPath string) error {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		db.logger.Println(err)
		return ErrOpenInitFile
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, err := f.Info()
		if err != nil {
			db.logger.Println(err)
			continue
		}
		if err := db.loadInitFile(filepath.Join(folderPath, info.Name())); err != nil {
			db.logger.Println(err)
			return ErrInitDB
		}
	}
	return nil
}

func (db *DataBase) loadInitFile(file string) error {
	query, err := parseFile(file)
	if err != nil {
		db.logger.Println(err)
		return ErrInitDB
	}
	if os.Getenv("ENVIRONMENT") == "DEVELOPMENT" {
		db.logger.Println(strings.ReplaceAll(query, "\\", "\\\n"))

	}
	for _, q := range strings.Split(query, "\\") {
		db.logger.Println("Executing init query")
		if _, err = db.pool.Exec(db.context, q); err != nil {
			db.logger.Println(err)
			return ErrInitDB
		}
	}
	return nil
}

func parseFile(fileName string) (string, error) {
	split := strings.Split(fileName, ".")
	suffix := split[len(split)-1]
	var query, val string
	var err error
	switch suffix {
	case "sql":
		if rawQuery, err := os.ReadFile(fileName); err != nil {
			return "", err
		} else {
			query = string(rawQuery)
		}
	case "json":
		if val, err = read(fileName, json.Unmarshal); err == nil {
			query = val
		}
	case "yaml":
		if val, err = read(fileName, yaml.Unmarshal); err == nil {
			query = val
		}
	default:
		return "", ErrOpenInitFile
	}
	if err != nil {
		return "", err
	}
	return query, nil
}

func read(fileName string, parser func(data []byte, v any) error) (string, error) {
	raw, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	initStruct := &DBInit{}
	err = parser(raw, initStruct)
	if err != nil {
		return "", err
	}
	return initStruct.String(), nil
}

// DBFunction
func (db *DataBase) CreateTable(ctx context.Context, table string, fields []DBField) error {
	db.logger.Println("creating table:", table)
	_, err := db.transactionWrapper(ctx, CREATE, table, fields)
	if err != nil {
		db.logger.Println(err)
		return ErrDBCreate
	}
	return nil
}

func (db *DataBase) DeleteTable(ctx context.Context, table string) error {
	db.logger.Println("deleting table:", table)

	// selecting to check if table is empty
	row, err := db.Select(ctx, table, nil, nil, "LIMIT 1")
	if err != nil {
		db.logger.Println(err)
		return ErrDBSelect
	}
	if len(row) > 0 {
		return ErrTableNotEmpty
	}

	_, err = db.transactionWrapper(ctx, DROP, table)
	if err != nil {
		db.logger.Println(err)
		return ErrDBDrop
	}
	return nil
}

func (db *DataBase) Select(ctx context.Context, table string, fields []FieldName, wm *WhereMap, args string) (DBResult, error) {
	db.logger.Println("selecting from table:", table)
	result, err := db.transactionWrapper(ctx, SELECT, table, fields, wm, args)
	if err != nil {
		db.logger.Println(err)
		return nil, ErrDBSelect
	}
	// db.logger.Printf("got data: %+v\n", result)
	db.logger.Println("selected successfully")
	return result, nil
}

func (db *DataBase) Insert(ctx context.Context, table string, fields []FieldName, values [][]interface{}) error {
	db.logger.Println("inserting into table:", table)
	for _, v := range values {
		if len(v) != len(fields) {
			db.logger.Println("warning: number of values does not match number of fields:", len(v), "/", len(fields))
		}
	}
	_, err := db.transactionWrapper(ctx, INSERT, table, fields, values)
	if err != nil {
		db.logger.Println(err)
		return ErrDBInsert
	}
	db.logger.Println("inserted successfully")
	return nil
}

func (db *DataBase) Update(ctx context.Context, table string, updates map[FieldName]DBValue, wm *WhereMap) error {
	db.logger.Println("updating table:", table)
	_, err := db.transactionWrapper(ctx, UPDATE, table, updates, wm)
	if err != nil {
		db.logger.Println(err)
		return ErrDBUpdate
	}
	db.logger.Println("updated successfully")
	return nil
}

func (db *DataBase) Delete(ctx context.Context, table string, wm *WhereMap) error {
	db.logger.Println("deleting from table:", table)
	_, err := db.transactionWrapper(ctx, DELETE, table, wm)
	if err != nil {
		db.logger.Println(err)
		return ErrDBDelete
	}
	db.logger.Println("deleted successfully")
	return nil
}

// transactionWrapper
func (db *DataBase) transactionWrapper(ctx context.Context, dbFunction DBMethod, table string, args ...any) (DBResult, error) {
	db.logger.Printf("wrapping %s transaction", dbFunction)
	parsedArgs, err := parseArgs(dbFunction, args)
	if err != nil {
		db.logger.Println(err)
		if !errors.Is(err, ErrNoArgs) {
			return nil, ErrParseArgs
		}
		db.logger.Println("no arguments passed")
	}
	query, err := buildQuery(dbFunction, table, parsedArgs)
	if err != nil {
		db.logger.Println(err)
		return nil, ErrBuildQuery
	}
	if os.Getenv("ENVIRONMENT") == "development" {
		db.logger.Printf("query: %s", query)
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		db.logger.Println(err)
		return nil, ErrTxStart
	}
	var tag pgconn.CommandTag
	var rows DBResult
	switch dbFunction {
	case SELECT:
		db.logger.Println("getting data")
		rows, err = db.getData(ctx, tx, query)
	case INSERT:
		db.logger.Println("inserting data")
		err = db.insert(ctx, tx, table, parsedArgs.fields, parsedArgs.values)
	default:
		tag, err = tx.Exec(ctx, query)
		db.logger.Printf("executed query: %s", query)
		db.logger.Printf("command tag: %s", tag)
	}
	if err != nil {
		db.logger.Printf("error acting on table, rolling back;\n%s", err)
		if e := tx.Rollback(ctx); e != nil {
			db.logger.Println(e)
			return nil, ErrTxRollback
		}
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		db.logger.Println(err)
		return nil, ErrTxCommit
	}
	db.logger.Println("transaction committed")
	return rows, nil
}

func (db *DataBase) getData(ctx context.Context, tx pgx.Tx, query string) (DBResult, error) {
	rows, err := tx.Query(ctx, query)
	if err != nil {
		db.logger.Println(err)
		return nil, ErrDBSelect
	}

	dbFields := rows.FieldDescriptions()
	defer rows.Close()
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		row := make(map[string]interface{}, len(dbFields))
		val, err := rows.Values()
		if err != nil {
			db.logger.Println(err)
			return nil, ErrDBSelect
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
		db.logger.Println(err)
		return ErrDBInsert
	}
	if copyCount != int64(len(values)) {
		db.logger.Println("error inserting all values into table")
		return ErrDBInsert
	}
	db.logger.Printf("inserted %d rows into %s", copyCount, table)
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
		return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s;", buildCreate(strings.ToLower(table), args.newFields)), nil
	case DROP:
		return fmt.Sprintf("DROP TABLE IF EXISTS %s;", table), nil
	case DELETE:
		sb.WriteString(fmt.Sprint("DELETE FROM ", table))
		if args.wm != nil {
			sb.WriteString(fmt.Sprint(" WHERE ", args.wm.String()))
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
		sb.WriteString(fmt.Sprint(" FROM ", table))
		if args.wm != nil {
			sb.WriteString(" WHERE ")
			sb.WriteString(args.wm.String())
		}
		if args.args != "" {
			sb.WriteString(fmt.Sprint(" ", args.args))
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

func parseArgs(dbFunction DBMethod, args []any) (*QueryArgs, error) {
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

func buildCreate(name string, fields []DBField) string {
	table := DBTable{name, fields}
	return table.String()
}

func buildUpdate(values map[FieldName]DBValue) string {
	buffer := make([]string, len(values))
	counter := 0
	for k, v := range values {
		buffer[counter] = fmt.Sprintf("%s = '%s'", strings.ToLower(fmt.Sprint(k)), fmt.Sprint(v))
		counter++
	}
	return join(buffer, ", ")
}
