package pbscommon

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"

	"github.com/klauspost/compress/zstd"
)

// PBS data blob format (proxmox-backup: pbs-datastore/src/file_formats.rs),
// used for chunks and for the stand-alone *.blob files:
//
//	plain:     MAGIC(8) || CRC32(4) || data
//	encrypted: MAGIC(8) || CRC32(4) || IV(16) || TAG(16) || AES-256-GCM(data)
//
// The CRC32 (IEEE, little endian) covers everything after the header, i.e.
// the ciphertext for encrypted blobs. Compression (zstd) happens before
// encryption and is only kept when it actually shrinks the data.
var (
	blobUncompressedMagic = []byte{66, 171, 56, 7, 190, 131, 112, 161}
	blobCompressedMagic   = []byte{49, 185, 88, 66, 111, 182, 163, 127}
	blobEncryptedMagic    = []byte{123, 103, 133, 190, 34, 45, 76, 240}
	blobEncrComprMagic    = []byte{230, 89, 27, 191, 11, 191, 216, 11}
)

const (
	blobHeaderSize          = 12
	blobIVSize              = 16
	blobTagSize             = 16
	encryptedBlobHeaderSize = blobHeaderSize + blobIVSize + blobTagSize
)

// ErrEncryptedNoKey is returned when an encrypted blob or chunk is read
// without an encryption key.
var ErrEncryptedNoKey = errors.New("backup is encrypted: an encryption key is required")

func zstdEncoderLevel(level CompressionLevel) zstd.EncoderLevel {
	switch level {
	case CompressionFastest:
		return zstd.SpeedFastest
	case CompressionBetter:
		return zstd.SpeedBetterCompression
	case CompressionBest:
		return zstd.SpeedBestCompression
	default: // CompressionDefault or empty
		return zstd.SpeedDefault
	}
}

// EncodeBlob builds a PBS data blob from data, compressing it with zstd when
// compress is set (and it helps), and encrypting it when crypt is non-nil.
func EncodeBlob(data []byte, crypt *CryptConfig, compress bool, level CompressionLevel) ([]byte, error) {
	payload := data
	compressed := false
	if compress {
		w, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstdEncoderLevel(level)))
		if err != nil {
			return nil, err
		}
		c := w.EncodeAll(data, nil)
		_ = w.Close()
		if len(c) < len(data) {
			payload, compressed = c, true
		}
	}

	if crypt == nil {
		magic := blobUncompressedMagic
		if compressed {
			magic = blobCompressedMagic
		}
		out := make([]byte, 0, blobHeaderSize+len(payload))
		out = append(out, magic...)
		out = binary.LittleEndian.AppendUint32(out, crc32.ChecksumIEEE(payload))
		return append(out, payload...), nil
	}

	var iv [blobIVSize]byte
	if _, err := rand.Read(iv[:]); err != nil {
		return nil, fmt.Errorf("generate IV: %w", err)
	}
	return encryptBlob(payload, compressed, crypt, iv), nil
}

// encryptBlob is EncodeBlob's encrypted path with an explicit IV (tests pin it
// to compare against golden vectors). payload is already compressed if
// compressed is set.
func encryptBlob(payload []byte, compressed bool, crypt *CryptConfig, iv [blobIVSize]byte) []byte {
	magic := blobEncryptedMagic
	if compressed {
		magic = blobEncrComprMagic
	}
	out := make([]byte, encryptedBlobHeaderSize, encryptedBlobHeaderSize+len(payload)+blobTagSize)
	copy(out, magic)
	copy(out[blobHeaderSize:], iv[:])
	// Go appends the tag after the ciphertext; PBS keeps it in the header.
	sealed := crypt.aead.Seal(out, iv[:], payload, nil)
	ctEnd := len(sealed) - blobTagSize
	copy(sealed[blobHeaderSize+blobIVSize:encryptedBlobHeaderSize], sealed[ctEnd:])
	sealed = sealed[:ctEnd]
	binary.LittleEndian.PutUint32(sealed[8:12], crc32.ChecksumIEEE(sealed[encryptedBlobHeaderSize:]))
	return sealed
}

// IsEncryptedBlob reports whether raw starts with an encrypted blob magic.
func IsEncryptedBlob(raw []byte) bool {
	return len(raw) >= 8 && (bytes.Equal(raw[:8], blobEncryptedMagic) || bytes.Equal(raw[:8], blobEncrComprMagic))
}

// DecodeBlob returns the plaintext of a PBS data blob, decrypting it with
// crypt when it is encrypted (ErrEncryptedNoKey if crypt is nil). The GCM tag
// authenticates encrypted blobs; a wrong key or tampered data fails here.
func DecodeBlob(raw []byte, crypt *CryptConfig) ([]byte, error) {
	data, _, err := decodeBlob(raw, crypt)
	return data, err
}

// decodeBlob is DecodeBlob that also reports whether the blob was encrypted,
// which decides how its chunk digest is computed (keyed or plain SHA-256).
func decodeBlob(raw []byte, crypt *CryptConfig) (data []byte, encrypted bool, err error) {
	if len(raw) < blobHeaderSize {
		return nil, false, fmt.Errorf("short blob: %d bytes", len(raw))
	}
	magic := raw[:8]
	switch {
	case bytes.Equal(magic, blobUncompressedMagic):
		return raw[blobHeaderSize:], false, nil
	case bytes.Equal(magic, blobCompressedMagic):
		data, err = zstdDecode(raw[blobHeaderSize:])
		return data, false, err
	case bytes.Equal(magic, blobEncryptedMagic), bytes.Equal(magic, blobEncrComprMagic):
		if crypt == nil {
			return nil, true, ErrEncryptedNoKey
		}
		if len(raw) < encryptedBlobHeaderSize {
			return nil, true, fmt.Errorf("short encrypted blob: %d bytes", len(raw))
		}
		iv := raw[blobHeaderSize : blobHeaderSize+blobIVSize]
		tag := raw[blobHeaderSize+blobIVSize : encryptedBlobHeaderSize]
		ct := raw[encryptedBlobHeaderSize:]
		sealed := make([]byte, 0, len(ct)+blobTagSize)
		sealed = append(append(sealed, ct...), tag...)
		plain, err := crypt.aead.Open(sealed[:0], iv, sealed, nil)
		if err != nil {
			return nil, true, fmt.Errorf("decrypt blob (wrong key or corrupted data): %w", err)
		}
		if bytes.Equal(magic, blobEncrComprMagic) {
			plain, err = zstdDecode(plain)
		}
		return plain, true, err
	default:
		return nil, false, fmt.Errorf("unknown blob magic %v", magic)
	}
}

func zstdDecode(data []byte) ([]byte, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	return dec.DecodeAll(data, nil)
}
