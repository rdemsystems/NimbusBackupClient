package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"pbscommon"
)

// Client-side encryption (pbscommon/crypt.go), configured per PBS server: the
// server entry carries a proxmox-backup-client key file in config.json
// ("encryption_key"). The key is stored unprotected (kdf null, as PVE stores
// its storage keys) so scheduled backups run unattended; config.json already
// holds the PBS token and sits in the restricted configuration folder. The
// same key file restores with proxmox-backup-client or PVE.
//
// This file is shared by the GUI and service builds; the GUI-only management
// methods (generate / import / export) live in encryption_gui.go.

// parseStoredKey accepts the key file as a JSON object, or as a JSON string
// holding that object (handy when pasting a key into a provisioned config).
func parseStoredKey(raw json.RawMessage) (*pbscommon.KeyConfig, error) {
	data := []byte(raw)
	if strings.HasPrefix(strings.TrimSpace(string(raw)), `"`) {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("clé de chiffrement invalide: %w", err)
		}
		data = []byte(s)
	}
	return pbscommon.ParseKeyConfig(data)
}

func hasStoredKey(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "null" && s != `""`
}

// cryptFromStoredKey unlocks a configured key; (nil, nil) when none is set.
// A configured but unusable key is an error: callers must fail rather than
// silently back up unencrypted.
func cryptFromStoredKey(raw json.RawMessage) (*pbscommon.CryptConfig, error) {
	if !hasStoredKey(raw) {
		return nil, nil
	}
	kc, err := parseStoredKey(raw)
	if err != nil {
		return nil, err
	}
	if kc.NeedsPassphrase() {
		return nil, errors.New("la clé de chiffrement configurée est protégée par une phrase secrète — importez-la depuis l'interface pour l'enregistrer déverrouillée")
	}
	key, err := kc.Decrypt(nil)
	if err != nil {
		return nil, fmt.Errorf("clé de chiffrement invalide: %w", err)
	}
	return pbscommon.NewCryptConfig(key)
}

// storedKeyFingerprint is the short fingerprint of a configured key, "" when
// none is set or it is unusable.
func storedKeyFingerprint(raw json.RawMessage) string {
	crypt, err := cryptFromStoredKey(raw)
	if err != nil || crypt == nil {
		return ""
	}
	return crypt.ShortFingerprint()
}

// applyEncryptionKey sets client.Crypt from a configured key (no-op without one).
func applyEncryptionKey(client *pbscommon.PBSClient, raw json.RawMessage) error {
	crypt, err := cryptFromStoredKey(raw)
	if err != nil {
		return err
	}
	client.Crypt = crypt
	return nil
}

// normalizeKeyFile unlocks a key file (with passphrase if it is protected) and
// re-wraps it unprotected for storage, keeping its creation date and hint.
func normalizeKeyFile(keyJSON, passphrase []byte) (json.RawMessage, error) {
	kc, err := pbscommon.ParseKeyConfig(keyJSON)
	if err != nil {
		return nil, err
	}
	key, err := kc.Decrypt(passphrase)
	if err != nil {
		return nil, err
	}
	stored, err := pbscommon.NewKeyConfig(key, nil, kc.Hint)
	if err != nil {
		return nil, err
	}
	if kc.Created != "" {
		stored.Created = kc.Created
	}
	return json.Marshal(stored)
}

// setEncryptionKeyLocal writes (or, with an empty key, removes) the key of PBS
// server id in THIS process's config.json. It works on a fresh, detached copy
// of the file — so a long-running service never writes back a stale cached
// config over newer GUI/provisioning edits — and publishes it as the active
// config only once saved, so this process never uses a key that is not on
// disk (or drops one that still is).
func (a *App) setEncryptionKeyLocal(id string, key json.RawMessage) error {
	if id == "" {
		return errors.New("identifiant de serveur PBS requis")
	}
	if hasStoredKey(key) {
		if _, err := cryptFromStoredKey(key); err != nil {
			return err
		}
	} else {
		key = nil
	}
	cfg := LoadConfig()
	current, err := cfg.GetPBSServer(id)
	if err != nil {
		return err
	}
	updated := *current
	updated.EncryptionKey = key
	if err := cfg.UpdatePBSServer(&updated); err != nil {
		return fmt.Errorf("impossible d'enregistrer la clé de chiffrement (config.json non accessible en écriture ?): %w", err)
	}
	a.config = cfg
	writeDebugLog(fmt.Sprintf("Encryption key for PBS server %q updated (set=%v)", id, key != nil))
	return nil
}

// SetServerEncryptionKey is the local-API entrypoint the service exposes so
// the unprivileged GUI can store or remove a server's key through the
// privileged service, the single writer of config.json. keyJSON is the key
// file (unprotected); empty removes the key.
func (a *App) SetServerEncryptionKey(id, keyJSON string) error {
	writeDebugLog(fmt.Sprintf("SetServerEncryptionKey(%s) called (service-side write)", id))
	return a.setEncryptionKeyLocal(id, json.RawMessage(keyJSON))
}
