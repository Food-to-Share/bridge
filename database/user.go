package database

import (
	"database/sql"
	"strings"

	log "maunium.net/go/maulogger/v2"

	"github.com/Food-to-Share/bridge/types"
)

type UserQuery struct {
	db  *Database
	log log.Logger
}

func (uq *UserQuery) CreateTable() error {
	_, err := uq.db.Exec(`CREATE TABLE IF NOT EXISTS user (
		mxid  VARCHAR(255) PRIMARY KEY,
		jid  VARCHAR(25)  UNIQUE,
		management_room VARCHAR(255),
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

func (uq *UserQuery) GetByMXID(userID types.MatrixUserID) *User {
	row := uq.db.QueryRow("SELECT * FROM user WHERE mxid=?", userID)
	if row == nil {
		return nil
	}
	return uq.New().Scan(row)
}

func (uq *UserQuery) GetByJID(userID types.AppID) *User {
	row := uq.db.QueryRow("SELECT * FROM user WHERE jid=?", stripSuffix(userID))
	if row == nil {
		return nil
	}
	return uq.New().Scan(row)
}

type User struct {
	db  *Database
	log log.Logger

	MXID           types.MatrixUserID
	JID            types.AppID
	ManagementRoom types.MatrixRoomID
}

func (user *User) Scan(row Scannable) *User {
	var jid, clientID, clientToken, serverToken sql.NullString
	var encKey, macKey []byte
	err := row.Scan(&user.MXID, &jid, &user.ManagementRoom, &clientID, &clientToken, &serverToken, &encKey, &macKey)
	if err != nil {
		if err != sql.ErrNoRows {
			user.log.Errorln("Database scan failed:", err)
		}
		return nil
	}
	// if len(jid.String) > 0 && len(clientID.String) > 0 {
	// 	user.JID = jid.String + whatsappExt.NewUserSuffix
	// 	user.Session = &whatsapp.Session{
	// 		ClientId:    clientID.String,
	// 		ClientToken: clientToken.String,
	// 		ServerToken: serverToken.String,
	// 		EncKey:      encKey,
	// 		MacKey:      macKey,
	// 		Wid:         jid.String + whatsappExt.OldUserSuffix,
	// 	}
	// } else {
	// 	user.Session = nil
	// }
	return user
}

func stripSuffix(jid types.AppID) string {
	if len(jid) == 0 {
		return jid
	}

	index := strings.IndexRune(jid, '@')
	if index < 0 {
		return jid
	}

	return jid[:index]
}

func (user *User) jidPtr() *string {
	if len(user.JID) > 0 {
		str := stripSuffix(user.JID)
		return &str
	}
	return nil
}

// func (user *User) sessionUnptr() (sess whatsapp.Session) {
// 	if user.Session != nil {
// 		sess = *user.Session
// 	}
// 	return
// }

func (user *User) Insert() {
	// sess := user.sessionUnptr()
	_, err := user.db.Exec("INSERT INTO user VALUES (?, ?, ?)", user.MXID, user.jidPtr(),
		user.ManagementRoom)
	if err != nil {
		user.log.Warnfln("Failed to insert %s: %v", user.MXID, err)
	}
}

// func (user *User) Update() {
// 	sess := user.sessionUnptr()
// 	_, err := user.db.Exec("UPDATE user SET jid=?, management_room=?, client_id=?, client_token=?, server_token=? WHERE mxid=?",
// 		user.jidPtr(), user.ManagementRoom,
// 		sess.ClientId, sess.ClientToken, sess.ServerToken, user.MXID)
// 	if err != nil {
// 		user.log.Warnfln("Failed to update %s: %v", user.MXID, err)
// 	}
// }
