package privacy

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
)

func TestI2PExternalStartsSilentForwardAndPublishesOnlyWhileActive(t *testing.T) {
	// A valid null-certificate public destination is 387 bytes. Append private
	// material to ensure b32 derivation does not hash secrets returned by SAM.
	destinationBytes := make([]byte, 387+64)
	destination := base64.StdEncoding.EncodeToString(destinationBytes)
	expectedB32 := b32FromDestination(destination)
	serverErr := make(chan error, 2)
	forwardClosed := make(chan error, 1)
	clientConn, serverConn := net.Pipe()
	forwardClient, forwardServer := net.Pipe()
	client := &samClient{
		sessionName: "relay-test",
		conn:        clientConn,
		reader:      bufio.NewReader(clientConn),
		dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return forwardClient, nil
		},
	}
	t.Cleanup(func() { _ = client.Close() })
	go serveSAMForSessionTest(serverConn, destination, serverErr)
	go serveSAMForForwardTest(forwardServer, 3333, serverErr, forwardClosed)

	if err := client.handshake(time.Second); err != nil {
		t.Fatalf("SAM handshake: %v", err)
	}
	if err := client.createSession(time.Second); err != nil {
		t.Fatalf("SAM session: %v", err)
	}
	if err := client.startForward("127.0.0.1", 3333, time.Second); err != nil {
		t.Fatalf("SAM silent forward: %v", err)
	}
	for range 2 {
		if err := <-serverErr; err != nil {
			t.Fatalf("fake SAM: %v", err)
		}
	}
	if err := client.Close(); err != nil {
		t.Fatalf("close SAM client: %v", err)
	}
	if err := <-forwardClosed; err != nil {
		t.Fatalf("forward socket lifecycle: %v", err)
	}

	service := &i2pService{
		store:   NewKeyStore(t.TempDir()),
		started: true,
		address: expectedB32,
		forward: true,
		target:  "127.0.0.1:3333",
	}
	wantURL := "ws://" + expectedB32
	if got := service.AuthURLs(); !reflect.DeepEqual(got, []string{wantURL}) {
		t.Fatalf("AuthURLs() = %#v, want %#v", got, []string{wantURL})
	}
	if err := service.saveExternalManifest(); err != nil {
		t.Fatalf("save I2P manifest: %v", err)
	}

	manifestData, err := service.store.Load("i2p.json")
	if err != nil {
		t.Fatalf("load I2P manifest: %v", err)
	}
	var manifest i2pManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("decode I2P manifest: %v", err)
	}
	if manifest.Target != "127.0.0.1:3333" || manifest.Forward != "STREAM FORWARD SILENT=true" {
		t.Fatalf("manifest = %#v, want loopback silent forward", manifest)
	}
	if manifest.Mode != "external" || manifest.Implementation != "sam-v3" || manifest.SAMAddress != "127.0.0.1:7656" {
		t.Fatalf("manifest runtime metadata = %#v, want external SAM metadata", manifest)
	}

	if err := service.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := service.AuthURLs(); len(got) != 0 {
		t.Fatalf("AuthURLs() after Close = %#v, want empty", got)
	}
}

func TestI2PNativeRejectsNonLoopbackSAMHost(t *testing.T) {
	service := newI2PService(config.I2PConfig{
		Mode:    "native",
		SAMHost: "192.0.2.10",
	}, zap.NewNop(), nil)
	err := service.Start(context.Background(), 3333)
	if err == nil || !strings.Contains(err.Error(), "sam_host must be loopback") {
		t.Fatalf("Start(native) error = %v, want loopback validation error", err)
	}
}

func TestYggAuthURLsRequireActiveForward(t *testing.T) {
	service := &yggService{
		started: true,
		addr:    &net.TCPAddr{IP: net.ParseIP("200:db8::1"), Port: 9000},
	}
	if got := service.AuthURLs(); len(got) != 0 {
		t.Fatalf("AuthURLs() without forward = %#v, want empty", got)
	}

	service.forward = true
	if got := service.AuthURLs(); !reflect.DeepEqual(got, []string{"ws://[200:db8::1]:9000"}) {
		t.Fatalf("AuthURLs() = %#v, want IPv6 websocket URL", got)
	}

	if err := service.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := service.AuthURLs(); len(got) != 0 {
		t.Fatalf("AuthURLs() after Close = %#v, want empty", got)
	}
}

func serveSAMForSessionTest(conn net.Conn, destination string, result chan<- error) {
	defer conn.Close()

	commands := []struct {
		want  string
		reply string
	}{
		{
			want:  "HELLO VERSION MIN=3.2 MAX=3.3",
			reply: "HELLO REPLY RESULT=OK VERSION=3.3",
		},
		{
			want:  "SESSION CREATE STYLE=STREAM ID=relay-test DESTINATION=TRANSIENT",
			reply: "SESSION STATUS RESULT=OK DESTINATION=" + destination,
		},
	}
	serveSAMCommands(conn, commands, result)
}

func serveSAMForForwardTest(conn net.Conn, relayPort int, result chan<- error, closed chan<- error) {
	defer conn.Close()
	commands := []struct {
		want  string
		reply string
	}{
		{
			want:  "HELLO VERSION MIN=3.2 MAX=3.3",
			reply: "HELLO REPLY RESULT=OK VERSION=3.3",
		},
		{
			want:  fmt.Sprintf("STREAM FORWARD ID=relay-test HOST=127.0.0.1 PORT=%d SILENT=true", relayPort),
			reply: "STREAM STATUS RESULT=OK",
		},
	}
	serveSAMCommands(conn, commands, result)
	// A conforming client keeps this socket open while the forward is active.
	// It must only close when its session is shut down.
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err := bufio.NewReader(conn).ReadByte()
	if err == nil {
		closed <- fmt.Errorf("forward socket received data instead of close")
		return
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		closed <- fmt.Errorf("forward socket was closed before lifecycle verification")
		return
	}
	closed <- nil
}

func serveSAMCommands(conn net.Conn, commands []struct {
	want  string
	reply string
}, result chan<- error) {
	reader := bufio.NewReader(conn)
	for _, command := range commands {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			result <- readErr
			return
		}
		if got := strings.TrimSpace(line); got != command.want {
			result <- fmt.Errorf("command = %q, want %q", got, command.want)
			return
		}
		if _, writeErr := fmt.Fprintln(conn, command.reply); writeErr != nil {
			result <- writeErr
			return
		}
	}
	result <- nil
}

func TestSAMStartForwardRejectsInvalidTarget(t *testing.T) {
	client := newSAMClient("127.0.0.1", 7656, "test", "")
	if err := client.startForward("127.0.0.1", 0, time.Second); err == nil {
		t.Fatal("startForward with port zero succeeded")
	}
}
