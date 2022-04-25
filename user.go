package main

import (
	log "maunium.net/go/maulogger/v2"

	"github.com/Food-to-Share/bridge/database"
	"github.com/Food-to-Share/bridge/types"
)

type User struct {
	*database.User

	Bridge *Bridge
	log    log.Logger
}

func (bridge *Bridge) GetUserByMXID(userID types.MatrixUserID) *User {
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
