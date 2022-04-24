package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	log "maunium.net/go/maulogger"
)

type Database struct {
	*sql.DB
	log *log.Sublogger

	User *UserQuery
}

func New(file string) (*Database, error) {
	conn, err := sql.Open("sqlite3", file)
	if err != nil {
		return nil, err
	}

	db := &Database{
		DB:  conn,
		log: log.CreateSublogger("Database", log.LevelDebug),
	}
	db.User = &UserQuery{
		db:  db,
		log: log.CreateSublogger("Database/User", log.LevelDebug),
	}
	return db, nil
}

type Scannable interface {
	Scan(...interface{}) error
}