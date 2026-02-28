package database

import (
	"github.com/slashbase/layerengine/modules/database/pg"
	lua "github.com/yuin/gopher-lua"
)

type Database struct{}

func (db Database) Init(l *lua.LState) *lua.LTable {
	table := l.NewTable()
	table.RawSetString("query", l.NewFunction(db.Query))
	table.RawSetString("exec", l.NewFunction(db.Exec))
	return table
}

func (Database) Name() string {
	return "database"
}

func (Database) Query(l *lua.LState) int {
	nArgs := l.GetTop()
	var query, dbName string
	var args []any
	for i := 1; i <= nArgs; i++ {
		if i == 1 {
			dbName = l.ToString(i)
			continue
		}
		if i == 2 {
			query = l.ToString(i)
			continue
		}
		value := l.Get(i)
		if value.Type() == lua.LTNumber {
			args = append(args, lua.LVAsNumber(value))
		} else if value.Type() == lua.LTString {
			args = append(args, lua.LVAsString(value))
		} else if value.Type() == lua.LTNil {
			args = append(args, nil)
		} else if value.Type() == lua.LTBool {
			args = append(args, lua.LVAsBool(value))
		}
	}
	data, err := pg.Get().GetDB(dbName).Query(l, query, args...)
	if err != nil {
		return 0
	}
	l.Push(data)
	return 1
}

func (Database) Exec(l *lua.LState) int {
	nArgs := l.GetTop()
	var query, dbName string
	var args []any
	for i := 1; i <= nArgs; i++ {
		if i == 1 {
			dbName = l.ToString(i)
			continue
		}
		if i == 2 {
			query = l.ToString(i)
			continue
		}
		value := l.Get(i)
		if value.Type() == lua.LTNumber {
			args = append(args, lua.LVAsNumber(value))
		} else if value.Type() == lua.LTString {
			args = append(args, lua.LVAsString(value))
		} else if value.Type() == lua.LTNil {
			args = append(args, nil)
		} else if value.Type() == lua.LTBool {
			args = append(args, lua.LVAsBool(value))
		}
	}
	rowsAffected, resultStr, err := pg.Get().GetDB(dbName).Exec(l, query, args...)
	if err != nil {
		return 0
	}
	l.Push(*rowsAffected)
	l.Push(*resultStr)
	return 2
}
