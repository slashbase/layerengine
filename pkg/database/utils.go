package database

import (
	"database/sql"
	"log"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	lua "github.com/yuin/gopher-lua"
)

func pgxRowsToLuaTable(rows pgx.Rows, l *lua.LState) *lua.LTable {
	arr := l.NewTable()
	// Iterate over rows
	for rows.Next() {
		// Get column names and values from the row
		columns := rows.FieldDescriptions()
		values, err := rows.Values()
		if err != nil {
			log.Fatal("Unable to get row values:", err)
		}
		// Convert row to Lua table
		luaTable := convertToLuaTable(l, columns, values)
		arr.Append(luaTable)
	}
	return arr
}

// Helper function to convert PGX row to Lua table
func convertToLuaTable(l *lua.LState, columns []pgconn.FieldDescription, values []interface{}) *lua.LTable {
	luaTable := l.NewTable()

	for i := 0; i < len(columns); i++ {
		columnName := columns[i].Name
		columnValue := values[i]

		switch v := columnValue.(type) {
		case int64:
		case int32:
			luaTable.RawSetString(columnName, lua.LNumber(v))
		case float32:
		case float64:
			luaTable.RawSetString(columnName, lua.LNumber(v))
		case string:
			luaTable.RawSetString(columnName, lua.LString(v))
		case bool:
			luaTable.RawSetString(columnName, lua.LBool(v))
		case nil:
			luaTable.RawSetString(columnName, lua.LNil)
		case sql.NullString:
			if v.Valid {
				luaTable.RawSetString(columnName, lua.LString(v.String))
			} else {
				luaTable.RawSetString(columnName, lua.LNil)
			}
		case sql.NullBool:
			if v.Valid {
				luaTable.RawSetString(columnName, lua.LBool(v.Bool))
			} else {
				luaTable.RawSetString(columnName, lua.LNil)
			}
		case sql.NullInt32:
			if v.Valid {
				luaTable.RawSetString(columnName, lua.LNumber(v.Int32))
			} else {
				luaTable.RawSetString(columnName, lua.LNil)
			}
		case sql.NullInt64:
			if v.Valid {
				luaTable.RawSetString(columnName, lua.LNumber(v.Int64))
			} else {
				luaTable.RawSetString(columnName, lua.LNil)
			}
		case sql.NullTime:
			if v.Valid {
				luaTable.RawSetString(columnName, lua.LString(v.Time.Format(time.RFC3339)))
			} else {
				luaTable.RawSetString(columnName, lua.LNil)
			}
		case pgtype.TID, pgtype.TextArray, pgtype.VarcharArray, pgtype.BoolArray,
			pgtype.UUIDArray, pgtype.DateArray, pgtype.Int2Array, pgtype.Int4Array,
			pgtype.Int8Array, pgtype.Float4Array, pgtype.Float8Array, pgtype.Interval:
			// Handle these cases as needed
			luaTable.RawSetString(columnName, lua.LString("Unsupported type"))
		case []byte:
			luaTable.RawSetString(columnName, lua.LString(string(v)))
		default:
			log.Printf("Unknown type for column '%s' value: %v\n", columnName, v)
		}
	}

	return luaTable
}
