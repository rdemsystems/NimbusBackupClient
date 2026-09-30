//go:build !service

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tizbac/proxmoxbackupclient_go/gui/api"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"pbscommon"
)

// EncryptionKeyInfo describes a PBS server's client-side encryption key for
// the frontend, without the key itself.
type EncryptionKeyInfo struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint"` // short form, e.g. "0e:ab:d4:46:0b:6e:b9:ca"
	Created     string `json:"created,omitempty"`
	Error       string `json:"error,omitempty"` // set when a stored key is unusable
}

func encryptionKeyInfo(raw json.RawMessage) *EncryptionKeyInfo {
	info := &EncryptionKeyInfo{Enabled: hasStoredKey(raw)}
	if !info.Enabled {
		return info
	}
	crypt, err := cryptFromStoredKey(raw)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.Fingerprint = crypt.ShortFingerprint()
	if kc, err := parseStoredKey(raw); err == nil {
		info.Created = kc.Created
	}
	return info
}

// storeEncryptionKey persists a server key. When a service owns config.json
// the write is delegated to it (like certificate pinning), since this
// unprivileged process cannot overwrite the service-owned file.
func (a *App) storeEncryptionKey(id string, key json.RawMessage) error {
	if !a.isServiceProcess && a.mode == api.ModeService && a.apiClient != nil {
		writeDebugLog(fmt.Sprintf("storeEncryptionKey(%s): delegating write to service", id))
		if err := a.apiClient.SetEncryptionKey(id, string(key)); err != nil {
			return err
		}
		a.ReloadConfig()
		return nil
	}
	return a.setEncryptionKeyLocal(id, key)
}

func (a *App) serverKey(id string) (json.RawMessage, error) {
	if id == "" {
		return nil, errors.New("identifiant de serveur PBS requis")
	}
	pbs, err := a.config.GetPBSServer(id)
	if err != nil {
		return nil, err
	}
	return pbs.EncryptionKey, nil
}

// GetEncryptionKeyInfo reports whether PBS server id encrypts its backups.
func (a *App) GetEncryptionKeyInfo(id string) (*EncryptionKeyInfo, error) {
	raw, err := a.serverKey(id)
	if err != nil {
		return nil, err
	}
	return encryptionKeyInfo(raw), nil
}

// GenerateEncryptionKey creates a new random key for PBS server id. It refuses
// to replace an existing key: backups made with it would become unrestorable
// unless it was exported, so the user must remove it explicitly first.
func (a *App) GenerateEncryptionKey(id string) (*EncryptionKeyInfo, error) {
	writeDebugLog(fmt.Sprintf("GenerateEncryptionKey(%s) called", id))
	raw, err := a.serverKey(id)
	if err != nil {
		return nil, err
	}
	if hasStoredKey(raw) {
		return nil, errors.New("ce serveur a déjà une clé de chiffrement — supprimez-la d'abord (après l'avoir exportée)")
	}
	key, err := pbscommon.GenerateKey()
	if err != nil {
		return nil, err
	}
	kc, err := pbscommon.NewKeyConfig(key, nil, "")
	if err != nil {
		return nil, err
	}
	stored, err := json.Marshal(kc)
	if err != nil {
		return nil, err
	}
	if err := a.storeEncryptionKey(id, stored); err != nil {
		return nil, err
	}
	return encryptionKeyInfo(stored), nil
}

// ImportEncryptionKey installs an existing key file (e.g. from
// `proxmox-backup-client key create` or PVE) on PBS server id. passphrase is
// only needed for a protected key file; the key is stored unlocked so
// scheduled backups can use it.
func (a *App) ImportEncryptionKey(id, keyJSON, passphrase string) (*EncryptionKeyInfo, error) {
	writeDebugLog(fmt.Sprintf("ImportEncryptionKey(%s) called", id))
	raw, err := a.serverKey(id)
	if err != nil {
		return nil, err
	}
	if hasStoredKey(raw) {
		return nil, errors.New("ce serveur a déjà une clé de chiffrement — supprimez-la d'abord (après l'avoir exportée)")
	}
	stored, err := normalizeKeyFile([]byte(strings.TrimSpace(keyJSON)), []byte(passphrase))
	if errors.Is(err, pbscommon.ErrKeyNeedsPassphrase) {
		return nil, errors.New("cette clé est protégée par une phrase secrète — saisissez-la pour l'importer")
	}
	if err != nil {
		return nil, err
	}
	if err := a.storeEncryptionKey(id, stored); err != nil {
		return nil, err
	}
	return encryptionKeyInfo(stored), nil
}

// ExportEncryptionKey saves PBS server id's key file through a native save
// dialog and returns the chosen path ("" if cancelled). With a passphrase the
// exported file is protected (scrypt, like proxmox-backup-client); either way
// it restores with the official client or PVE.
func (a *App) ExportEncryptionKey(id, passphrase string) (string, error) {
	writeDebugLog(fmt.Sprintf("ExportEncryptionKey(%s) called", id))
	out, err := a.exportKeyFile(id, passphrase)
	if err != nil {
		return "", err
	}
	if a.ctx == nil || a.isServiceProcess {
		return "", errors.New("boîte de dialogue indisponible — utilisez « Afficher la clé » et copiez-la")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Exporter la clé de chiffrement",
		DefaultFilename: fmt.Sprintf("encryption-key-%s.json", id),
		Filters:         []runtime.FileFilter{{DisplayName: "Clé de chiffrement PBS (*.json)", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return "", fmt.Errorf("écriture de %s impossible: %w", path, err)
	}
	return path, nil
}

// GetEncryptionKeyFile returns PBS server id's key file as text, for copying
// it somewhere safe (password manager, paper). Same passphrase rule as
// ExportEncryptionKey.
func (a *App) GetEncryptionKeyFile(id, passphrase string) (string, error) {
	writeDebugLog(fmt.Sprintf("GetEncryptionKeyFile(%s) called", id))
	out, err := a.exportKeyFile(id, passphrase)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (a *App) exportKeyFile(id, passphrase string) ([]byte, error) {
	raw, err := a.serverKey(id)
	if err != nil {
		return nil, err
	}
	if !hasStoredKey(raw) {
		return nil, errors.New("aucune clé de chiffrement configurée pour ce serveur")
	}
	kc, err := parseStoredKey(raw)
	if err != nil {
		return nil, err
	}
	key, err := kc.Decrypt(nil)
	if err != nil {
		return nil, err
	}
	out := kc
	if passphrase != "" {
		if out, err = pbscommon.NewKeyConfig(key, []byte(passphrase), kc.Hint); err != nil {
			return nil, err
		}
		if kc.Created != "" {
			out.Created = kc.Created
		}
	}
	return out.Marshal()
}

// RemoveEncryptionKey stops encrypting PBS server id's backups. Existing
// encrypted snapshots then need an exported copy of the key to be restored.
func (a *App) RemoveEncryptionKey(id string) error {
	writeDebugLog(fmt.Sprintf("RemoveEncryptionKey(%s) called", id))
	if _, err := a.serverKey(id); err != nil {
		return err
	}
	return a.storeEncryptionKey(id, nil)
}
