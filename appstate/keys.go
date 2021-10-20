// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package appstate

import (
	"encoding/base64"
	"sync"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/util/hkdfutil"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type WAPatchName string

const (
	WAPatchCriticalBlock      WAPatchName = "critical_block"
	WAPatchCriticalUnblockLow WAPatchName = "critical_unblock_low"
	WAPatchRegularLow         WAPatchName = "regular_low"
	WAPatchRegularHigh        WAPatchName = "regular_high"
	WAPatchRegular            WAPatchName = "regular"
)

type Processor struct {
	keyCache     map[string]ExpandedAppStateKeys
	keyCacheLock sync.Mutex
	Store        *store.Device
	Log          waLog.Logger
}

func NewProcessor(store *store.Device, log waLog.Logger) *Processor {
	return &Processor{
		keyCache: make(map[string]ExpandedAppStateKeys),
		Store:    store,
		Log:      log,
	}
}

type ExpandedAppStateKeys struct {
	Index           []byte
	ValueEncryption []byte
	ValueMAC        []byte
	SnapshotMAC     []byte
	PatchMAC        []byte
}

func expandAppStateKeys(keyData []byte) (keys ExpandedAppStateKeys) {
	appStateKeyExpanded := hkdfutil.SHA256(keyData, nil, []byte("WhatsApp Mutation Keys"), 160)
	return ExpandedAppStateKeys{appStateKeyExpanded[0:32], appStateKeyExpanded[32:64], appStateKeyExpanded[64:96], appStateKeyExpanded[96:128], appStateKeyExpanded[128:160]}
}

func (proc *Processor) getAppStateKey(keyID []byte) (keys ExpandedAppStateKeys, err error) {
	keyCacheID := base64.RawStdEncoding.EncodeToString(keyID)
	var ok bool

	proc.keyCacheLock.Lock()
	defer proc.keyCacheLock.Unlock()

	keys, ok = proc.keyCache[keyCacheID]
	if !ok {
		var keyData *store.AppStateSyncKey
		keyData, err = proc.Store.AppStateKeys.GetAppStateSyncKey(keyID)
		if keyData != nil {
			keys = expandAppStateKeys(keyData.Data)
			proc.keyCache[keyCacheID] = keys
		}
	}
	return
}