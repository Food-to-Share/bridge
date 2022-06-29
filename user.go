package main

import (
	"errors"
	"strings"

	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/Food-to-Share/bridge/database"
)

type User struct {
	*database.User

	Bridge *Bridge
	log    log.Logger
}

func (bridge *Bridge) GetUserByMXID(userID id.UserID) *User {
	bridge.usersLock.Lock()
	defer bridge.usersLock.Unlock()
	user, ok := bridge.usersByMXID[userID]
	if !ok {
		dbUser := bridge.DB.User.GetByMXID(userID)
		if dbUser == nil {
			dbUser = bridge.DB.User.New()
			dbUser.MXID = userID
			dbUser.Insert()
		}
		user = bridge.NewUser(dbUser)
		bridge.usersByMXID[user.MXID] = user
		if len(user.JID) > 0 {
			bridge.usersByJID[user.JID] = user
		}
		if len(user.ManagementRoom) > 0 {
			bridge.managementRooms[user.ManagementRoom] = user
		}
	}
	return user
}

func (bridge *Bridge) GetUserByJID(userID string) *User {
	bridge.usersLock.Lock()
	defer bridge.usersLock.Unlock()
	user, ok := bridge.usersByJID[userID]
	if !ok {
		dbUser := bridge.DB.User.GetByJID(userID)
		if dbUser == nil {
			return nil
		}
		user = bridge.NewUser(dbUser)
		bridge.usersByMXID[user.MXID] = user
		bridge.usersByJID[user.JID] = user
		if len(user.ManagementRoom) > 0 {
			bridge.managementRooms[user.ManagementRoom] = user
		}
	}
	return user
}

func (bridge *Bridge) GetAllUsers() []*User {
	bridge.usersLock.Lock()
	defer bridge.usersLock.Unlock()
	dbUsers := bridge.DB.User.GetAll()
	output := make([]*User, len(dbUsers))
	for index, dbUser := range dbUsers {
		user, ok := bridge.usersByMXID[dbUser.MXID]
		if !ok {
			user = bridge.NewUser(dbUser)
			bridge.usersByMXID[user.MXID] = user
			if len(user.JID) > 0 {
				bridge.usersByJID[user.JID] = user
			}
			if len(user.ManagementRoom) > 0 {
				bridge.managementRooms[user.ManagementRoom] = user
			}
		}
		output[index] = user
	}
	return output
}

func (bridge *Bridge) NewUser(dbUser *database.User) *User {
	user := &User{
		User:   dbUser,
		Bridge: bridge,
		log:    bridge.Log.Sub("User").Sub(string(dbUser.MXID)),
	}
	return user
}

func (user *User) ensureInvited(intent *appservice.IntentAPI, roomID id.RoomID) (ok bool) {
	inviteContent := event.Content{
		Parsed: &event.MemberEventContent{
			Membership: event.MembershipInvite,
			IsDirect:   false,
		},
		Raw: map[string]interface{}{},
	}
	customPuppet := user.Bridge.getPuppetByCustomMXID(user.MXID)

	if customPuppet != nil && customPuppet.CustomIntent() != nil {
		inviteContent.Raw["fi.mau.will_auto_accept"] = true
	}
	_, err := intent.SendStateEvent(roomID, event.StateMember, user.MXID.String(), &inviteContent)

	var httpErr mautrix.HTTPError
	if err != nil && errors.As(err, &httpErr) && httpErr.RespError != nil && strings.Contains(httpErr.RespError.Err, "is already in the room") {
		user.Bridge.StateStore.SetMembership(roomID, user.MXID, event.MembershipJoin)
		ok = true
		return
	} else if err != nil {
		user.log.Warnfln("Failed to invite user %s: %v", roomID, err)
	} else {
		ok = true
	}

	if customPuppet != nil && customPuppet.CustomIntent() != nil {
		err = customPuppet.CustomIntent().EnsureJoined(roomID, appservice.EnsureJoinedParams{IgnoreCache: true})
		if err != nil {
			user.log.Warnfln("Failed to auto-join %s: %v", roomID, err)
			ok = false
		} else {
			ok = true
		}
	}
	return
}

func (user *User) StartUser(jid string, displayName string, reason string) (*Portal, *Puppet, bool, error) {
	user.log.Debugfln("Starting User with", jid, "from", reason)
	puppet := user.Bridge.GetPuppetByJID(jid)
	puppet.Sync(user, displayName)
	portal := user.Bridge.GetPortalByJID(database.NewPortalKey(puppet.JID, user.JID))

	if len(portal.MXID) > 0 {
		ok := portal.ensureUserInvited(user)
		if !ok {
			portal.log.Warnfln("ensureUserInvited(%s) return false, creating new portal", user.MXID)
			portal.MXID = ""
		} else {
			return portal, puppet, false, nil
		}
	}
	err := portal.CreateMatrixRoom(user)
	return portal, puppet, true, err
}
