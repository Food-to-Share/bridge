package database

import (
	"database/sql"
	"strings"

	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix/id"
)

type PortalKey struct {
	JID      string
	Receiver string
}

func GroupPortalKey(jid string) PortalKey {
	return PortalKey{
		JID:      jid,
		Receiver: jid,
	}
}

func NewPortalKey(jid, receiver string) PortalKey {
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
		mxid  VARCHAR(255) UNIQUE,
		name VARCHAR(255),
		PRIMARY KEY (jid, receiver)
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
	return pq.get("SELECT * FROM portal WHERE jid=$1 AND receiver=$2", key.JID, key.Receiver)
}

func (pq *PortalQuery) GetByMXID(mxid id.RoomID) *Portal {
	return pq.get("SELECT * FROM portal WHERE mxid=$1", mxid)
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
	MXID id.RoomID

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
	portal.MXID = id.RoomID(mxid.String)
	return portal
}

func (portal *Portal) mxidPtr() *id.RoomID {
	if len(portal.MXID) > 0 {
		return &portal.MXID
	}
	return nil
}

func (portal *Portal) Insert() {
	_, err := portal.db.Exec("INSERT INTO portal VALUES ($1, $2, $3, $4)",
		portal.Key.JID, portal.Key.Receiver, portal.mxidPtr(), portal.Name)
	if err != nil {
		portal.log.Warnfln("Failed to insert %s: %v", portal.Key, err)
	}
}

func (portal *Portal) Update() {
	_, err := portal.db.Exec("UPDATE portal SET mxid=$1, name=$2 WHERE jid=$3 AND receiver=$4",
		portal.mxidPtr(), portal.Name, portal.Key.JID, portal.Key.Receiver)
	if err != nil {
		portal.log.Warnfln("Failed to update %s: %v", portal.Key, err)
	}
}
