package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix/id"
)

type ProvisioningAPI struct {
	bridge *Bridge
	log    log.Logger
}

func (prov *ProvisioningAPI) Init() {
	prov.log = prov.bridge.Log.Sub("Provisiong")

	prov.log.Debugfln("Enabling provisioning API at", prov.bridge.Config.AppService.Provisioning.SegmentKey, prov.log)

	r := prov.bridge.AS.Router.PathPrefix(prov.bridge.Config.AppService.Provisioning.Prefix).Subrouter()
	r.HandleFunc("/v1/startUser", prov.StartUser).Methods(http.MethodPost)
}

type Error struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	ErrCode string `json"errcode"`
}

type Response struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
}

type NewUser struct {
	jid         string `json:"jid"`
	displayName string `json:"name"`
}

type OtherUserInfo struct {
	MXID id.UserID `json:"mxid"`
	JID  string    `json:"jid"`
	Name string    `json:"displayname"`
}

type PortalInfo struct {
	RoomID      id.RoomID      `json:"room_id"`
	OtherUser   *OtherUserInfo `json:"other_user,omitempty"`
	JustCreated bool           `json:"just_created"`
}

func (prov *ProvisioningAPI) StartUser(w http.ResponseWriter, r *http.Request) {
	user := prov.bridge.GetUserByMXID(id.UserID("@kevindom:localhost"))

	jid, displayName := prov.resolveIdentifier(w, r)
	if user == nil {
		return
	}

	portal, puppet, justCreated := user.StartUser(jid, displayName, "provisioning API New User")

	status := http.StatusOK
	// if justCreated{
	// 	status
	// }
	jsonResponse(w, status, PortalInfo{
		RoomID: portal.MXID,
		OtherUser: &OtherUserInfo{
			JID:  puppet.JID,
			MXID: puppet.MXID,
			Name: puppet.Displayname,
		},
		JustCreated: justCreated,
	})
}

func (prov *ProvisioningAPI) resolveIdentifier(w http.ResponseWriter, r *http.Request) (string, string) {
	decoder := json.NewDecoder(r.Body)
	var u NewUser
	err := decoder.Decode(&u)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, Error{
			Error:   fmt.Sprintf("User not found"),
			ErrCode: "Not found!",
		})
		return "", ""
	}
	return u.jid, u.displayName
}

func jsonResponse(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
