package database

import (
	"database/sql"

	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix/id"
)

type PuppetQuery struct {
	db  *Database
	log log.Logger
}

func (pq *PuppetQuery) CreateTable() error {
	_, err := pq.db.Exec(`CREATE TABLE IF NOT EXISTS puppet (
		jid          VARCHAR(255) PRIMARY KEY,
		displayname  VARCHAR(255),
		name_quality SMALLINT,
		custom_mxid  VARCHAR(255),
		access_token VARCHAR(255),
		next_batch   VARCHAR(255)
	)`)

	return err
}

func (pq *PuppetQuery) New() *Puppet {
	return &Puppet{
		db:  pq.db,
		log: pq.log,
	}
}

func (pq *PuppetQuery) GetAll() (puppets []*Puppet) {
	rows, err := pq.db.Query("SELECT jid, displayname, name_quality, custom_mxid, access_token, next_batch FROM puppet")
	if err != nil || rows == nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		puppets = append(puppets, pq.New().Scan(rows))
	}
	return
}

func (pq *PuppetQuery) Get(jid string) *Puppet {
	row := pq.db.QueryRow("SELECT jid, displayname, name_quality, custom_mxid, access_token, next_batch  FROM puppet WHERE jid=$1", jid)
	if row == nil {
		return nil
	}
	return pq.New().Scan(row)
}

func (pq *PuppetQuery) GetByCustomMXID(mxid id.UserID) *Puppet {
	row := pq.db.QueryRow("SELECT jid, displayname, name_quality, custom_mxid, access_token, next_batch  FROM puppet WHERE custom_mxid=$1", mxid)
	if row == nil {
		return nil
	}
	return pq.New().Scan(row)
}

func (pq *PuppetQuery) GetAllWithCustomMXID() (puppets []*Puppet) {
	rows, err := pq.db.Query("SELECT * FROM puppet WHERE custom_mxid<>''")
	if err != nil || rows == nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		puppets = append(puppets, pq.New().Scan(rows))
	}
	return
}

type Puppet struct {
	db  *Database
	log log.Logger

	JID         string
	Displayname string
	NameQuality int8

	CustomMXID  id.UserID
	AccessToken string
	NextBatch   string
}

func (puppet *Puppet) Scan(row Scannable) *Puppet {
	var displayname, customMXID, accessToken, nextBatch sql.NullString
	var quality sql.NullInt64
	err := row.Scan(&puppet.JID, &displayname, &quality, &customMXID, &accessToken, &nextBatch)
	if err != nil {
		if err != sql.ErrNoRows {
			puppet.log.Errorln("Database scan failed:", err)
		}
		return nil
	}
	puppet.Displayname = displayname.String
	puppet.NameQuality = int8(quality.Int64)
	puppet.CustomMXID = id.UserID(customMXID.String)
	puppet.AccessToken = accessToken.String
	puppet.NextBatch = nextBatch.String
	return puppet
}

func (puppet *Puppet) Insert() {
	_, err := puppet.db.Exec("INSERT INTO puppet VALUES ($1, $2, $3, $4, $5, $6)",
		puppet.JID, puppet.Displayname, puppet.NameQuality, puppet.CustomMXID, puppet.AccessToken, puppet.NextBatch)
	if err != nil {
		puppet.log.Warnfln("Failed to insert %s: %v", puppet.JID, err)
	}
}

func (puppet *Puppet) Update() {
	_, err := puppet.db.Exec("UPDATE puppet SET displayname=$1, name_quality=$2, custom_mxid=$3, access_token=$4, next_batch=$5 WHERE jid=$6",
		puppet.Displayname, puppet.NameQuality, puppet.CustomMXID, puppet.AccessToken, puppet.NextBatch, puppet.JID)
	if err != nil {
		puppet.log.Warnfln("Failed to update %s->%s: %v", puppet.JID, err)
	}
}
