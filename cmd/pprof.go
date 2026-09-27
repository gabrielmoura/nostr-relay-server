//go:build diagnostics

package cmd

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"go.uber.org/zap"
)

const developmentPprofAddress = "127.0.0.1:6060"

type developmentPprofServer struct {
	server   *http.Server
	listener net.Listener
}

func (s *developmentPprofServer) Close() error {
	return s.server.Close()
}

// startDevelopmentPprof serves pprof only for local development. It always
// binds the IPv4 loopback interface and is deliberately unavailable in every
// other environment, including production.
func startDevelopmentPprof(appEnv string, logger *zap.Logger) io.Closer {
	return startDevelopmentPprofAt(appEnv, developmentPprofAddress, logger)
}

func startDevelopmentPprofAt(appEnv, address string, logger *zap.Logger) *developmentPprofServer {
	if appEnv != "development" {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil || !isLoopbackHost(host) {
		logger.Warn("development pprof requires a loopback address", zap.String("address", address), zap.Error(err))
		return nil
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Warn("development pprof is unavailable", zap.String("address", address), zap.Error(err))
		return nil
	}

	server := &http.Server{Handler: developmentPprofMux()}
	pprofServer := &developmentPprofServer{server: server, listener: listener}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Warn("development pprof server stopped unexpectedly", zap.Error(err))
		}
	}()
	logger.Info("development pprof enabled", zap.String("address", address))
	return pprofServer
}

func developmentPprofMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func closeDevelopmentPprof(server io.Closer) {
	if server == nil {
		return
	}
	if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Logger.Warn("failed to close development pprof server", zap.Error(err))
	}
}
