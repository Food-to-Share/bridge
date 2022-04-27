package database

import (
	"database/sql"
	"strings"

	log "maunium.net/go/maulogger/v2"

	"github.com/Food-to-Share/bridge/types"
)

type PortalKey struct {
	JID      types.AppID
	Receiver types.AppID
}

func GroupPortalKey(jid types.AppID) PortalKey {
	return PortalKey{
		JID:      jid,
		Receiver: jid,
	}
}

func NewPortalKey(jid, receiver types.AppID) PortalKey {
	if strings.HasSuffix(jid, "@g.us") {
		receiver = jid
	}
	return PortalKey{
		JID:      jid,
		Receiver: receiver,
	}
}

func (key PortalKey) String() string {
	if key.Receiver == key.JID {
		return key.JID
	}
	return key.JID
}

type PortalQuery struct {
	db  *Database
	log log.Logger
}

func (pq *PortalQuery) CreateTable() error {
	_, err := pq.db.Exec(`CREATE TABLE IF NOT EXISTS portal (
		jid   VARCHAR(255),
		receiver VARCHAR(255),
		mxid  VARCHAR(255) NOT NULL UNIQUE,
		name VARCHAR(255) NOT NULL
		PRIMARY KEY (jid, receiver),
		FOREIGN KEY (receiver) REFERENCES user(mxid)
	)`)
	return err
}

func (pq *PortalQuery) New() *Portal {
	return &Portal{
		db:  pq.db,
		log: pq.log,
	}
}

func (pq *PortalQuery) GetAll() (portals []*Portal) {
	rows, err := pq.db.Query("SELECT * FROM portal")
	if err != nil || rows == nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		portals = append(portals, pq.New().Scan(rows))
	}
	return
}

func (pq *PortalQuery) GetByJID(key PortalKey) *Portal {
	return pq.get("SELECT * FROM portal WHERE jid=? AND receiver=?", key.JID, key.Receiver)
}

func (pq *PortalQuery) GetByMXID(mxid types.MatrixRoomID) *Portal {
	return pq.get("SELECT * FROM portal WHERE mxid=?", mxid)
}

func (pq *PortalQuery) get(query string, args ...interface{}) *Portal {
	row := pq.db.QueryRow(query, args...)
	if row == nil {
		return nil
	}
	return pq.New().Scan(row)
}

type Portal struct {
	db  *Database
	log log.Logger

	Key  PortalKey
	MXID types.MatrixRoomID

	Name string
}

func (portal *Portal) Scan(row Scannable) *Portal {
	err := row.Scan(&portal.Key.JID, &portal.MXID, &portal.Name)
	var mxid sql.NullString
	if err != nil {
		if err != sql.ErrNoRows {
			portal.log.Errorln("Database scan failed:", err)
		}
		return nil
	}
	portal.MXID = mxid.String
	return portal
}

func (portal *Portal) mxidPtr() *string {
	if len(portal.MXID) > 0 {
		return &portal.MXID
	}
	return nil
}

func (portal *Portal) Insert() {
	_, err := portal.db.Exec("INSERT INTO portal VALUES (?, ?, ?, ?)",
		portal.Key.JID, portal.Key.Receiver, portal.mxidPtr(), portal.Name)
	if err != nil {
		portal.log.Warnfln("Failed to insert %s: %v", portal.Key, err)
	}
}

func (portal *Portal) Update() {
	var mxid *string
	if len(portal.MXID) > 0 {
		mxid = &portal.MXID
	}
	_, err := portal.db.Exec("UPDATE portal SET mxid=?, name=? WHERE jid=? AND receiver=?",
		mxid, portal.Name, portal.Key.JID, portal.Key.Receiver)
	if err != nil {
		portal.log.Warnfln("Failed to update %s: %v", portal.Key, err)
	}
}
