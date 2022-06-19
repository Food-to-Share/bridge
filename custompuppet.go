package main

import (
	"github.com/pkg/errors"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/id"
)

var (
	ErrNoCustomMXID    = errors.New("No custom mxid set")
	ErrMismatchingMXID = errors.New("Id does not match custom mxid")
)

func (puppet *Puppet) newCustomIntent() (*appservice.IntentAPI, error) {
	if len(puppet.CustomMXID) == 0 {
		return nil, ErrNoCustomMXID
	}
	client, err := mautrix.NewClient(puppet.bridge.AS.HomeserverURL, puppet.CustomMXID, puppet.AccessToken)
	if err != nil {
		return nil, err
	}

	client.Store = puppet

	ia := puppet.bridge.AS.NewIntentAPI("custom")
	ia.Client = client
	ia.Localpart, _, _ = puppet.CustomMXID.Parse()
	ia.UserID = puppet.CustomMXID
	ia.IsCustomPuppet = true
	return ia, nil

}

func (puppet *Puppet) clearCustomMXID() {
	puppet.CustomMXID = ""
	puppet.AccessToken = ""
	puppet.customIntent = nil
}

func (puppet *Puppet) StartCustomMXID() error {
	if len(puppet.CustomMXID) == 0 {
		puppet.clearCustomMXID()
		return nil
	}

	intent, err := puppet.newCustomIntent()
	if err != nil {
		puppet.clearCustomMXID()
		return err
	}
	resp, err := intent.Whoami()
	if err != nil {
		if errors.Is(err, mautrix.MUnknownToken) {
			puppet.clearCustomMXID()
			return err
		}
		intent.AccessToken = puppet.AccessToken
	} else if resp.UserID != puppet.CustomMXID {
		puppet.clearCustomMXID()
		return ErrMismatchingMXID
	}
	puppet.customIntent = intent
	puppet.startSyncing()
	return nil
}

func (puppet *Puppet) startSyncing() {
	go func() {
		err := puppet.customIntent.Sync()
		if err != nil {
			puppet.log.Errorln("Fatal error syncing:", err)
		}
	}()
}

func (puppet *Puppet) stopSyncing() {
	puppet.customIntent.StopSync()
}

func (puppet *Puppet) SaveFilterID(_ id.UserID, _ string)    {}
func (puppet *Puppet) SaveNextBatch(_ id.UserID, nbt string) { puppet.NextBatch = nbt; puppet.Update() }
func (puppet *Puppet) SaveRoom(_ *mautrix.Room)              {}
func (puppet *Puppet) LoadFilterID(_ id.UserID) string       { return "" }
func (puppet *Puppet) LoadNextBatch(_ id.UserID) string      { return puppet.NextBatch }
func (puppet *Puppet) LoadRoom(_ id.RoomID) *mautrix.Room    { return nil }
