package main

import (
	log "maunium.net/go/maulogger/v2"
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
