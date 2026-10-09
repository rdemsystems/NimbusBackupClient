package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"pbscommon"
	"security"
)

// Safekeeping copies of an encryption key file: the key as text with its QR
// code, a printable paper key (`proxmox-backup-client key paperkey` format),
// and the way back, a key file rebuilt from text copied or scanned off such a
// copy. The GUI only ever reads the key file the user points it at.

// EncryptionKeyCopy is a key file as shown for safekeeping: its text and the
// QR code of that text (an SVG document), both from the same export so a
// passphrase-protected copy matches its QR code.
type EncryptionKeyCopy struct {
	Key   string `json:"key"`
	QRSVG string `json:"qr_svg"`
}

// PaperKeyLabels is the localized text of the printable paper key page.
type PaperKeyLabels struct {
	Lang  string   `json:"lang"`
	Title string   `json:"title"`
	Notes []string `json:"notes"`
}

// keyCopy loads the key file at path for a copy. With a passphrase, an
// unprotected key is re-wrapped with it (scrypt, like `key create`) so the
// copy is useless without the passphrase; a key file that is already
// protected is copied as is.
func keyCopy(path, passphrase string) (*pbscommon.PaperKey, *pbscommon.KeyConfig, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil, errors.New("aucun fichier de cle configure")
	}
	kc, err := pbscommon.LoadKeyConfig(path)
	if err != nil {
		return nil, nil, err
	}
	out := kc
	if passphrase != "" && kc.KDF == nil {
		crypt, err := kc.CryptConfig(nil)
		if err != nil {
			return nil, nil, err
		}
		if out, err = pbscommon.ProtectKeyConfig(crypt, []byte(passphrase), kc.Hint); err != nil {
			return nil, nil, err
		}
		if kc.Created != "" {
			out.Created = kc.Created
		}
	}
	pk, err := pbscommon.NewPaperKey(out)
	if err != nil {
		return nil, nil, err
	}
	return pk, kc, nil
}

// GetEncryptionKeyCopy returns the key file at path and its QR code, for
// copying it somewhere safe (password manager, paper, phone). The QR code
// holds the key file as proxmox-backup-client prints it, so scanning it gives
// an importable key.
func (a *App) GetEncryptionKeyCopy(path, passphrase string) (*EncryptionKeyCopy, error) {
	writeDebugLog("GetEncryptionKeyCopy called")
	pk, _, err := keyCopy(path, passphrase)
	if err != nil {
		return nil, err
	}
	return &EncryptionKeyCopy{Key: pk.JSON, QRSVG: pk.QRSVG}, nil
}

// ExportEncryptionPaperKey saves the key file at path as a printable HTML page
// (key text + QR code) through a native save dialog and returns the chosen
// path ("" if cancelled).
func (a *App) ExportEncryptionPaperKey(path, passphrase string, labels PaperKeyLabels) (string, error) {
	writeDebugLog("ExportEncryptionPaperKey called")
	if a.ctx == nil {
		return "", fmt.Errorf("interface graphique non initialisee")
	}
	pk, kc, err := keyCopy(path, passphrase)
	if err != nil {
		return "", err
	}
	subject := filepath.Base(strings.TrimSpace(path))
	if kc.Fingerprint != "" {
		subject += " — " + kc.Fingerprint
	}
	page := pk.HTML(pbscommon.PaperKeyPage{Lang: labels.Lang, Title: labels.Title, Subject: subject, Notes: labels.Notes})
	out, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           labels.Title,
		DefaultFilename: "paperkey.html",
		Filters:         []runtime.FileFilter{{DisplayName: "HTML (*.html)", Pattern: "*.html"}},
	})
	if err != nil || out == "" {
		return "", err
	}
	if err := os.WriteFile(out, page, 0o600); err != nil {
		return "", fmt.Errorf("ecriture de %s: %w", out, err)
	}
	return out, nil
}

// ImportEncryptionKeyText rebuilds a key file from text: a key file pasted
// as is, the payload of a scanned QR code, or the text copied off a paper key
// (BEGIN/END markers included). A passphrase-protected key is unlocked with
// passphrase, because the GUI can only use unprotected key files. The new
// file is written where the user chooses (never over an existing file) and
// its info is returned, Path included, for the key field.
func (a *App) ImportEncryptionKeyText(text, passphrase string) (EncryptionKeyInfo, error) {
	writeDebugLog("ImportEncryptionKeyText called")
	kc, err := pbscommon.ParseKeyConfig([]byte(text))
	if err != nil {
		return EncryptionKeyInfo{}, err
	}
	if kc.KDF != nil && passphrase == "" {
		return EncryptionKeyInfo{}, errors.New("cette cle est protegee par une phrase de passe: saisissez-la pour l'importer")
	}
	crypt, err := kc.CryptConfig([]byte(passphrase))
	if err != nil {
		return EncryptionKeyInfo{}, err
	}
	dest, err := a.OpenEncryptionKeySaveDialog()
	if err != nil || dest == "" {
		return EncryptionKeyInfo{}, err
	}
	if err := a.writeNewKeyFile(dest, crypt); err != nil {
		return EncryptionKeyInfo{}, err
	}
	info := inspectKeyFile(dest)
	if !info.Usable {
		return EncryptionKeyInfo{}, fmt.Errorf("la cle %s a ete ecrite mais ne peut pas etre relue: %s", dest, info.Reason)
	}
	writeDebugLog(fmt.Sprintf("ImportEncryptionKeyText: wrote %s (fingerprint %s)", dest, info.Fingerprint))
	return info, nil
}

// writeNewKeyFile writes crypt's key, unprotected, to a NEW file at path,
// with the same checks as GenerateEncryptionKeyFile: an existing file is never
// overwritten, since it may be the only key able to decrypt other snapshots.
func (a *App) writeNewKeyFile(path string, crypt *pbscommon.CryptConfig) error {
	path = strings.TrimSpace(path)
	if err := security.ValidatePath(path); err != nil {
		return fmt.Errorf("chemin de cle invalide: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("le fichier %s existe deja: il n'est jamais ecrase, choisissez un autre nom", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("verification de %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creation du dossier %s: %w", filepath.Dir(path), err)
	}
	if err := pbscommon.SaveKeyFile(path, crypt); err != nil {
		return fmt.Errorf("ecriture de %s: %w", path, err)
	}
	return nil
}
