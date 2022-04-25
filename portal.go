package main

import (
	"fmt"
	"sync"

	"github.com/Food-to-Share/bridge/database"
	"github.com/Food-to-Share/bridge/types"
	log "maunium.net/go/maulogger/v2"
)

func (bridge *Bridge) GetPortalByMXID(mxid types.MatrixRoomID) *Portal {
	bridge.portalsLock.Lock()
	defer bridge.portalsLock.Unlock()
	portal, ok := bridge.portalsByMXID[mxid]
	if !ok {
		dbPortal := bridge.DB.Portal.GetByMXID(mxid)
		if dbPortal == nil {
			return nil
		}
		portal = bridge.NewPortal(dbPortal)
		bridge.portalsByJID[portal.Key] = portal
		if len(portal.MXID) > 0 {
			bridge.portalsByMXID[portal.MXID] = portal
		}
	}
	return portal
}

func (bridge *Bridge) GetPortalByJID(key database.PortalKey) *Portal {
	bridge.portalsLock.Lock()
	defer bridge.portalsLock.Unlock()
	portal, ok := bridge.portalsByJID[key]
	if !ok {
		dbPortal := bridge.DB.Portal.GetByJID(key)
		if dbPortal == nil {
			dbPortal = bridge.DB.Portal.New()
			dbPortal.Key = key
			dbPortal.Insert()
		}
		portal = bridge.NewPortal(dbPortal)
		bridge.portalsByJID[portal.Key] = portal
		if len(portal.MXID) > 0 {
			bridge.portalsByMXID[portal.MXID] = portal
		}
	}
	return portal
}

func (bridge *Bridge) GetAllPortals() []*Portal {
	bridge.portalsLock.Lock()
	defer bridge.portalsLock.Unlock()
	dbPortals := bridge.DB.Portal.GetAll()
	output := make([]*Portal, len(dbPortals))
	for index, dbPortal := range dbPortals {
		portal, ok := bridge.portalsByJID[dbPortal.Key]
		if !ok {
			portal = bridge.NewPortal(dbPortal)
			bridge.portalsByJID[portal.Key] = portal
			if len(dbPortal.MXID) > 0 {
				bridge.portalsByMXID[dbPortal.MXID] = portal
			}
		}
		output[index] = portal
	}
	return output
}

func (bridge *Bridge) NewPortal(dbPortal *database.Portal) *Portal {
	return &Portal{
		Portal: dbPortal,
		bridge: bridge,
		log:    bridge.Log.Sub(fmt.Sprintf("Portal/%s", dbPortal.Key)),
	}
}

const recentlyHandledLength = 100

type Portal struct {
	*database.Portal

	bridge *Bridge
	log    log.Logger

	roomCreateLock sync.Mutex
}
