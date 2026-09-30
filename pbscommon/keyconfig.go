package pbscommon

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/scrypt"
)

// KeyConfig is the PBS encryption key file (proxmox-backup:
// pbs-key-config/src/lib.rs), the JSON written by
// `proxmox-backup-client key create`. With a KDF, Data is
// IV(16) || TAG(16) || AES-256-GCM(derived key, raw key); without one
// (KDF == nil, as PVE stores its keys) Data is the raw 32-byte key.
type KeyConfig struct {
	KDF         *KeyDerivation `json:"kdf"`
	Created     string         `json:"created"`
	Modified    string         `json:"modified"`
	Data        []byte         `json:"data"`
	Fingerprint string         `json:"fingerprint,omitempty"`
	Hint        string         `json:"hint,omitempty"`
}

// KeyDerivation mirrors the serde enum {"Scrypt":{...}} / {"PBKDF2":{...}}.
type KeyDerivation struct {
	Scrypt *ScryptParams `json:"Scrypt,omitempty"`
	PBKDF2 *PBKDF2Params `json:"PBKDF2,omitempty"`
}

// ScryptParams are the scrypt settings of a passphrase-protected key.
type ScryptParams struct {
	N    uint64 `json:"n"`
	R    uint64 `json:"r"`
	P    uint64 `json:"p"`
	Salt []byte `json:"salt"`
}

// PBKDF2Params are the PBKDF2-HMAC-SHA256 settings of a passphrase-protected key.
type PBKDF2Params struct {
	Iter int    `json:"iter"`
	Salt []byte `json:"salt"`
}

// MinPassphraseLen is the shortest passphrase PBS accepts.
const MinPassphraseLen = 5

// ErrKeyNeedsPassphrase is returned by KeyConfig.Decrypt when the key file is
// passphrase-protected and no passphrase was given.
var ErrKeyNeedsPassphrase = errors.New("encryption key is protected by a passphrase")

// Upper bounds on KDF parameters read from a key file, so a crafted file
// cannot make us allocate or spin without limit. PBS itself writes
// scrypt n=65536 r=8 p=1 and PBKDF2 65535 rounds.
const (
	maxScryptN    = 1 << 20
	maxScryptR    = 32
	maxScryptP    = 16
	maxScryptMem  = 1025 << 20 // same memory cap as PBS
	maxPBKDF2Iter = 10_000_000
)

func (k *KeyDerivation) deriveKey(passphrase []byte) ([]byte, error) {
	switch {
	case k.Scrypt != nil:
		s := k.Scrypt
		if s.N < 2 || s.N > maxScryptN || s.N&(s.N-1) != 0 || s.R == 0 || s.R > maxScryptR || s.P == 0 || s.P > maxScryptP ||
			128*s.N*s.R > maxScryptMem {
			return nil, fmt.Errorf("unsupported scrypt parameters n=%d r=%d p=%d", s.N, s.R, s.P)
		}
		return scrypt.Key(passphrase, s.Salt, int(s.N), int(s.R), int(s.P), KeySize)
	case k.PBKDF2 != nil:
		if k.PBKDF2.Iter <= 0 || k.PBKDF2.Iter > maxPBKDF2Iter {
			return nil, fmt.Errorf("unsupported PBKDF2 iteration count %d", k.PBKDF2.Iter)
		}
		return pbkdf2.Key(sha256.New, string(passphrase), k.PBKDF2.Salt, k.PBKDF2.Iter, KeySize)
	default:
		return nil, errors.New("unknown key derivation function")
	}
}

// GenerateKey returns a new random encryption key.
func GenerateKey() ([KeySize]byte, error) {
	var key [KeySize]byte
	_, err := rand.Read(key[:])
	return key, err
}

func rfc3339Now() string {
	// PBS parses RFC 3339 without fractional seconds (Z or ±hh:mm).
	return time.Now().Truncate(time.Second).Format(time.RFC3339)
}

// NewKeyConfig wraps key in a key file. With an empty passphrase the key is
// stored in clear (kdf: null, like PVE does); otherwise it is encrypted with a
// scrypt-derived key using the same parameters as proxmox-backup-client.
func NewKeyConfig(key [KeySize]byte, passphrase []byte, hint string) (*KeyConfig, error) {
	cc, err := NewCryptConfig(key)
	if err != nil {
		return nil, err
	}
	now := rfc3339Now()
	kc := &KeyConfig{
		Created:     now,
		Modified:    now,
		Fingerprint: cc.FingerprintString(),
		Hint:        hint,
	}
	if len(passphrase) == 0 {
		kc.Data = append([]byte(nil), key[:]...)
		return kc, nil
	}
	if len(passphrase) < MinPassphraseLen {
		return nil, fmt.Errorf("passphrase is too short (minimum %d characters)", MinPassphraseLen)
	}

	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	kc.KDF = &KeyDerivation{Scrypt: &ScryptParams{N: 65536, R: 8, P: 1, Salt: salt}}
	derived, err := kc.KDF.deriveKey(passphrase)
	if err != nil {
		return nil, err
	}
	aead, err := keyAEAD(derived)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, blobIVSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	sealed := aead.Seal(nil, iv, key[:], nil) // ciphertext || tag
	ct, tag := sealed[:KeySize], sealed[KeySize:]
	kc.Data = make([]byte, 0, blobIVSize+blobTagSize+KeySize)
	kc.Data = append(append(append(kc.Data, iv...), tag...), ct...)
	return kc, nil
}

func keyAEAD(derived []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(derived)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithNonceSize(block, blobIVSize)
}

// ParseKeyConfig parses a key file's JSON.
func ParseKeyConfig(data []byte) (*KeyConfig, error) {
	var kc KeyConfig
	if err := json.Unmarshal(data, &kc); err != nil {
		return nil, fmt.Errorf("invalid encryption key file: %w", err)
	}
	if len(kc.Data) == 0 {
		return nil, errors.New("invalid encryption key file: no key data")
	}
	return &kc, nil
}

// LoadKeyConfig reads and parses a key file.
func LoadKeyConfig(path string) (*KeyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseKeyConfig(data)
}

// NeedsPassphrase reports whether Decrypt needs a passphrase.
func (kc *KeyConfig) NeedsPassphrase() bool { return kc.KDF != nil }

// Decrypt unlocks the key (passphrase is ignored for unprotected key files)
// and checks it against the stored fingerprint.
func (kc *KeyConfig) Decrypt(passphrase []byte) ([KeySize]byte, error) {
	var key [KeySize]byte
	if kc.KDF == nil {
		if len(kc.Data) != KeySize {
			return key, fmt.Errorf("invalid key length %d", len(kc.Data))
		}
		copy(key[:], kc.Data)
	} else {
		if len(passphrase) == 0 {
			return key, ErrKeyNeedsPassphrase
		}
		if len(passphrase) < MinPassphraseLen {
			return key, errors.New("passphrase is too short")
		}
		if len(kc.Data) != blobIVSize+blobTagSize+KeySize {
			return key, fmt.Errorf("invalid encrypted key length %d", len(kc.Data))
		}
		derived, err := kc.KDF.deriveKey(passphrase)
		if err != nil {
			return key, err
		}
		aead, err := keyAEAD(derived)
		if err != nil {
			return key, err
		}
		iv := kc.Data[:blobIVSize]
		tag := kc.Data[blobIVSize : blobIVSize+blobTagSize]
		ct := kc.Data[blobIVSize+blobTagSize:]
		plain, err := aead.Open(nil, iv, append(append([]byte(nil), ct...), tag...), nil)
		if err != nil {
			if kc.Hint != "" {
				return key, fmt.Errorf("unable to decrypt key (password hint: %s)", kc.Hint)
			}
			return key, errors.New("unable to decrypt key (wrong passphrase?)")
		}
		copy(key[:], plain)
	}

	if kc.Fingerprint != "" {
		want, err := ParseFingerprint(kc.Fingerprint)
		if err != nil {
			return key, err
		}
		cc, err := NewCryptConfig(key)
		if err != nil {
			return key, err
		}
		if cc.Fingerprint() != want {
			return key, fmt.Errorf("key file fingerprint %s does not match the contained key %s", kc.Fingerprint, cc.FingerprintString())
		}
	}
	return key, nil
}

// Marshal serializes the key file as PBS writes it.
func (kc *KeyConfig) Marshal() ([]byte, error) {
	return json.MarshalIndent(kc, "", "  ")
}

// LoadCryptConfig is the one-call helper for backup/restore jobs: read the key
// file at path, unlock it and return a ready CryptConfig.
func LoadCryptConfig(path string, passphrase []byte) (*CryptConfig, error) {
	kc, err := LoadKeyConfig(path)
	if err != nil {
		return nil, fmt.Errorf("load encryption key %s: %w", path, err)
	}
	key, err := kc.Decrypt(passphrase)
	if err != nil {
		return nil, fmt.Errorf("unlock encryption key %s: %w", path, err)
	}
	return NewCryptConfig(key)
}
