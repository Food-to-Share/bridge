package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "maunium.net/go/maulogger/v2"
	"maunium.net/go/mautrix/id"
)

var IdentityServerURL = "http://localhost:8090"

type ProvisioningAPI struct {
	bridge *Bridge
	log    log.Logger
}

func (prov *ProvisioningAPI) Init() {
	prov.log = prov.bridge.Log.Sub("Provisiong")

	prov.log.Debugfln("Enabling provisioning API at", prov.bridge.Config.AppService.Provisioning.SegmentKey, prov.log)

	r := prov.bridge.AS.Router.PathPrefix(prov.bridge.Config.AppService.Provisioning.Prefix).Subrouter()
	r.HandleFunc("/v1/startUser", prov.StartUser).Methods(http.MethodPost)
	r.HandleFunc("/v1/syncNiss", prov.SyncNiss).Methods(http.MethodPost)
	r.HandleFunc("/v1/lookup", prov.Lookup).Methods(http.MethodPost)
}

type Error struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	ErrCode string `json:"errcode"`
}

type Response struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
}

type NewUser struct {
	Jid         string `json:"jid"`
	DisplayName string `json:"displayName"`
}

type NumberToSync struct {
	Jid    string `json:"jid"`
	Number string `json:"number"`
}

type NissToSync struct {
	Jid  string `json:"jid"`
	Niss string `json:"niss"`
}

type RequestTokenNumber struct {
	Client_Secret string `json:"client_secret"`
	Country       string `json:"country"`
	Phone_number  string `json:"phone_number"`
	Send_attempt  int    `json:"send_attempt"`
}

type RequestTokenNiss struct {
	Client_Secret string `json:"client_secret"`
	Niss          string `json:"niss"`
	Send_attempt  int    `json:"send_attempt"`
}

type BindToken struct {
	Client_Secret string `json:"client_secret"`
	Mxid          string `json:"mxid"`
	Sid           string `json:"sid"`
}

type RequestTokenNumberResp struct {
	Sid    string `json:"sid"`
	Msisdn string `json:"msisdn"`
}

type RequestTokenNissResp struct {
	Sid string `json:"sid"`
}

type LookupReq struct {
	Niss string `json:"niss"`
}

type LookupResp struct {
	Medium  string `json:"medium"`
	Address string `json:"address"`
	MXID    string `json:"mxid"`
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
	user := prov.bridge.GetUserByMXID(id.UserID("@foodtosharebot:" + prov.bridge.AS.HomeserverDomain))

	jid, displayName := prov.resolveIdentifier(w, r)
	if user == nil {
		return
	}

	portal, puppet, justCreated, err := user.StartUser(jid, displayName, "provisioning API New User")

	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, Error{
			Error: fmt.Sprintf("Failed to create portal: %v", err),
		})
	}

	status := http.StatusOK

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

func (prov *ProvisioningAPI) SyncNiss(w http.ResponseWriter, r *http.Request) {
	// user := prov.bridge.GetUserByMXID(id.UserID("@kevindom:localhost"))
	jid, niss := prov.resolveIdentifierNiss(w, r)

	fmt.Println("JID: " + jid)
	fmt.Println("Niss: " + niss)
	puppet := prov.bridge.GetPuppetByJID(jid)

	fmt.Println("MXID: " + puppet.MXID)

	httpposturl := IdentityServerURL + "/_matrix/identity/api/v1/validate/niss/requestToken"

	requestToken := RequestTokenNiss{
		Client_Secret: "uij4hri2n4h42jn34k2n4nmwenjhjhnrj3n4j1b4",
		Niss:          niss,
		Send_attempt:  1,
	}

	data, err := json.Marshal(requestToken)
	if err != nil {
		log.Fatal(err)
	}
	reader := bytes.NewBuffer(data)

	response, errorPost := http.Post(httpposturl, "application/json", reader)
	if errorPost != nil {
		panic(errorPost)
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		respbody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			//Failed to read response.
			panic(err)
		}
		var rtr RequestTokenNissResp

		err = json.Unmarshal(respbody, &rtr)

		newRequest := BindToken{
			Client_Secret: "uij4hri2n4h42jn34k2n4nmwenjhjhnrj3n4j1b4",
			Mxid:          string(puppet.MXID),
			Sid:           rtr.Sid,
		}

		prov.bind(w, r, newRequest)
		// status := http.StatusOK
		// jsonResponse(w, status, rtr)
	}
}

func (prov *ProvisioningAPI) Lookup(w http.ResponseWriter, r *http.Request) {
	niss := prov.resolveIdentifierNissToLookup(w, r)

	httpposturl := IdentityServerURL + "/_matrix/identity/api/v1/lookup?medium=niss&address=" + niss

	response, errorPost := http.Get(httpposturl)
	if errorPost != nil {
		panic(errorPost)
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		respbody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			//Failed to read response.
			panic(err)
		}
		var lr LookupResp

		err = json.Unmarshal(respbody, &lr)

		if lr.MXID == "" {
			lr.MXID = "None"
			lr.Medium = "niss"
			lr.Address = "Not in the system"
		}

		status := http.StatusOK
		jsonResponse(w, status, lr)
	}
}

/*  This was only necessary with mxisd server, with sydent the bind is done automatically        */
func (prov *ProvisioningAPI) bind(w http.ResponseWriter, r *http.Request, newRequest BindToken) {

	httpposturl := IdentityServerURL + "/_matrix/identity/api/v1/3pid/bind"

	data, err := json.Marshal(newRequest)
	if err != nil {
		log.Fatal(err)
	}
	reader := bytes.NewBuffer(data)

	response, errorPost := http.Post(httpposturl, "application/json", reader)
	if errorPost != nil {
		panic(errorPost)
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		respbody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			//Failed to read response.
			panic(err)
		}
		// var rtr RequestTokenEmailResp

		// err = json.Unmarshal(respbody, &rtr)

		status := http.StatusOK
		jsonResponse(w, status, respbody)
	}
}

func (prov *ProvisioningAPI) resolveIdentifierNiss(w http.ResponseWriter, r *http.Request) (string, string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return "", ""
	}

	defer r.Body.Close()
	var es NissToSync

	err = json.Unmarshal(b, &es)

	if err != nil {
		jsonResponse(w, http.StatusNotFound, Error{
			Error:   fmt.Sprintf("Niss not found"),
			ErrCode: "Not found!",
		})
		return "", ""
	}

	return es.Jid, es.Niss
}

func (prov *ProvisioningAPI) resolveIdentifierNissToLookup(w http.ResponseWriter, r *http.Request) string {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return ""
	}

	defer r.Body.Close()
	var niss LookupReq

	err = json.Unmarshal(b, &niss)

	if err != nil {
		jsonResponse(w, http.StatusNotFound, Error{
			Error:   fmt.Sprintf("Niss not found"),
			ErrCode: "Not found!",
		})
		return ""
	}

	return niss.Niss
}

func (prov *ProvisioningAPI) resolveIdentifier(w http.ResponseWriter, r *http.Request) (string, string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return "", ""
	}

	defer r.Body.Close()
	var u NewUser

	err = json.Unmarshal(b, &u)

	if err != nil {
		jsonResponse(w, http.StatusNotFound, Error{
			Error:   fmt.Sprintf("User not found"),
			ErrCode: "Not found!",
		})
		return "", ""
	}
	return u.Jid, u.DisplayName
}

func jsonResponse(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
