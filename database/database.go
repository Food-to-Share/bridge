package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	log "maunium.net/go/maulogger/v2"
)

type Database struct {
	*sql.DB
	log log.Logger

	User   *UserQuery
	Portal *PortalQuery
	Puppet *PuppetQuery
}

func New(file string) (*Database, error) {
	conn, err := sql.Open("sqlite3", file)
	if err != nil {
		return nil, err
	}

	db := &Database{
		DB:  conn,
		log: log.Sub("Database"),
	}
	db.User = &UserQuery{
		db:  db,
		log: db.log.Sub("User"),
	}
	db.Portal = &PortalQuery{
		db:  db,
		log: db.log.Sub("Portal"),
	}
	db.Puppet = &PuppetQuery{
		db:  db,
		log: db.log.Sub("Puppet"),
	}
	return db, nil
}

func (db *Database) CreateTables() error {
	err := db.User.CreateTable()
	if err != nil {
		return err
	}
	err = db.Portal.CreateTable()
	if err != nil {
		return err
	}
	err = db.Puppet.CreateTable()
	if err != nil {
		return err
	}
	return nil
}

type Scannable interface {
	Scan(...interface{}) error
}
