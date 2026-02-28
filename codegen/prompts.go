package codegen

const (
	modulesInfo = `Use:
versionString = layer.system.version() - get system version
resultArray = layer.database.query(dbname, querystring, args...) - query the database, use $1, $2 for args
rowsAffected, resultString = layer.database.exec(dbname, execstring, args...) - exec query on database, use $1, $2 for args`
)

const (
	promptGuide = "Write a Lua function that takes input parameters, processes the input according to given description and returns output.\nWrite code in json field \"code\"."
)
