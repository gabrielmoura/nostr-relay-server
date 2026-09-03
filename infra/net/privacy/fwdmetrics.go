package privacy

import (
	"net"
	"sync"
	"sync/atomic"
)

// countedConn wraps a net.Conn and accumulates the number of bytes read from
// and written to the wire. It is used to measure tx/rx traffic on the privacy
// forwarder without depending on library-specific counters (which are not
// uniformly exposed by bine / go-i2p / ratatoskr). Reads/writes still delegate
// to the underlying connection; only the byte counters are incremented.
type countedConn struct {
	net.Conn
	rx *atomic.Int64 // bytes read (from the network into the relay)
	tx *atomic.Int64 // bytes written (from the relay out to the network)
}

func (c *countedConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.rx.Add(int64(n))
	}
	return n, err
}

func (c *countedConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.tx.Add(int64(n))
	}
	return n, err
}

// countedListener wraps a net.Listener and tracks the number of active
// connections as well as the aggregate tx/rx byte counters shared across all of
// the connections it accepts.
type countedListener struct {
	net.Listener
	rx          *atomic.Int64
	tx          *atomic.Int64
	connections atomic.Int64 // currently-accepted, not-yet-closed conns
}

// NewCountedListener returns a listener wrapper that shares the given tx/rx
// counters and tracks active connections. Pass nil counters to allocate fresh
// ones.
func NewCountedListener(l net.Listener, tx, rx *atomic.Int64) *countedListener {
	if tx == nil {
		tx = &atomic.Int64{}
	}
	if rx == nil {
		rx = &atomic.Int64{}
	}
	return &countedListener{Listener: l, rx: rx, tx: tx}
}

// Snapshot returns a copy of the current traffic/connection counters.
func (l *countedListener) Snapshot() (rxBytes, txBytes int64, connections int64) {
	return l.rx.Load(), l.tx.Load(), l.connections.Load()
}

// Accept wraps each accepted connection with byte accounting and tracks the
// active connection count.
func (l *countedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	l.connections.Add(1)
	wrapped := &countedConn{Conn: conn, rx: l.rx, tx: l.tx}
	return &trackedConn{Conn: wrapped, listener: l}, nil
}

// trackedConn decrements the active-connection counter when closed.
type trackedConn struct {
	net.Conn
	listener *countedListener
	once     sync.Once
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.listener.connections.Add(-1) })
	return err
}

// ensure the helper types implement the expected interfaces.
var (
	_ net.Listener = (*countedListener)(nil)
	_ net.Conn     = (*countedConn)(nil)
	_ net.Conn     = (*trackedConn)(nil)
)
