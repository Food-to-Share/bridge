package main

import (
	"fmt"
	"sync"

	"github.com/Food-to-Share/bridge/database"
	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func (bridge *Bridge) GetPortalByMXID(mxid id.RoomID) *Portal {
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

func (portal *Portal) GetBasePowerLevels() *event.PowerLevelsEventContent {
	anyone := 0
	nope := 99
	invite := 50

	return &event.PowerLevelsEventContent{
		UsersDefault:    anyone,
		EventsDefault:   anyone,
		RedactPtr:       &anyone,
		StateDefaultPtr: &nope,
		BanPtr:          &nope,
		InvitePtr:       &invite,
		Users: map[id.UserID]int{
			portal.MainIntent().UserID: 100,
		},
		Events: map[string]int{
			event.StateRoomName.Type:   anyone,
			event.StateRoomAvatar.Type: anyone,
			event.StateTopic.Type:      anyone,
			event.EventReaction.Type:   anyone,
			event.EventRedaction.Type:  anyone,
		},
	}
}

func (portal *Portal) ensureUserInvited(user *User) bool {
	return user.ensureInvited(portal.MainIntent(), portal.MXID)
}

func (portal *Portal) MainIntent() *appservice.IntentAPI {
	return portal.bridge.GetPuppetByJID(portal.Key.JID).DefaultIntent()
}

func (portal *Portal) CreateMatrixRoom(user *User) error {
	portal.roomCreateLock.Lock()
	defer portal.roomCreateLock.Unlock()
	if len(portal.MXID) > 0 {
		return nil
	}

	intent := portal.MainIntent()
	if err := intent.EnsureRegistered(); err != nil {
		return err
	}

	portal.log.Infofln("Creating Matrix room. Info source:", user.MXID)

	puppet := portal.bridge.GetPuppetByJID(portal.Key.JID)
	portal.Name = puppet.Displayname

	initialState := []*event.Event{{
		Type: event.StatePowerLevels,
		Content: event.Content{
			Parsed: portal.GetBasePowerLevels(),
		},
	}}

	var invite []id.UserID

	invite = append(invite, portal.bridge.Bot.UserID)

	creationContent := make(map[string]interface{})
	creationContent["m.federate"] = true

	resp, err := intent.CreateRoom(&mautrix.ReqCreateRoom{
		Visibility:      "private",
		Name:            portal.Name,
		Topic:           "Help " + portal.Name,
		Invite:          invite,
		Preset:          "private_chat",
		IsDirect:        false,
		InitialState:    initialState,
		CreationContent: creationContent,
	})

	portal.MXID = resp.RoomID
	portal.bridge.portalsLock.Lock()
	portal.bridge.portalsByMXID[portal.MXID] = portal
	portal.bridge.portalsLock.Unlock()
	portal.Update()
	portal.log.Infofln("Matrix room created:", portal.MXID)

	for _, userID := range invite {
		portal.bridge.StateStore.SetMembership(portal.MXID, userID, event.MembershipInvite)
	}

	portal.ensureUserInvited(user)

	if err != nil {
		return err
	}
	return nil
}
