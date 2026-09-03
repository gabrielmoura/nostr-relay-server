package privacy

import (
	"bufio"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// samClient is a minimal SAM v3 (Simple Anonymous Messaging) client. It talks to
// an already-running I2P router (i2pd or Java-I2P) on the SAM API port (default
// 7656). This is the supported, production-default way to expose the relay over
// I2P: it interoperates with stock daemons and avoids depending on go-i2p's
// unstable embedded-router streaming API.
type samClient struct {
	addr        string
	sessionName string
	conn        net.Conn
	mu          sync.Mutex
	destination string // base64 destination (public key)
	b32address  string // .b32.i2p base-address
	persisted   string // base64 destination blob to reuse across runs ("" = transient)
	closed      bool
}

func newSAMClient(host string, port int, sessionName string, persisted string) *samClient {
	if sessionName == "" {
		sessionName = "nostr-relay"
	}
	return &samClient{
		addr:        net.JoinHostPort(host, fmt.Sprintf("%d", port)),
		sessionName: sessionName,
		persisted:   persisted,
	}
}

// connect performs the SAM v3 handshake and creates a transient STREAM session.
// The session is preserved so the same destination remains reachable until Close.
func (c *samClient) connect(timeout time.Duration) error {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", c.addr)
	if err != nil {
		return fmt.Errorf("sam connect %s: %w", c.addr, err)
	}
	c.conn = conn

	if err := c.handshake(timeout); err != nil {
		_ = conn.Close()
		return err
	}
	if err := c.createSession(timeout); err != nil {
		_ = conn.Close()
		return err
	}
	return nil
}

func (c *samClient) handshake(timeout time.Duration) error {
	if err := c.writeLine("HELLO VERSION MIN=3.2 MAX=3.3"); err != nil {
		return err
	}
	reply, err := c.readLine(timeout)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(reply, "HELLO REPLY RESULT=OK") {
		return fmt.Errorf("sam handshake failed: %s", reply)
	}
	return nil
}

func (c *samClient) createSession(timeout time.Duration) error {
	dest := c.persisted
	if dest == "" {
		dest = "TRANSIENT"
	}
	cmd := fmt.Sprintf("SESSION CREATE STYLE=STREAM ID=%s DESTINATION=%s", c.sessionName, dest)
	if err := c.writeLine(cmd); err != nil {
		return err
	}
	reply, err := c.readLine(timeout)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(reply, "SESSION STATUS RESULT=OK") {
		return fmt.Errorf("sam session create failed: %s", reply)
	}
	replyDest := samField(reply, "DESTINATION")
	switch {
	case c.persisted != "":
		// Reusing a persisted identity: the router may echo back the same
		// destination; trust the persisted blob, or the echo if present.
		c.destination = c.persisted
		if replyDest != "" {
			c.destination = replyDest
		}
	case replyDest != "":
		// First (transient) run: capture the router-generated destination so the
		// caller can persist it for reuse.
		c.destination = replyDest
	default:
		return fmt.Errorf("sam session create: missing DESTINATION in %q", reply)
	}
	c.b32address = b32FromDestination(c.destination)
	return nil
}

// Destination returns the base64 destination blob for this session. When it was
// created from a persisted identity this is the reusable value; otherwise it is
// the router-generated destination that can be persisted for reuse.
func (c *samClient) Destination() string { return c.destination }

func (c *samClient) writeLine(line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := fmt.Fprintf(c.conn, "%s\n", line)
	return err
}

func (c *samClient) readLine(timeout time.Duration) (string, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	line, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// B32Address returns the .b32.i2p base-address of this session's destination.
func (c *samClient) B32Address() string { return c.b32address }

func (c *samClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// samField extracts the value of a KEY=VALUE token from a SAM reply.
func samField(reply, key string) string {
	for _, tok := range strings.Fields(reply) {
		if strings.HasPrefix(tok, key+"=") {
			return strings.TrimPrefix(tok, key+"=")
		}
	}
	return ""
}

// b32FromDestination computes the .b32.i2p base-address from a base64 destination:
// SHA-256 of the public key bytes, base32-encoded, lowercased, first 52 chars.
func b32FromDestination(destB64 string) string {
	raw, err := base64.StdEncoding.DecodeString(destB64)
	if err != nil || len(raw) < 32 {
		return ""
	}
	sum := sha256.Sum256(raw[:32])
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	return strings.ToLower(s[:52]) + ".b32.i2p"
}
