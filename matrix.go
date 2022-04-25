package main

import (
	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix"
	appservice "maunium.net/go/mautrix-appservice"

	"github.com/Food-to-Share/bridge/types"
)

type MatrixListener struct {
	bridge *Bridge
	log    log.Logger
	as     *appservice.AppService
}

func NewMatrixListener(bridge *Bridge) *MatrixListener {
	handler := &MatrixListener{
		bridge: bridge,
		as:     bridge.AS,
		log:    bridge.Log.Sub("Matrix"),
	}
	bridge.EventProcessor.On(mautrix.StateMember, handler.HandleMembership)
	bridge.EventProcessor.On(mautrix.StateRoomName, handler.HandleRoomMetadata)

	return handler
}

func (mx *MatrixHandler) HandleMembership(evt *mautrix.Event) {
	if evt.Content.Membership == "invite" && evt.GetStateKey() == mx.as.BotMXID() {
		mx.HandleBotInvite(evt)
	}
}

func (mx *MatrixHandler) HandleRoomMetadata(evt *mautrix.Event) {
	user := mx.bridge.GetUserByMXID(types.MatrixUserID(evt.Sender))
	if user == nil || !user.Whitelisted || !user.IsLoggedIn() {
		return
	}

	portal := mx.bridge.GetPortalByMXID(evt.RoomID)
	if portal == nil || portal.IsPrivateChat() {
		return
	}

	var resp <-chan string
	var err error
	switch evt.Type {
	case mautrix.StateRoomName:
		resp, err = user.Conn.UpdateGroupSubject(evt.Content.Name, portal.Key.JID)
	case mautrix.StateTopic:
		return
	}
	if err != nil {
		mx.log.Errorln(err)
	} else {
		out := <-resp
		mx.log.Infoln(out)
	}
}
