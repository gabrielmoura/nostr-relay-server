package privacy

import (
	"bufio"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"errors"
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
	reader      *bufio.Reader
	forwardConn net.Conn
	dial        func(network, address string, timeout time.Duration) (net.Conn, error)
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
	conn, err := c.dialSAM(timeout)
	if err != nil {
		return fmt.Errorf("sam connect %s: %w", c.addr, err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)

	if err := c.handshake(timeout); err != nil {
		_ = conn.Close()
		c.conn = nil
		return err
	}
	if err := c.createSession(timeout); err != nil {
		_ = conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *samClient) handshake(timeout time.Duration) error {
	reply, err := c.request("HELLO VERSION MIN=3.2 MAX=3.3", timeout)
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
	reply, err := c.request(cmd, timeout)
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

// startForward asks the router to bridge inbound I2P STREAM connections to the
// relay's loopback TCP listener. SILENT=true is required: SAM must proxy the
// stream unchanged, without sending the peer destination before relay bytes.
func (c *samClient) startForward(host string, port int, timeout time.Duration) error {
	if host == "" || port <= 0 || port > 65535 {
		return fmt.Errorf("sam stream forward: invalid target %q:%d", host, port)
	}

	// SAM v3 requires FORWARD on a second control connection. The original
	// connection owns the session and the forwarding connection keeps the
	// mapping alive; closing it tells conforming routers to stop listening.
	conn, err := c.dialSAM(timeout)
	if err != nil {
		return fmt.Errorf("sam forward connect %s: %w", c.addr, err)
	}
	reader := bufio.NewReader(conn)
	reply, err := samRequest(conn, reader, "HELLO VERSION MIN=3.2 MAX=3.3", timeout)
	if err != nil || !strings.HasPrefix(reply, "HELLO REPLY RESULT=OK") {
		_ = conn.Close()
		if err != nil {
			return fmt.Errorf("sam forward handshake: %w", err)
		}
		return fmt.Errorf("sam forward handshake failed: %s", reply)
	}

	cmd := fmt.Sprintf(
		"STREAM FORWARD ID=%s HOST=%s PORT=%d SILENT=true",
		c.sessionName,
		host,
		port,
	)
	reply, err = samRequest(conn, reader, cmd, timeout)
	if err != nil {
		_ = conn.Close()
		return err
	}
	if !strings.HasPrefix(reply, "STREAM STATUS RESULT=OK") {
		_ = conn.Close()
		return fmt.Errorf("sam stream forward failed: %s", reply)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		_ = conn.Close()
		return net.ErrClosed
	}
	if c.forwardConn != nil {
		_ = c.forwardConn.Close()
	}
	c.forwardConn = conn
	return nil
}

func (c *samClient) dialSAM(timeout time.Duration) (net.Conn, error) {
	if c.dial != nil {
		return c.dial("tcp", c.addr, timeout)
	}
	d := net.Dialer{Timeout: timeout}
	return d.Dial("tcp", c.addr)
}

// Destination returns the base64 destination blob for this session. When it was
// created from a persisted identity this is the reusable value; otherwise it is
// the router-generated destination that can be persisted for reuse.
func (c *samClient) Destination() string { return c.destination }

func (c *samClient) request(line string, timeout time.Duration) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil || c.reader == nil {
		return "", net.ErrClosed
	}
	return samRequest(c.conn, c.reader, line, timeout)
}

func samRequest(conn net.Conn, reader *bufio.Reader, line string, timeout time.Duration) (string, error) {
	if _, err := fmt.Fprintf(conn, "%s\n", line); err != nil {
		return "", err
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	reply, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(reply), nil
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
	var closeErr error
	if c.conn != nil {
		closeErr = errors.Join(closeErr, c.conn.Close())
	}
	if c.forwardConn != nil {
		closeErr = errors.Join(closeErr, c.forwardConn.Close())
	}
	return closeErr
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

// b32FromDestination computes the .b32.i2p base-address from a base64 I2P
// destination: SHA-256 of its public destination portion, base32-encoded and
// lowercased. SAM SESSION STATUS may append private keys; they are never part
// of a b32 address.
func b32FromDestination(destB64 string) string {
	raw, err := base64.StdEncoding.DecodeString(destB64)
	if err != nil || len(raw) < 387 {
		return ""
	}
	// A Destination starts with 256-byte crypto and 128-byte signing public
	// keys, followed by a 3-byte certificate. The certificate length extends
	// only the public destination (for non-DSA signing types).
	certLen := int(raw[385])<<8 | int(raw[386])
	destinationLen := 387 + certLen
	if destinationLen > len(raw) {
		return ""
	}
	sum := sha256.Sum256(raw[:destinationLen])
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	return strings.ToLower(s[:52]) + ".b32.i2p"
}
