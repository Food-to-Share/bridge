package database

import (
	log "maunium.net/go/maulogger"
	"github.com/Rhymen/go-whatsapp"
)

type UserQuery struct {
	db  *Database
	log *log.Sublogger
}

func (uq *UserQuery) CreateTable() error {
	_, err := uq.db.Exec(`CREATE TABLE IF NOT EXISTS user (
		mxid  VARCHAR(255) PRIMARY KEY,
		client_id    VARCHAR(255),
		client_token VARCHAR(255),
		server_token VARCHAR(255),
		organization VARCHAR(255)
	)`)
	return err
}

func (uq *UserQuery) New() *User {
	return &User{
		db:  uq.db,
		log: uq.log,
	}
}

func (uq *UserQuery) GetAll() (users []*User) {
	rows, err := uq.db.Query("SELECT * FROM user")
	if err != nil || rows == nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		users = append(users, uq.New().Scan(rows))
	}
	return
}

func (uq *UserQuery) Get(userID string) *User {
	row := uq.db.QueryRow("SELECT * FROM user WHERE mxid=?", userID)
	if row == nil {
		return nil
	}
	return uq.New().Scan(row)
}

type User struct {
	db  *Database
	log *log.Sublogger

	UserID string

	session whatsapp.Session
}

func (user *User) Scan(row Scannable) *User {
	err := row.Scan(&user.UserID, &user.session.ClientId, &user.session.ClientToken, &user.session.ServerToken,
		&user.session.EncKey, &user.session.MacKey, &user.session.Wid)
	if err != nil {
		user.log.Fatalln("Database scan failed:", err)
	}
	return user
}
/*
func (user *User) Insert() error {
	_, err := user.db.Exec("INSERT INTO user VALUES (?, ?, ?, ?, ?, ?, ?)", user.UserID, user.session.ClientId,
		user.session.ClientToken, user.session.ServerToken, user.session.EncKey, user.session.MacKey, user.session.Wid)
	return err
}

func (user *User) Update() error {
	_, err := user.db.Exec("UPDATE user SET client_id=?, client_token=?, server_token=?, enc_key=?, mac_key=?, wid=? WHERE mxid=?",
		user.session.ClientId, user.session.ClientToken, user.session.ServerToken, user.session.EncKey, user.session.MacKey,
		user.session.Wid, user.UserID)
	return err
}*/