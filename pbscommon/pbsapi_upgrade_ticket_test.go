package pbscommon

import (
	"bufio"
	"crypto/tls"
	"net"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeUpgradePBS answers the HTTP/1.1 upgrade that opens a backup session: 101
// for the ticket "fresh" (or the API token tok@pbs!t:good), 401 for anything
// else, as PBS does for a ticket older than 2 hours.
type fakeUpgradePBS struct {
	mu       sync.Mutex
	upgrades int
	refused  int
}

func newFakeUpgradePBS(t *testing.T) (*fakeUpgradePBS, string) {
	t.Helper()
	cert := httptest.NewUnstartedServer(nil)
	cert.StartTLS() // only for its certificate
	tlsCfg := cert.TLS.Clone()
	cert.Close()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	f := &fakeUpgradePBS{}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(c)
		}
	}()
	return f, ln.Addr().String()
}

func (f *fakeUpgradePBS) serve(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	ok := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if line == "Cookie: PBSAuthCookie=fresh" || line == "Authorization: PBSAPIToken=tok@pbs!t:good" {
			ok = true
		}
	}
	f.mu.Lock()
	f.upgrades++
	if !ok {
		f.refused++
	}
	f.mu.Unlock()
	if !ok {
		body := `{"data":null,"message":"authentication failed - invalid credentials"}`
		_, _ = c.Write([]byte("HTTP/1.1 401 Unauthorized\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body))
		return
	}
	_, _ = c.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: proxmox-backup-protocol-v1\r\nConnection: Upgrade\r\n\r\n"))
	_, _ = r.ReadByte() // hold the connection until the client closes it
}

func upgradeClient(addr string) *PBSClient {
	c := ticketClient("https://" + addr)
	c.TLSConfig = tls.Config{InsecureSkipVerify: true} // #nosec G402 -- a test server's own certificate
	return c
}

// A backup that starts again after its ticket expired (Sean's third attempt
// on issue 142 came 4 hours after the first) gets a new ticket and its session.
func TestOpenSessionRenewsARefusedTicket(t *testing.T) {
	f, addr := newFakeUpgradePBS(t)
	c := upgradeClient(addr)
	calls := 0
	c.RenewTicket = func() (string, string, error) { calls++; return "fresh", "csrf2", nil }
	conn, err := c.openSession("tcp", addr, false)
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}
	conn.Close()
	if calls != 1 || f.upgrades != 2 || f.refused != 1 {
		t.Fatalf("renewals %d, upgrades %d, refused %d; want one renewal and one more upgrade", calls, f.upgrades, f.refused)
	}
	if c.Ticket != "fresh" {
		t.Errorf("the client kept ticket %q", c.Ticket)
	}
}

// A ticket that works opens the session at once, with no login.
func TestOpenSessionWithAValidTicketDoesNotRenew(t *testing.T) {
	f, addr := newFakeUpgradePBS(t)
	c := upgradeClient(addr)
	c.Ticket = "fresh"
	c.RenewTicket = func() (string, string, error) { t.Error("renewed a ticket that worked"); return "", "", nil }
	conn, err := c.openSession("tcp", addr, false)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if f.upgrades != 1 {
		t.Fatalf("upgrades %d", f.upgrades)
	}
}

// An API token does not expire: its 401 is the answer, with no renewal.
func TestOpenSessionDoesNotRetryAnAPIToken(t *testing.T) {
	f, addr := newFakeUpgradePBS(t)
	c := upgradeClient(addr)
	c.Ticket, c.CSRFToken = "", ""
	c.AuthID, c.Secret = "tok@pbs!t", "wrong"
	_, err := c.openSession("tcp", addr, false)
	if !ticketRefused(err) {
		t.Fatalf("openSession with a wrong token = %v, want the 401", err)
	}
	if f.upgrades != 1 {
		t.Fatalf("upgrades %d, want 1", f.upgrades)
	}
}

// No way to get a new ticket: the error says why.
func TestOpenSessionWithoutALoginSaysTheTicketExpired(t *testing.T) {
	_, addr := newFakeUpgradePBS(t)
	c := upgradeClient(addr)
	_, err := c.openSession("tcp", addr, false)
	if err == nil || !strings.Contains(err.Error(), "ticket lasts 2 hours") || !strings.Contains(err.Error(), "401") {
		t.Fatalf("openSession = %v, want the 401 and the expired-ticket reason", err)
	}
}

// A new ticket PBS also refuses is not tried again and again.
func TestOpenSessionRetriesOnlyOnce(t *testing.T) {
	f, addr := newFakeUpgradePBS(t)
	c := upgradeClient(addr)
	c.RenewTicket = func() (string, string, error) { return "also-refused", "c", nil }
	_, err := c.openSession("tcp", addr, false)
	if !ticketRefused(err) {
		t.Fatalf("openSession = %v, want the second 401", err)
	}
	if f.upgrades != 2 {
		t.Fatalf("upgrades %d, want 2", f.upgrades)
	}
}
