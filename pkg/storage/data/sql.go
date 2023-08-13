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
	users   *UserDB
	logger  *log.Logger
}

var defaultConfig = map[string]interface{}{
	"Host":      "localhost",
	"Port":      "5432",
	"user":      "postgres",
	"password":  "postgres",
	"DBname":    "postgres",
	"sslmode":   "disable",
	"MaxConns":  "10",
	"redis.use": false,
	"initFile":  "db.init.d",
}

func NewConnector(ctx context.Context, logger *log.Logger, config *util.Config) (*DataBase, error) {
	if logger == nil {
		logger = log.Default()
	}

	if config == nil {
		config = util.NewConfig(defaultConfig, logger)
	} else {
		if err := config.MergeDefault(defaultConfig); err != nil {
			logger.Println(err)
			return nil, ErrParseConfig
		}
	}

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
	logger.Println("pinging database...")
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
	dbConn.users = NewUserDB(dbConn, "users", "shadow", logger)
	if redisConfig, ok := config.Get("redis").(*util.Config); ok && redisConfig != nil {
		dbConn.cache, err = setupRedisConnector(redisConfig, logger)
		if err != nil {
			return nil, ErrConnectRedis
		}
	} else if !ok {
		return nil, errors.New("error parsing redis config")
	}

	dbConnConfig := db.Config().ConnConfig.Copy()
	logger.Println("connected to database", dbConnConfig.Database, "on", dbConnConfig.Host, ":", dbConnConfig.Port, "with user", dbConnConfig.User)
	return dbConn, nil
}

func setupRedisConnector(redisConfig *util.Config, logger *log.Logger) (*cache.RedisConnector, error) {
	if !redisConfig.GetBool("use") {
		return cache.NewStubConnector(), nil
	}
	logger.Println("using redis cache")
	return cache.NewConnector(redisConfig, logger)
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

	if dbName, ok := config.GetString("name"); ok {
		stringBuffer.WriteString(dbName + "?")
	} else {
		return ""
	}

	if sslMode, ok := config.GetString("sslmode"); ok {
		stringBuffer.WriteString("sslmode=" + sslMode)
	}

	if poolMaxConns, ok := config.GetString("MaxConns"); ok {
		stringBuffer.WriteString("&pool_max_conns=" + poolMaxConns)
	}

	return stringBuffer.String()
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
	db.logger.Println("Executing init query")
	result, err := db.transactionWrapper(db.context, query)
	if err != nil {
		db.logger.Println(err)
		return ErrInitDB
	}
	if result != nil {
		db.logger.Println(result)
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
func (db *DataBase) CreateTable(ctx context.Context, table string, fields []DBField, constraints DBConstraint) error {
	db.logger.Println("creating table:", table)
	newTable := DBTable{
		Name:        table,
		Fields:      fields,
		Constraints: constraints,
	}

	_, err := db.transactionWrapper(ctx, newTable.String())
	if err != nil {
		db.logger.Println(err)
		return ErrDBCreate
	}
	return nil
}

func (db *DataBase) DeleteTable(ctx context.Context, table string) error {
	db.logger.Println("deleting table:", table)

	// selecting to check if table is empty
	row := db.Select(ctx, table, nil, nil, "LIMIT 1")
	if len(row) > 0 {
		db.logger.Println("table is not empty")
		return ErrDBSelect
	}
	if len(row) > 0 {
		return ErrTableNotEmpty
	}
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s;", table)
	_, err := db.transactionWrapper(ctx, query)
	if err != nil {
		db.logger.Println(err)
		return ErrDBDrop
	}
	return nil
}

func (db *DataBase) Select(ctx context.Context, table string, fields []string, wm *WhereMap, args string) DBResult {
	db.logger.Println("selecting from table:", table)
	sb := strings.Builder{}
	sb.WriteString("SELECT ")
	if len(fields) == 0 {
		sb.WriteString("*")
	} else {
		sb.WriteString(strings.Join(fields, ", "))
	}
	sb.WriteString(fmt.Sprint(" FROM ", table))
	if wm != nil {
		sb.WriteString(" WHERE ")
		sb.WriteString(wm.String())
	}
	if args != "" {
		sb.WriteString(fmt.Sprint(" ", args))
	}
	sb.WriteString(";")
	query := sb.String()
	result, err := db.transactionWrapper(ctx, query)
	if err != nil {
		db.logger.Println(err)
		return nil
	}
	// db.logger.Printf("got data: %+v\n", result)
	db.logger.Println("selected successfully")
	return result
}

func (db *DataBase) Insert(ctx context.Context, table string, fields []FieldName, values [][]interface{}) error {
	db.logger.Println("inserting into table:", table)
	for _, v := range values {
		if len(v) != len(fields) {
			db.logger.Println("warning: number of values does not match number of fields:", len(v), "/", len(fields))
		}
	}
	// insert is handled differently for performance reasons
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		db.logger.Println(err)
		return ErrTxStart
	}

	err = db.insert(ctx, tx, table, fields, values)
	if err != nil {
		db.logger.Printf("error acting on table, rolling back;\n%s", err)
		if e := tx.Rollback(ctx); e != nil {
			db.logger.Println(e)
			return ErrTxRollback
		}
		return ErrDBInsert
	}
	if err = tx.Commit(ctx); err != nil {
		db.logger.Println(err)
		return ErrTxCommit
	}
	db.logger.Println("transaction committed")
	db.logger.Println("inserted successfully")
	return nil
}

func (db *DataBase) Update(ctx context.Context, table string, updates map[FieldName]DBValue, wm *WhereMap) error {
	db.logger.Println("updating table:", table)
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("UPDATE %s SET ", table))
	buffer := make([]string, len(updates))
	counter := 0
	for k, v := range updates {
		buffer[counter] = fmt.Sprintf("%s = '%s'", strings.ToLower(fmt.Sprint(k)), fmt.Sprint(v))
		counter++
	}
	sb.WriteString(join(buffer, ", "))

	if wm != nil {
		sb.WriteString(" WHERE ")
		sb.WriteString(wm.String())
	}

	sb.WriteString(";")
	query := sb.String()
	_, err := db.transactionWrapper(ctx, query)
	if err != nil {
		db.logger.Println(err)
		return ErrDBUpdate
	}
	db.logger.Println("updated successfully")
	return nil
}

func (db *DataBase) Delete(ctx context.Context, table string, wm *WhereMap) error {
	db.logger.Println("deleting from table:", table)
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprint("DELETE FROM ", table))
	if wm != nil {
		sb.WriteString(fmt.Sprint(" WHERE ", wm.String()))
	}
	query := sb.String() + ";"
	_, err := db.transactionWrapper(ctx, query)
	if err != nil {
		db.logger.Println(err)
		return ErrDBDelete
	}
	db.logger.Println("deleted successfully")
	return nil
}

// transactionWrapper
func (db *DataBase) transactionWrapper(ctx context.Context, query string) (DBResult, error) {
	db.logger.Printf("wrapping transaction")
	accessMode := pgx.ReadWrite
	if strings.HasPrefix(query, "SELECT") {
		accessMode = pgx.ReadOnly
	}
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:       pgx.Serializable,
		AccessMode:     accessMode,
		DeferrableMode: pgx.NotDeferrable,
	})
	if err != nil {
		db.logger.Println(err)
		return nil, ErrTxStart
	}
	var rows DBResult

	if strings.HasPrefix(query, "SELECT") {
		db.logger.Println("getting data")
		rows, err = db.getData(ctx, tx, query)
	} else {
		_, err = tx.Exec(ctx, query)
		if env, ok := os.LookupEnv("ENVIRONMENT"); ok && strings.ToLower(env) == "development" {
			db.logger.Printf("executed query: %s", query)
		}
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

func (db *DataBase) GetUserDB() *UserDB {
	return db.users
}
