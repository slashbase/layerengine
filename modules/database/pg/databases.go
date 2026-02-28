package pg

type databases struct {
	dbs map[string]*Database
}

var dbs *databases

func Init(connStrs map[string]string) {
	dbs = &databases{
		dbs: map[string]*Database{},
	}
	for name, connStr := range connStrs {
		db := &Database{}
		db.Init(connStr)
		dbs.dbs[name] = db
	}
}

func Get() *databases {
	return dbs
}

func (dbs *databases) GetDB(name string) *Database {
	return dbs.dbs[name]
}

func (dbs *databases) Close() {
	for _, db := range dbs.dbs {
		db.Close()
	}
}
