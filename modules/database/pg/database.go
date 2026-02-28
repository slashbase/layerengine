package pg

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	lua "github.com/yuin/gopher-lua"
)

type DBType string

const (
	PostgreSQL DBType = "postgresql"
)

type Database struct {
	Type DBType
	db   interface{}
}

func (db *Database) Init(connectionString string) {
	conn, err := pgx.Connect(context.Background(), connectionString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	db.db = conn
	db.Type = PostgreSQL
}

func (db *Database) Query(l *lua.LState, query string, args ...any) (*lua.LTable, error) {
	if db.Type == PostgreSQL {
		conn := db.db.(*pgx.Conn)
		rows, err := conn.Query(context.Background(), query, args...)
		if err != nil {
			return nil, err
		}
		arr := pgxRowsToLuaTable(rows, l)
		return arr, nil
	}
	return nil, nil
}

func (db *Database) Exec(l *lua.LState, query string, args ...any) (*lua.LNumber, *lua.LString, error) {
	if db.Type == PostgreSQL {
		conn := db.db.(*pgx.Conn)
		cmdTag, err := conn.Exec(context.Background(), query, args...)
		if err != nil {
			return nil, nil, err
		}
		num := lua.LNumber(cmdTag.RowsAffected())
		str := lua.LString(cmdTag.String())
		return &num, &str, nil
	}
	return nil, nil, nil
}

func (db *Database) Close() {
	if db.Type == PostgreSQL {
		conn := db.db.(*pgx.Conn)
		conn.Close(context.Background())
		return
	}
}
