package privacy

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	goi2pconfig "github.com/go-i2p/go-i2p/lib/config"
	"github.com/go-i2p/go-sam-bridge/lib/embedding"
	"go.uber.org/zap"
)

// nativeI2P owns only the public embedding lifecycle. The fork owns the
// embedded router, its I2CP client/provider, and the SAM bridge internals.
type nativeI2P struct{ bridge *embedding.Bridge }

func (s *i2pService) startNative(ctx context.Context, relayPort int) (*nativeI2P, error) {
	host := s.cfg.SAMHost
	if host == "" {
		host = "127.0.0.1"
	}
	if !isLoopbackI2PHost(host) {
		return nil, fmt.Errorf("i2p native: sam_host must be loopback, got %q", host)
	}
	samPort := s.cfg.SAMPort
	if samPort == 0 {
		samPort = 7656
	}
	i2cpPort := s.cfg.I2CPPort
	if i2cpPort == 0 {
		i2cpPort = 7654
	}
	dataDir := s.cfg.DataDir
	if dataDir == "" {
		dataDir = "./data/i2p"
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("i2p native: create data_dir: %w", err)
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(samPort)))
	if err != nil {
		return nil, fmt.Errorf("i2p native: bind SAM listener: %w", err)
	}
	closeListener := true
	defer func() {
		if closeListener {
			_ = listener.Close()
		}
	}()

	routerCfg := goi2pconfig.DefaultRouterConfig()
	routerCfg.BaseDir = filepath.Join(dataDir, "base")
	routerCfg.WorkingDir = filepath.Join(dataDir, "router")
	routerCfg.NetDB.Path = filepath.Join(dataDir, "netdb")
	routerCfg.I2CP.Enabled = true
	routerCfg.I2CP.Address = net.JoinHostPort(host, strconv.Itoa(i2cpPort))
	if err := os.MkdirAll(routerCfg.WorkingDir, 0o700); err != nil {
		return nil, fmt.Errorf("i2p native: create router state: %w", err)
	}
	if err := os.MkdirAll(routerCfg.NetDB.Path, 0o700); err != nil {
		return nil, fmt.Errorf("i2p native: create netdb state: %w", err)
	}

	bridge, err := embedding.New(
		embedding.WithListener(listener),
		embedding.WithI2CPAddr(routerCfg.I2CP.Address),
		embedding.WithRouterConfig(routerCfg),
		embedding.WithDatagramPort(0),
	)
	if err != nil {
		return nil, fmt.Errorf("i2p native: create SAM bridge: %w", err)
	}
	if err := bridge.Start(ctx); err != nil {
		return nil, fmt.Errorf("i2p native: start SAM bridge: %w", err)
	}
	closeListener = false

	if err := s.startNativeSAM(host, samPort, relayPort); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = bridge.Stop(shutdownCtx)
		return nil, err
	}
	return &nativeI2P{bridge: bridge}, nil
}

func (s *i2pService) startNativeSAM(host string, samPort, relayPort int) error {
	var persisted string
	if s.store != nil {
		var err error
		persisted, err = loadOrCreateString(s.store, "i2p.key")
		if err != nil {
			return err
		}
	}
	client := newSAMClient(host, samPort, s.cfg.SessionName, persisted)
	if err := client.connect(2 * time.Minute); err != nil {
		return err
	}
	if err := client.startForward("127.0.0.1", relayPort, 30*time.Second); err != nil {
		_ = client.Close()
		return err
	}
	if client.B32Address() == "" {
		_ = client.Close()
		return fmt.Errorf("could not derive .b32.i2p address")
	}
	if s.store != nil && persisted == "" && client.Destination() != "" {
		if err := s.store.Save("i2p.key", []byte(client.Destination())); err != nil {
			s.logger.Warn("i2p: failed to persist destination", zap.Error(err))
		}
	}
	s.sam, s.address, s.forward = client, client.B32Address(), true
	s.target = net.JoinHostPort("127.0.0.1", strconv.Itoa(relayPort))
	return nil
}

func (n *nativeI2P) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return n.bridge.Stop(ctx)
}

func isLoopbackI2PHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
