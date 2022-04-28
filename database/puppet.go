package database

import (
	"database/sql"

	log "maunium.net/go/maulogger/v2"
)

type PuppetQuery struct {
	db  *Database
	log log.Logger
}

func (pq *PuppetQuery) CreateTable() error {
	_, err := pq.db.Exec(`CREATE TABLE IF NOT EXISTS puppet (
		jid          VARCHAR(25) PRIMARY KEY,
		displayname  VARCHAR(255),
		name_quality SMALLINT
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
	rows, err := pq.db.Query("SELECT * FROM puppet")
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
	row := pq.db.QueryRow("SELECT * FROM puppet WHERE jid=$1", jid)
	if row == nil {
		return nil
	}
	return pq.New().Scan(row)
}

type Puppet struct {
	db  *Database
	log log.Logger

	JID         string
	Displayname string
	NameQuality int8
}

func (puppet *Puppet) Scan(row Scannable) *Puppet {
	var displayname sql.NullString
	var quality sql.NullInt64
	err := row.Scan(&puppet.JID, &displayname, &quality)
	if err != nil {
		if err != sql.ErrNoRows {
			puppet.log.Errorln("Database scan failed:", err)
		}
		return nil
	}
	puppet.Displayname = displayname.String
	puppet.NameQuality = int8(quality.Int64)
	return puppet
}

func (puppet *Puppet) Insert() {
	_, err := puppet.db.Exec("INSERT INTO puppet VALUES ($1, $2, $3)",
		puppet.JID, puppet.Displayname, puppet.NameQuality)
	if err != nil {
		puppet.log.Warnfln("Failed to insert %s: %v", puppet.JID, err)
	}
}

func (puppet *Puppet) Update() {
	_, err := puppet.db.Exec("UPDATE puppet SET displayname=$1, name_quality=$2 WHERE jid=$3",
		puppet.Displayname, puppet.NameQuality, puppet.JID)
	if err != nil {
		puppet.log.Warnfln("Failed to update %s->%s: %v", puppet.JID, err)
	}
}
