package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"maunium.net/go/mautrix"
	appservice "maunium.net/go/mautrix-appservice"
)

type AutosavingStateStore struct {
	appservice.StateStore
	Path string
}

func NewAutosavingStateStore(path string) *AutosavingStateStore {
	return &AutosavingStateStore{
		StateStore: appservice.NewBasicStateStore(),
		Path:       path,
	}
}

func (store *AutosavingStateStore) Save() error {
	store.RLock()
	defer store.RUnlock()
	data, err := json.Marshal(store.StateStore)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(store.Path, data, 0600)
}

func (store *AutosavingStateStore) Load() error {
	store.Lock()
	defer store.Unlock()
	data, err := ioutil.ReadFile(store.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, store.StateStore)
}

func (store *AutosavingStateStore) MarkRegistered(userID string) {
	store.StateStore.MarkRegistered(userID)
	store.Save()
}

func (store *AutosavingStateStore) SetMembership(roomID, userID string, membership mautrix.Membership) {
	store.StateStore.SetMembership(roomID, userID, membership)
	store.Save()
}

func (store *AutosavingStateStore) SetPowerLevels(roomID string, levels *mautrix.PowerLevels) {
	store.StateStore.SetPowerLevels(roomID, levels)
	store.Save()
}
