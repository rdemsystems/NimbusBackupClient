package pbscommon

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Client-side encryption, byte-compatible with proxmox-backup-client
// (proxmox-backup: pbs-tools/src/crypt_config.rs). A backup encrypted here can
// be restored by the official client / PVE with the same key file, and the
// other way around. The PBS server never sees the key: it stores opaque
// encrypted blobs and can only check their CRC.
//
//   - enc_key: 32 random bytes, the AES-256-GCM key for chunks and blobs.
//   - id_key:  PBKDF2-HMAC-SHA256(enc_key, "_id_key", 10 rounds). It keys the
//     chunk digests (so identical data under different keys never
//     deduplicates) and the manifest signature.

// KeySize is the size in bytes of a PBS encryption key.
const KeySize = 32

// fingerprintInput is sha256("Proxmox Backup Encryption Key Fingerprint").
var fingerprintInput = [32]byte{
	110, 208, 239, 119, 71, 31, 255, 77, 85, 199, 168, 254, 74, 157, 182, 33,
	97, 64, 127, 19, 76, 114, 93, 223, 48, 153, 45, 37, 236, 69, 237, 38,
}

// CryptConfig holds an unlocked encryption key. A nil *CryptConfig means
// "no encryption": ChunkDigest then falls back to the plain SHA-256 PBS uses
// for unencrypted chunks, so callers need no branching.
type CryptConfig struct {
	idKey [32]byte
	aead  cipher.AEAD
}

// NewCryptConfig derives the digest/signing key and prepares the cipher.
func NewCryptConfig(key [KeySize]byte) (*CryptConfig, error) {
	idKey, err := pbkdf2.Key(sha256.New, string(key[:]), []byte("_id_key"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("derive id key: %w", err)
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	// PBS uses a 16-byte IV (not the usual 12) with an empty AAD.
	aead, err := cipher.NewGCMWithNonceSize(block, blobIVSize)
	if err != nil {
		return nil, err
	}
	c := &CryptConfig{aead: aead}
	copy(c.idKey[:], idKey)
	return c, nil
}

// ChunkDigest returns the chunk identifier: SHA256(data || id_key) when
// encrypting, plain SHA256(data) when c is nil.
func (c *CryptConfig) ChunkDigest(data []byte) [32]byte {
	if c == nil {
		return sha256.Sum256(data)
	}
	h := sha256.New()
	h.Write(data)
	h.Write(c.idKey[:]) // at the end, like PBS, to avoid length extension
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// AuthTag returns HMAC-SHA256(id_key, data), used to sign the manifest.
func (c *CryptConfig) AuthTag(data []byte) [32]byte {
	m := hmac.New(sha256.New, c.idKey[:])
	m.Write(data)
	var out [32]byte
	m.Sum(out[:0])
	return out
}

// Fingerprint identifies the key without revealing it.
func (c *CryptConfig) Fingerprint() [32]byte {
	return c.ChunkDigest(fingerprintInput[:])
}

// FingerprintString is the full fingerprint as PBS serializes it
// ("aa:bb:...", 32 bytes), e.g. in key files and manifests.
func (c *CryptConfig) FingerprintString() string {
	fp := c.Fingerprint()
	return FormatFingerprint(fp[:])
}

// ShortFingerprint is the 8-byte form PBS shows to users.
func (c *CryptConfig) ShortFingerprint() string {
	fp := c.Fingerprint()
	return FormatFingerprint(fp[:8])
}

// FormatFingerprint renders bytes as colon-separated lowercase hex.
func FormatFingerprint(b []byte) string {
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = hex.EncodeToString([]byte{v})
	}
	return strings.Join(parts, ":")
}

// ParseFingerprint accepts the "aa:bb:..." form (colons optional).
func ParseFingerprint(s string) ([32]byte, error) {
	var out [32]byte
	raw, err := hex.DecodeString(strings.ReplaceAll(s, ":", ""))
	if err != nil {
		return out, fmt.Errorf("invalid fingerprint %q: %w", s, err)
	}
	if len(raw) != len(out) {
		return out, fmt.Errorf("invalid fingerprint %q: %d bytes, want %d", s, len(raw), len(out))
	}
	copy(out[:], raw)
	return out, nil
}
