package main

import (
	"fmt"
	"regexp"

	"github.com/Food-to-Share/bridge/database"
	"github.com/Food-to-Share/bridge/types"
	log "maunium.net/go/maulogger/v2"
)

func (bridge *Bridge) ParsePuppetMXID(mxid types.MatrixUserID) (types.AppID, bool) {
	userIDRegex, err := regexp.Compile(fmt.Sprintf("^@%s:%s$",
		bridge.Config.Bridge.FormatUsername("([0-9]+)"),
		bridge.Config.Homeserver.Domain))
	if err != nil {
		bridge.Log.Warnfln("Failed to compile puppet user ID regex:", err)
		return "", false
	}
	match := userIDRegex.FindStringSubmatch(string(mxid))
	if match == nil || len(match) != 2 {
		return "", false
	}

	jid := types.AppID(match[1])
	return jid, true
}

func (bridge *Bridge) GetPuppetByMXID(mxid types.MatrixUserID) *Puppet {
	jid, ok := bridge.ParsePuppetMXID(mxid)
	if !ok {
		return nil
	}

	return bridge.GetPuppetByJID(jid)
}

func (bridge *Bridge) GetPuppetByJID(jid types.AppID) *Puppet {
	bridge.puppetsLock.Lock()
	defer bridge.puppetsLock.Unlock()
	puppet, ok := bridge.puppets[jid]
	if !ok {
		dbPuppet := bridge.DB.Puppet.Get(jid)
		if dbPuppet == nil {
			dbPuppet = bridge.DB.Puppet.New()
			dbPuppet.JID = jid
			dbPuppet.Insert()
		}
		puppet = bridge.NewPuppet(dbPuppet)
		bridge.puppets[puppet.JID] = puppet
	}
	return puppet
}

func (bridge *Bridge) GetAllPuppets() []*Puppet {
	bridge.puppetsLock.Lock()
	defer bridge.puppetsLock.Unlock()
	dbPuppets := bridge.DB.Puppet.GetAll()
	output := make([]*Puppet, len(dbPuppets))
	for index, dbPuppet := range dbPuppets {
		puppet, ok := bridge.puppets[dbPuppet.JID]
		if !ok {
			puppet = bridge.NewPuppet(dbPuppet)
			bridge.puppets[dbPuppet.JID] = puppet
		}
		output[index] = puppet
	}
	return output
}

func (bridge *Bridge) NewPuppet(dbPuppet *database.Puppet) *Puppet {
	return &Puppet{
		Puppet: dbPuppet,
		bridge: bridge,
		log:    bridge.Log.Sub(fmt.Sprintf("Puppet/%s", dbPuppet.JID)),

		MXID: fmt.Sprintf("@%s:%s", dbPuppet.JID,
			bridge.Config.Homeserver.Domain),
	}
}

type Puppet struct {
	*database.Puppet

	bridge *Bridge
	log    log.Logger

	MXID types.MatrixUserID
}

// func (puppet *Puppet) Intent() *appservice.IntentAPI {
// 	return puppet.bridge.AS.Intent(puppet.MXID)
// }
