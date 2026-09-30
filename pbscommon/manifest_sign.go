package pbscommon

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// Encrypted backups keep index.json.blob readable but signed (proxmox-backup:
// pbs-datastore/src/manifest.rs): "signature" is the hex HMAC-SHA256
// (id_key) of the canonical JSON of the manifest without its "unprotected"
// and "signature" members, and unprotected["key-fingerprint"] names the key.

// signedManifest is the PBS manifest schema (pbs-datastore BackupManifest).
// PBS rewrites index.json.blob through that typed struct when it finishes the
// backup (update_manifest), silently dropping any other top-level member such
// as our "comment" -- so only these members may be covered by the signature.
type signedManifest struct {
	BackupType  string      `json:"backup-type"`
	BackupID    string      `json:"backup-id"`
	BackupTime  int64       `json:"backup-time"`
	Files       []File      `json:"files"`
	Unprotected Unprotected `json:"unprotected"`
	Signature   any         `json:"signature"`
}

func toSignedManifest(m BackupManifest) signedManifest {
	files := m.Files
	if files == nil {
		files = []File{} // canonical JSON rejects null
	}
	return signedManifest{
		BackupType:  m.BackupType,
		BackupID:    m.BackupID,
		BackupTime:  m.BackupTime,
		Files:       files,
		Unprotected: m.Unprotected,
	}
}

// ManifestSignature computes the PBS signature of manifest m.
func ManifestSignature(m BackupManifest, crypt *CryptConfig) ([32]byte, error) {
	raw, err := json.Marshal(toSignedManifest(m))
	if err != nil {
		return [32]byte{}, err
	}
	obj, err := decodeJSONObject(raw)
	if err != nil {
		return [32]byte{}, err
	}
	return manifestObjectSignature(obj, crypt)
}

func manifestObjectSignature(obj map[string]any, crypt *CryptConfig) ([32]byte, error) {
	signed := make(map[string]any, len(obj))
	for k, v := range obj {
		if k != "unprotected" && k != "signature" {
			signed[k] = v
		}
	}
	var buf bytes.Buffer
	if err := writeCanonicalJSON(&buf, signed); err != nil {
		return [32]byte{}, err
	}
	return crypt.AuthTag(buf.Bytes()), nil
}

// EncodeManifest serializes m for upload. Unencrypted manifests are written
// as before; encrypted ones are restricted to the PBS schema (see
// signedManifest), signed, and record the key fingerprint.
func EncodeManifest(m BackupManifest, crypt *CryptConfig) ([]byte, error) {
	m.Signature = nil
	m.Unprotected.KeyFingerprint = ""
	if crypt == nil {
		return json.Marshal(m)
	}
	sm := toSignedManifest(m)
	sig, err := ManifestSignature(m, crypt)
	if err != nil {
		return nil, fmt.Errorf("sign manifest: %w", err)
	}
	sm.Signature = hex.EncodeToString(sig[:])
	sm.Unprotected.KeyFingerprint = crypt.FingerprintString()
	return json.Marshal(sm)
}

// ManifestKeyFingerprint returns the key fingerprint recorded in a raw
// manifest, or "" for an unencrypted backup.
func ManifestKeyFingerprint(raw []byte) (string, error) {
	var m struct {
		Unprotected struct {
			KeyFingerprint string `json:"key-fingerprint"`
		} `json:"unprotected"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("parse manifest: %w", err)
	}
	return m.Unprotected.KeyFingerprint, nil
}

// VerifyManifest checks a raw (decoded) manifest against crypt, like
// BackupManifest::from_data + check_fingerprint in PBS: a manifest recording
// a key fingerprint needs the matching key; whenever a key is given, a
// present signature is verified whatever the (unsigned) fingerprint says,
// and a manifest with a fingerprint but no signature is rejected. An
// unencrypted, unsigned manifest is accepted with or without a key.
func VerifyManifest(raw []byte, crypt *CryptConfig) error {
	obj, err := decodeJSONObject(raw)
	if err != nil {
		return err
	}
	fp := ""
	if unprot, ok := obj["unprotected"].(map[string]any); ok {
		fp, _ = unprot["key-fingerprint"].(string)
	}
	sig, _ := obj["signature"].(string)

	if crypt == nil {
		if fp != "" {
			return ErrEncryptedNoKey
		}
		return nil
	}
	if fp != "" {
		want, err := ParseFingerprint(fp)
		if err != nil {
			return err
		}
		if crypt.Fingerprint() != want {
			return fmt.Errorf("wrong encryption key: backup was made with key %s, provided key is %s",
				FormatFingerprint(want[:8]), crypt.ShortFingerprint())
		}
	}
	if sig == "" {
		if fp != "" {
			return errors.New("encrypted manifest is not signed")
		}
		return nil
	}
	expected, err := manifestObjectSignature(obj, crypt)
	if err != nil {
		return err
	}
	if sig != hex.EncodeToString(expected[:]) {
		return errors.New("wrong signature in manifest")
	}
	return nil
}

func decodeJSONObject(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // keep integers exactly as written (no float64 round trip)
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if obj == nil {
		return nil, errors.New("parse manifest: not a JSON object")
	}
	return obj, nil
}

// writeCanonicalJSON mirrors proxmox_serde::json::write_canonical_json:
// compact, object keys sorted bytewise, strings/numbers as serde_json writes
// them, and null rejected.
func writeCanonicalJSON(buf *bytes.Buffer, v any) error {
	switch t := v.(type) {
	case nil:
		return errors.New("canonical json: unexpected null value")
	case bool:
		if t {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case json.Number:
		buf.WriteString(t.String())
	case string:
		writeCanonicalString(buf, t)
	case []any:
		buf.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonicalJSON(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeCanonicalString(buf, k)
			buf.WriteByte(':')
			if err := writeCanonicalJSON(buf, t[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("canonical json: unsupported type %T", v)
	}
	return nil
}

// writeCanonicalString escapes exactly like serde_json: short escapes for
// \" \\ \b \f \n \r \t, \u00xx (lowercase) for other control characters, and
// everything else -- including <, >, &, U+2028/U+2029 -- written raw, unlike
// encoding/json.
func writeCanonicalString(buf *bytes.Buffer, s string) {
	const hexDigits = "0123456789abcdef"
	buf.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if c < 0x20 {
				buf.WriteString(`\u00`)
				buf.WriteByte(hexDigits[c>>4])
				buf.WriteByte(hexDigits[c&0xf])
			} else {
				buf.WriteByte(c)
			}
		}
	}
	buf.WriteByte('"')
}
