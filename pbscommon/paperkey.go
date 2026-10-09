package pbscommon

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
	"rsc.io/qr"
)

// Paper key: a printable copy of a key file with a QR code, like
// `proxmox-backup-client key paperkey` (proxmox-backup:
// pbs-datastore/src/paperkey.rs). The QR code holds the key file exactly as
// PBS pretty-prints it (EC level M, like its `qrencode -lm`), so scanning it
// gives a key file that this client and proxmox-backup-client both import.

const (
	paperKeyBegin = "-----BEGIN PROXMOX BACKUP KEY-----"
	paperKeyEnd   = "-----END PROXMOX BACKUP KEY-----"
)

// PaperKey is the printable form of a key file.
type PaperKey struct {
	JSON  string // key file as serde_json::to_string_pretty writes it; the QR payload
	Text  string // JSON between the BEGIN/END markers, as PBS prints it
	QRSVG string // QR code of JSON, as a standalone SVG document
}

// NewPaperKey renders kc as a paper key. Protect kc with a passphrase first
// (ProtectKeyConfig) if the paper may be seen by others: the paper key holds
// exactly what the key file holds.
func NewPaperKey(kc *KeyConfig) (*PaperKey, error) {
	data, err := kc.prettyJSON()
	if err != nil {
		return nil, err
	}
	code, err := qr.Encode(data, qr.M)
	if err != nil {
		return nil, fmt.Errorf("paper key QR code: %w", err)
	}
	return &PaperKey{
		JSON:  data,
		Text:  paperKeyBegin + "\n" + data + "\n" + paperKeyEnd + "\n",
		QRSVG: qrSVG(code, 4),
	}, nil
}

// pbsKeyConfig is KeyConfig laid out as proxmox-backup serializes it (field
// order of pbs-key-config, "kdf": null when the key is not protected).
type pbsKeyConfig struct {
	KDF         *KeyDerivationConfig `json:"kdf"`
	Created     string               `json:"created"`
	Modified    string               `json:"modified"`
	Data        string               `json:"data"`
	Fingerprint string               `json:"fingerprint,omitempty"`
	Hint        string               `json:"hint,omitempty"`
}

// prettyJSON matches serde_json::to_string_pretty of the PBS KeyConfig:
// two-space indent, no trailing newline, and no HTML escaping of <, > and &.
func (c *KeyConfig) prettyJSON() (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(pbsKeyConfig(*c)); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// ProtectKeyConfig wraps the key of crypt in a passphrase-protected key file,
// as `proxmox-backup-client key create` writes it (scrypt n=65536 r=8 p=1,
// data = IV || TAG || AES-256-GCM(derived key, raw key)). Use it to print or
// export a copy that is useless without the passphrase.
func ProtectKeyConfig(crypt *CryptConfig, passphrase []byte, hint string) (*KeyConfig, error) {
	if len(passphrase) < 5 {
		return nil, errors.New("the passphrase must be at least 5 characters long")
	}
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	kdf := &KeyDerivationConfig{KDF: "scrypt", N: 65536, R: 8, P: 1, Salt: base64.StdEncoding.EncodeToString(salt)}
	derived, err := scrypt.Key(passphrase, salt, int(kdf.N), int(kdf.R), int(kdf.P), BlobEncryptionKeySize)
	if err != nil {
		return nil, err
	}
	wrap, err := NewCryptConfig(derived)
	if err != nil {
		return nil, err
	}
	iv, ct, tag, err := wrap.Seal(crypt.encKey)
	if err != nil {
		return nil, err
	}
	data := make([]byte, 0, len(iv)+len(tag)+len(ct))
	data = append(append(append(data, iv...), tag...), ct...)
	now := time.Now().UTC().Format(time.RFC3339)
	return &KeyConfig{
		KDF:         kdf,
		Created:     now,
		Modified:    now,
		Data:        base64.StdEncoding.EncodeToString(data),
		Fingerprint: crypt.Fingerprint(),
		Hint:        hint,
	}, nil
}

// qrSVG draws code with a quiet zone of margin modules (PBS uses none; the QR
// spec asks for 4, which scanners read more reliably). Each horizontal run of
// dark modules is one rectangle of the path.
func qrSVG(code *qr.Code, margin int) string {
	n := code.Size + 2*margin
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`, n, n)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="`, n, n)
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; {
			if !code.Black(x, y) {
				x++
				continue
			}
			start := x
			for x < code.Size && code.Black(x, y) {
				x++
			}
			fmt.Fprintf(&b, "M%d %dh%dv1h-%dz", start+margin, y+margin, x-start, x-start)
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String()
}

// PaperKeyPage holds the caller-supplied (localized) text of a paper key page.
type PaperKeyPage struct {
	Lang    string   // HTML lang attribute; "" = "en"
	Title   string   // page title and heading; "" = "Proxmox Backup Paperkey"
	Subject string   // printed under the title as is (PBS prints "Subject: ..."); "" = none
	Notes   []string // paragraphs printed under the QR code (e.g. restore steps)
}

// HTML returns a standalone printable page: the key text and its QR code on
// one sheet, the layout of `proxmox-backup-client key paperkey --output-format
// html`. All caller text is escaped.
func (p *PaperKey) HTML(page PaperKeyPage) []byte {
	lang := page.Lang
	if lang == "" {
		lang = "en"
	}
	title := page.Title
	if title == "" {
		title = "Proxmox Backup Paperkey"
	}
	esc := html.EscapeString

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n")
	fmt.Fprintf(&b, "<html lang=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n", esc(lang))
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", esc(title))
	b.WriteString(`<style>
  body { font-family: sans-serif; margin: 1.5cm; color: #000; background: #fff; }
  h1 { font-size: 16pt; margin: 0 0 0.5em; }
  p.key { font-size: 11pt; font-family: monospace; white-space: pre-wrap; line-break: anywhere; }
  .sheet { page-break-inside: avoid; break-inside: avoid; }
  .qr { text-align: center; margin: 1em 0; }
  .qr svg { width: 11cm; height: 11cm; }
  p.note { font-size: 10pt; }
</style>
</head>
<body>
<div class="sheet">
`)
	fmt.Fprintf(&b, "<h1>%s</h1>\n", esc(title))
	if page.Subject != "" {
		fmt.Fprintf(&b, "<p>%s</p>\n", esc(page.Subject))
	}
	fmt.Fprintf(&b, "<p class=\"key\">%s</p>\n", esc(strings.TrimSuffix(p.Text, "\n")))
	fmt.Fprintf(&b, "<div class=\"qr\">%s</div>\n", p.QRSVG)
	for _, note := range page.Notes {
		fmt.Fprintf(&b, "<p class=\"note\">%s</p>\n", esc(note))
	}
	b.WriteString("</div>\n</body>\n</html>\n")
	return []byte(b.String())
}

// stripPaperKeyMarkers extracts the key file from text copied off a paper key
// (anything around the BEGIN/END marker lines, such as PBS's "Subject:" line,
// is dropped). Markers count only on a line of their own, so a hint quoting
// them stays intact. Plain JSON is returned unchanged.
func stripPaperKeyMarkers(data []byte) []byte {
	if strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		return data
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != paperKeyBegin {
			continue
		}
		body := lines[i+1:]
		for j, l := range body {
			if strings.TrimSpace(l) == paperKeyEnd {
				body = body[:j]
				break
			}
		}
		return []byte(strings.Join(body, "\n"))
	}
	return data
}
