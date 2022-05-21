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

type ProvisioningAPI struct {
	bridge *Bridge
	log    log.Logger
}

func (prov *ProvisioningAPI) Init() {
	prov.log = prov.bridge.Log.Sub("Provisiong")

	prov.log.Debugfln("Enabling provisioning API at", prov.bridge.Config.AppService.Provisioning.SegmentKey, prov.log)

	r := prov.bridge.AS.Router.PathPrefix(prov.bridge.Config.AppService.Provisioning.Prefix).Subrouter()
	r.HandleFunc("/v1/startUser", prov.StartUser).Methods(http.MethodPost)
	r.HandleFunc("/v1/syncNumber", prov.SyncNumber).Methods(http.MethodPost)
	r.HandleFunc("/v1/syncEmail", prov.SyncEmail).Methods(http.MethodPost)
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

type EmailToSync struct {
	Jid   string `json:"jid"`
	Email string `json:"email"`
}

type RequestTokenNumber struct {
	Client_Secret string `json:"client_secret"`
	Country       string `json:"country"`
	Phone_number  string `json:"phone_number"`
	Send_attempt  int    `json:"send_attempt"`
}

type RequestTokenEmail struct {
	Client_Secret string `json:"client_secret"`
	Email         string `json:"email"`
	Send_attempt  int    `json:"send_attempt"`
}

type RequestTokenNumberResp struct {
	Sid    string `json:"sid"`
	Msisdn string `json:"msisdn"`
}

type RequestTokenEmailResp struct {
	Sid string `json:"sid"`
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

func (prov *ProvisioningAPI) SyncNumber(w http.ResponseWriter, r *http.Request) {
	// user := prov.bridge.GetUserByMXID(id.UserID("@kevindom:localhost"))
	jid, number := prov.resolveIdentifierNumber(w, r)

	fmt.Println("JID: " + jid)
	fmt.Println("Number: " + number)
	// puppet := prov.bridge.GetPuppetByJID(jid)

	httpposturl := "http://localhost:8090/_matrix/identity/api/v1/validate/msisdn/requestToken"

	requestToken := RequestTokenNumber{
		Client_Secret: "uij4hri2n4h42jn34k2n4nmwenjhjhnrj3n4j1b4",
		Country:       "pt",
		Phone_number:  number,
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
		var rtr RequestTokenNumberResp

		err = json.Unmarshal(respbody, &rtr)
		status := http.StatusOK
		jsonResponse(w, status, rtr)
	}
}

func (prov *ProvisioningAPI) SyncEmail(w http.ResponseWriter, r *http.Request) {
	// user := prov.bridge.GetUserByMXID(id.UserID("@kevindom:localhost"))
	jid, email := prov.resolveIdentifierEmail(w, r)

	fmt.Println("JID: " + jid)
	fmt.Println("Email: " + email)
	// puppet := prov.bridge.GetPuppetByJID(jid)

	httpposturl := "http://localhost:8090/_matrix/identity/api/v1/validate/email/requestToken"

	requestToken := RequestTokenEmail{
		Client_Secret: "uij4hri2n4h42jn34k2n4nmwenjhjhnrj3n4j1b4",
		Email:         email,
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
		var rtr RequestTokenEmailResp

		err = json.Unmarshal(respbody, &rtr)
		status := http.StatusOK
		jsonResponse(w, status, rtr)
	}
}

func (prov *ProvisioningAPI) resolveIdentifierEmail(w http.ResponseWriter, r *http.Request) (string, string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return "", ""
	}

	defer r.Body.Close()
	var es EmailToSync

	err = json.Unmarshal(b, &es)

	if err != nil {
		jsonResponse(w, http.StatusNotFound, Error{
			Error:   fmt.Sprintf("Email not found"),
			ErrCode: "Not found!",
		})
		return "", ""
	}

	return es.Jid, es.Email
}

func (prov *ProvisioningAPI) resolveIdentifierNumber(w http.ResponseWriter, r *http.Request) (string, string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return "", ""
	}

	defer r.Body.Close()
	var ns NumberToSync

	err = json.Unmarshal(b, &ns)

	if err != nil {
		jsonResponse(w, http.StatusNotFound, Error{
			Error:   fmt.Sprintf("Number not found"),
			ErrCode: "Not found!",
		})
		return "", ""
	}

	return ns.Jid, ns.Number
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
