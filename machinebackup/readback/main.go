// Command readback is an end-to-end test helper (testing/e2e): it reads a
// fixed-index image (.fidx) of the latest snapshot of a backup group back
// through pbscommon, decrypting with -keyfile, and writes the image to -out.
// It checks the reverse interop direction: snapshots written by the official
// proxmox-backup-client (compressed + encrypted chunks by default) must be
// readable by our code, as they are by the NBD server and the GUI restore.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"pbscommon"
)

func main() {
	baseURL := flag.String("baseurl", "", "PBS base URL, e.g. https://127.0.0.1:8007")
	fingerprint := flag.String("certfingerprint", "", "PBS certificate fingerprint")
	authID := flag.String("authid", "", "API token ID")
	secret := flag.String("secret", "", "API token secret")
	datastore := flag.String("datastore", "", "Datastore")
	keyFile := flag.String("keyfile", "", "Encryption key file (unprotected)")
	backupType := flag.String("type", "host", "Backup type")
	backupID := flag.String("id", "", "Backup ID")
	archive := flag.String("archive", "", "Fixed index archive name, e.g. disk.img.fidx")
	out := flag.String("out", "", "Output image file")
	flag.Parse()

	if err := run(*baseURL, *fingerprint, *authID, *secret, *datastore, *keyFile, *backupType, *backupID, *archive, *out); err != nil {
		fmt.Fprintln(os.Stderr, "readback:", err)
		os.Exit(1)
	}
}

func run(baseURL, fingerprint, authID, secret, datastore, keyFile, backupType, backupID, archive, out string) error {
	client := &pbscommon.PBSClient{
		BaseURL:         baseURL,
		CertFingerPrint: fingerprint,
		AuthID:          authID,
		Secret:          secret,
		Datastore:       datastore,
		Insecure:        true,
	}
	if keyFile != "" {
		crypt, err := pbscommon.LoadKeyFile(keyFile, nil)
		if err != nil {
			return err
		}
		client.Crypt = crypt
	}

	snaps, err := client.ListSnapshots()
	if err != nil {
		return fmt.Errorf("list snapshots: %w", err)
	}
	var latest *pbscommon.BackupManifest
	for i := range snaps {
		s := &snaps[i]
		if s.BackupType == backupType && s.BackupID == backupID && (latest == nil || s.BackupTime > latest.BackupTime) {
			latest = s
		}
	}
	if latest == nil {
		return fmt.Errorf("no snapshot %s/%s", backupType, backupID)
	}

	client.Manifest.BackupType = backupType
	client.Manifest.BackupID = backupID
	client.Manifest.BackupTime = latest.BackupTime
	client.Connect(true, backupType)
	defer client.Close()

	index, err := client.DownloadToBytes(archive)
	if err != nil {
		return fmt.Errorf("download %s: %w", archive, err)
	}
	var hdr pbscommon.FIDXHeader
	rdr := bytes.NewReader(index)
	if err := binary.Read(rdr, binary.LittleEndian, &hdr); err != nil {
		return fmt.Errorf("fidx header: %w", err)
	}
	if !bytes.Equal(hdr.Magic[:], []byte{47, 127, 65, 237, 145, 253, 15, 205}) || hdr.ChunkSize == 0 {
		return fmt.Errorf("%s is not a fixed index", archive)
	}
	count := (hdr.Size + hdr.ChunkSize - 1) / hdr.ChunkSize

	f, err := os.Create(out) // #nosec G304 -- test helper, path from the e2e script
	if err != nil {
		return err
	}
	defer f.Close()
	written := uint64(0)
	digest := make([]byte, 32)
	for i := uint64(0); i < count; i++ {
		if _, err := rdr.Read(digest); err != nil {
			return fmt.Errorf("fidx digest %d: %w", i, err)
		}
		data, err := client.GetChunkData(hex.EncodeToString(digest))
		if err != nil {
			return fmt.Errorf("chunk %d: %w", i, err)
		}
		if rest := hdr.Size - written; uint64(len(data)) > rest {
			data = data[:rest]
		}
		if _, err := f.Write(data); err != nil {
			return err
		}
		written += uint64(len(data))
	}
	if written != hdr.Size {
		return fmt.Errorf("wrote %d bytes, index says %d", written, hdr.Size)
	}
	fmt.Printf("readback: %s/%s %s -> %s (%d bytes, %d chunks)\n", backupType, backupID, archive, out, written, count)
	return f.Close()
}
